# frozen_string_literal: true

require 'securerandom'
require 'mysql2'
require 'mysql2-cs-bind'
require 'open3'
require 'openssl'
require 'set'
require 'sinatra/base'
require 'sinatra/cookies'
require 'sinatra/json'

require_relative './admin'

module Isuconquest

  class HttpError < StandardError
    attr_reader :code

    def initialize(code, message)
      super(message)
      @code = code
    end
  end

  class App < Sinatra::Base
    enable :logging
    set :show_exceptions, :after_handler

    DECK_CARD_NUMBER = 3
    PRESENT_COUNT_PER_PAGE = 100

    SQL_DIRECTORY = '../sql/'

    error HttpError do
      e = env['sinatra.error']
      logger.error("status=#{e.code}, err=#{e.inspect}")
      content_type :json
      status e.code
      JSON.dump(status_code: e.code, message: e.message)
    end

    helpers do
      def json_params
        @json_params ||= JSON.parse(request.body.tap(&:rewind).read, symbolize_names: true)
      rescue JSON::ParserError => e
        raise HttpError.new(400, e.inspect)
      end

      def connect_db(batch = false)
        Mysql2::Client.new(
          host: ENV.fetch('ISUCON_DB_HOST', '127.0.0.1'),
          port: ENV.fetch('ISUCON_DB_PORT', '3306').to_i,
          username: ENV.fetch('ISUCON_DB_USER', 'isucon'),
          password: ENV.fetch('ISUCON_DB_PASSWORD', 'isucon'),
          database: ENV.fetch('ISUCON_DB_NAME', 'isucon'),
          database_timezone: :local,
          cast_booleans: true,
          symbolize_keys: true,
          reconnect: true,
          flags: batch ? Mysql2::Client::MULTI_STATEMENTS : nil,
        )
      end

      def db
        Thread.current[:db] ||= connect_db()
      end

      def db_transaction(&block)
        db.query('BEGIN')
        done = false
        retval = block.call

        db.query('COMMIT')
        done = true
        return retval
      ensure
        db.query('ROLLBACK') unless done
      end

      def check_one_time_token!(token, token_type, request_at)
        query = 'SELECT * FROM user_one_time_tokens WHERE token=? AND token_type=? AND deleted_at IS NULL'
        tk = db.xquery(query, token, token_type).first
        raise HttpError.new(400, 'invalid token') unless tk

        if tk.fetch(:expired_at) < request_at
          query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?'
          db.xquery(query, request_at, token)
          raise HttpError.new(400, 'invalid token')
        end

        # 使ったトークンを失効する
        query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?'
        db.xquery(query, request_at, token)
      end

      def check_viewer_id!(user_id, viewer_id)
        query = 'SELECT * FROM user_devices WHERE user_id=? AND platform_id=?'
        device = db.xquery(query, user_id, viewer_id).first
        raise HttpError.new(404, 'not found user device') unless device
      end

      def check_ban?(user_id)
        query = 'SELECT * FROM user_bans WHERE user_id=?'
        ban_user = db.xquery(query, user_id).first
        return false unless ban_user
        true
      end

      def get_request_time
        request.env.fetch('isuconquest.request_time')
      rescue KeyError
        raise HttpError.new(500, 'failed to get request time')
      end

      # ログイン処理
      def login_process(user_id, request_at)
        query = 'SELECT * FROM users WHERE id=?'
        user = db.xquery(query, user_id).first&.then { User.new(_1) }
        raise HttpError.new(404, 'not found user') unless user

        # ログインボーナス処理
        login_bonuses = obtain_login_bonus(user_id, request_at)

        # 全員プレゼント取得
        all_presents = obtain_present(user_id, request_at)

        unless db.xquery('SELECT isu_coin FROM users WHERE id=?', user.id).first
          raise HttpError.new(404, 'not found user')
        end

        user.updated_at = request_at
        user.last_activated_at = request_at

        query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?'
        db.xquery(query, request_at, request_at, user_id)

        [user, login_bonuses, all_presents]
      end

      # ログイン処理が終わっているか
      def complete_today_login?(last_activated_at_unixtime, request_at_unixtime)
        last_activated_at = Time.at(last_activated_at_unixtime, in: "+09:00")
        request_at = Time.at(request_at_unixtime, in: "+09:00")
        last_activated_at.year == request_at.year && 
          last_activated_at.month == request_at.month && 
          last_activated_at.day == request_at.day
      end

      def obtain_login_bonus(user_id, request_at)
        # login bonus masterから有効なログインボーナスを取得
        query = 'SELECT * FROM login_bonus_masters WHERE start_at <= ? AND end_at >= ?'
        login_bonuses = db.xquery(query, request_at, request_at)

        send_login_bonuses = []

        login_bonuses.each do |bonus|
          init_bonus = false
          # ボーナスの進捗取得
          query = 'SELECT * FROM user_login_bonuses WHERE user_id=? AND login_bonus_id=?'
          user_bonus = db.xquery(query, user_id, bonus.fetch(:id)).first&.then { UserLoginBonus.new(_1) }
          unless user_bonus
            init_bonus = true

            user_bonus_id = generate_id()
            user_bonus = UserLoginBonus.new(
              id: user_bonus_id,
              user_id: user_id,
              login_bonus_id: bonus.fetch(:id),
              last_reward_sequence: 0,
              loop_count: 1,
              created_at: request_at,
              updated_at: request_at,
            )
          end

          # ボーナス進捗更新
          if user_bonus.last_reward_sequence < bonus.fetch(:column_count)
            user_bonus.last_reward_sequence += 1
          else
            if bonus.fetch(:looped)
              user_bonus.loop_count += 1
              user_bonus.last_reward_sequence = 1
            else
              # 上限まで付与完了
              next
            end
          end
          user_bonus.updated_at = request_at

          # 今回付与するリソース取得
          query = 'SELECT * FROM login_bonus_reward_masters WHERE login_bonus_id=? AND reward_sequence=?'
          reward_item = db.xquery(query, bonus.fetch(:id), user_bonus.last_reward_sequence).first
          raise HttpError.new(404, 'not found login bonus reward') unless reward_item

          obtain_item(user_id, reward_item.fetch(:item_id), reward_item.fetch(:item_type), reward_item.fetch(:amount), request_at)

          # 進捗の保存
          if init_bonus
            query = 'INSERT INTO user_login_bonuses(id, user_id, login_bonus_id, last_reward_sequence, loop_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
            db.xquery(query, user_bonus.id, user_bonus.user_id, user_bonus.login_bonus_id, user_bonus.last_reward_sequence, user_bonus.loop_count, user_bonus.created_at, user_bonus.updated_at)
          else
            query = 'UPDATE user_login_bonuses SET last_reward_sequence=?, loop_count=?, updated_at=? WHERE id=?'
            db.xquery(query, user_bonus.last_reward_sequence, user_bonus.loop_count, user_bonus.updated_at, user_bonus.id)
          end

          send_login_bonuses.push(user_bonus)
        end

        send_login_bonuses
      end

      # プレゼント付与処理
      def obtain_present(user_id, request_at)
        normal_presents = db.xquery('SELECT * FROM present_all_masters WHERE registered_start_at <= ? AND registered_end_at >= ?', request_at, request_at)
        obtain_presents = []
        normal_presents.each do |normal_present_|
          normal_present = PresentAllMaster.new(normal_present_)

          query = 'SELECT * FROM user_present_all_received_history WHERE user_id=? AND present_all_id=?'
          user_present_all_received_history = db.xquery(query, user_id, normal_present.id).first
          next if user_present_all_received_history # プレゼント配布済

          # user present boxに入れる
          user_present_id = generate_id()
          user_present = UserPresent.new(
            id: user_present_id,
            user_id: user_id,
            sent_at: request_at,
            item_type: normal_present.item_type,
            item_id: normal_present.item_id,
            amount: normal_present.amount,
            present_message: normal_present.present_message,
            created_at: request_at,
            updated_at: request_at,
          )
          query = 'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)'
          db.xquery(query, user_present.id, user_present.user_id, user_present.sent_at, user_present.item_type, user_present.item_id, user_present.amount, user_present.present_message, user_present.created_at, user_present.updated_at)

          # historyに入れる
          present_history_id = generate_id()
          history = UserPresentAllReceivedHistory.new(
            id: present_history_id,
            user_id: user_id,
            present_all_id: normal_present.id,
            received_at: request_at,
            created_at: request_at,
            updated_at: request_at,
          )
          query = 'INSERT INTO user_present_all_received_history(id, user_id, present_all_id, received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)'
          db.xquery(
            query,
            history.id,
            history.user_id,
            history.present_all_id,
            history.received_at,
            history.created_at,
            history.updated_at,
          )

          obtain_presents.push(user_present)
        end

        obtain_presents
      end

      # アイテム付与処理
      def obtain_item(user_id, item_id, item_type, obtain_amount, request_at)
        obtain_coins = []
        obtain_cards = []
        obtain_items = []

        case item_type
        when 1 # coin
          query = 'SELECT * FROM users WHERE id=?'
          user = db.xquery(query, user_id).first
          raise HttpError.new(404, 'not found user') unless user

          query = 'UPDATE users SET isu_coin=? WHERE id=?'
          total_coin = user.fetch(:isu_coin) + obtain_amount
          db.xquery(query, total_coin, user_id)

          obtain_coins.push(obtain_amount)

        when 2 # card(ハンマー)
          query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?'
          item = db.xquery(query, item_id, item_type).first
          raise HttpError.new(404, 'not found item') unless item

          user_card_id = generate_id()
          card = UserCard.new(
            id: user_card_id,
            user_id:,
            card_id: item.fetch(:id),
            amount_per_sec: item.fetch(:amount_per_sec),
            level: 1,
            total_exp: 0,
            created_at: request_at,
            updated_at: request_at,
          )
          query = 'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)'
          db.xquery(query, card.id, card.user_id, card.card_id, card.amount_per_sec, card.level, card.total_exp, card.created_at, card.updated_at)

          obtain_cards.push(card)

        when 3, 4 # 強化素材
          query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?'
          item = db.xquery(query, item_id, item_type).first
          raise HttpError.new(404, 'not found item') unless item

          # 所持数取得
          query = 'SELECT * FROM user_items WHERE user_id=? AND item_id=?'
          user_item = db.xquery(query, user_id, item.fetch(:id)).first&.then { UserItem.new(_1) }
          if user_item.nil? # 新規作成
            user_item_id = generate_id()
            user_item = UserItem.new(
              id: user_item_id,
              user_id: user_id,
              item_type: item.fetch(:item_type),
              item_id: item.fetch(:id),
              amount: obtain_amount,
              created_at: request_at,
              updated_at: request_at,
            )
            query = "INSERT INTO user_items(id, user_id, item_id, item_type, amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
            db.xquery(query, user_item.id, user_id, user_item.item_id, user_item.item_type, user_item.amount, request_at, request_at)
          else # 更新
            user_item.amount += obtain_amount
            user_item.updated_at = request_at
            query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?'
            db.xquery(query, user_item.amount, user_item.updated_at, user_item.id)
          end

          obtain_items.push(user_item)

        else
          raise HttpError.new(400, 'invalid item type')
        end

        [obtain_coins, obtain_cards, obtain_items]
      end

      def generate_id
        db = Thread.current[:generate_id_db] ||= connect_db
        update_error = nil
        100.times do
          begin
            db.query('UPDATE id_generator SET id=LAST_INSERT_ID(id+1)')
          rescue Mysql2::Error => e
            if e.error_number == 1213
              update_error = e
              next
            end
            raise e
          end

          return db.last_id
        end

        raise "failed to generate id: #{update_error.inspect}"
      end

      def generate_uuid
        SecureRandom.uuid
      end

      def get_user_id
        Integer(params[:user_id], 10)
      rescue ArgumentError => e
        raise HttpError.new(400, e.inspect)
      end
    end

    # adminMiddleware
    before %r{/admin/.*} do # /admin/*
      request.env['isuconquest.request_time'] = Time.now.to_i
    end

    # apiMiddleware
    before %r{/(?:user(?:/.*|)|login)} do # /user, /user/*, /login
      request.env['isuconquest.request_time'] = (request.get_header('HTTP_X_ISU_DATE')&.then do |header|
        Time.rfc822(header)
      rescue ArgumentError
        nil
      end || Time.now).to_i

      # マスタ確認
      master_version = db.query('SELECT * FROM version_masters WHERE status=1').first
      unless master_version
        raise HttpError.new(404, 'active master version is not found')
      end

      if master_version.fetch(:master_version) != request.get_header('HTTP_X_MASTER_VERSION')
        raise HttpError.new(422, 'invalid master version')
      end

      # check ban
      is_ban = begin
        user_id = get_user_id()
        check_ban?(user_id)
      rescue HttpError
        # user_id is not available, do nothing
      end
      raise HttpError.new(403, 'forbidden') if is_ban
    end

    set(:check_session) do |_bool|
      condition do
        sess_id = request.get_header('HTTP_X_SESSION')
        raise HttpError.new(401, 'unauthorized user') if sess_id.nil? || sess_id.empty?

        user_id = get_user_id()

        request_at = get_request_time()

        query = 'SELECT * FROM user_sessions WHERE session_id=? AND deleted_at IS NULL'
        user_session = db.xquery(query, sess_id).first
        raise HttpError.new(401, 'unauthorized user') if user_session.nil?

        if user_session.fetch(:user_id) != user_id
          raise HttpError.new(403, 'forbidden')
        end

        if user_session.fetch(:expired_at) < request_at
          query = 'UPDATE user_sessions SET deleted_at=? WHERE session_id=?'
          db.xquery(query, request_at, sess_id)
          raise HttpError.new(401, 'session expired')
        end
      end
    end

    post '/initialize' do
      connect_db(true)

      out, status = Open3.capture2e("/bin/sh", "-c", "#{SQL_DIRECTORY}init.sh")
      unless status.success?
        raise HttpError.new(500, "Failed to initialize: #{out}")
      end

      json(
        language: 'ruby',
      )
    end

    get '/health' do
      content_type :text
      'OK'
    end

    # create_user ユーザの作成
    post '/user' do
      if !json_params[:viewerId].is_a?(String) || json_params[:viewerId].empty? || !json_params[:platformType].is_a?(Integer) || json_params[:platformType] < 1 || json_params[:platformType] > 3
        raise HttpError.new(400, 'invalid request body')
      end

      request_at = get_request_time()

      db_transaction do
        user_id = generate_id()
        user = User.new(
          id: user_id,
          isu_coin: 0,
          last_getreward_at: request_at,
          last_activated_at: request_at,
          registered_at: request_at,
          created_at: request_at,
          updated_at: request_at,
        )

        query = 'INSERT INTO users(id, last_activated_at, registered_at, last_getreward_at, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?)'
        db.xquery(query, user.id, user.last_activated_at, user.registered_at, user.last_getreward_at, user.created_at, user.updated_at)

        user_device_id = generate_id()
        user_device = UserDevice.new(
          id: user_device_id,
          user_id: user.id,
          platform_id: json_params.fetch(:viewerId),
          platform_type: json_params.fetch(:platformType),
          created_at: request_at,
          updated_at: request_at,
        )
        query = 'INSERT INTO user_devices(id, user_id, platform_id, platform_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)'
        db.xquery(query, user_device.id, user.id, json_params.fetch(:viewerId), json_params.fetch(:platformType), request_at, request_at)

        # 初期デッキ付与
        query = 'SELECT * FROM item_masters WHERE id=?'
        init_card = db.xquery(query, 2).first
        raise HttpError.new(404, 'not found item') unless init_card

        init_cards = 3.times.map do
          card_id = generate_id()
          card = UserCard.new(
            id: card_id,
            user_id: user.id,
            card_id: init_card.fetch(:id),
            amount_per_sec: init_card.fetch(:amount_per_sec),
            level: 1,
            total_exp: 0,
            created_at: request_at,
            updated_at: request_at,
          )
          query = 'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)'
          db.xquery(query, card.id, card.user_id, card.card_id, card.amount_per_sec, card.level, card.total_exp, card.created_at, card.updated_at)

          card
        end

        deck_id = generate_id()
        init_deck = UserDeck.new(
          id: deck_id,
          user_id: user.id,
          user_card_id_1: init_cards.fetch(0).id,
          user_card_id_2: init_cards.fetch(1).id,
          user_card_id_3: init_cards.fetch(2).id,
          created_at: request_at,
          updated_at: request_at,
        )
        query = 'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
        db.xquery(query, init_deck.id, init_deck.user_id, init_deck.user_card_id_1, init_deck.user_card_id_2, init_deck.user_card_id_3, init_deck.created_at, init_deck.updated_at)

        # ログイン処理
        user, login_bonuses, presents = login_process(user_id, request_at)

        # generate session
        session_id = generate_id()
        sess_id = generate_uuid()
        sess = Session.new(
          id: session_id,
          user_id: user.id,
          session_id: sess_id,
          created_at: request_at,
          updated_at: request_at,
          expired_at: request_at + 86400,
        )
        query = 'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
        db.xquery(query, sess.id, sess.user_id, sess.session_id, sess.created_at, sess.updated_at, sess.expired_at)

        json(
          userId: user.id,
          viewerId: json_params.fetch(:viewerId),
          sessionId: sess.session_id,
          createdAt: request_at,
          updatedResources: UpdatedResources.new(request_at, user, user_device, init_cards, [init_deck], nil, login_bonuses, presents).as_json,
        )
      end
    end

    # login
    post '/login' do
      request_at = get_request_time()

      query = 'SELECT * FROM users WHERE id=?'
      user = db.xquery(query, json_params[:userId]).first&.then { User.new(_1) }
      raise HttpError.new(404, 'not found user') unless user

      # check ban
      is_ban = check_ban?(user.id)
      raise HttpError.new(403, 'forbidden') if is_ban

      # viewer id check
      check_viewer_id!(user.id, json_params[:viewerId])

      db_transaction do
        # sessionを更新
        query = 'UPDATE user_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
        db.xquery(query, request_at, json_params[:userId])

        session_id = generate_id()
        sess_id = generate_uuid()
        sess = Session.new(
          id: session_id,
          user_id: json_params[:userId],
          session_id: sess_id,
          created_at: request_at,
          updated_at: request_at,
          expired_at: request_at + 86400,
        )
        query = 'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
        db.xquery(query, sess.id, sess.user_id, sess.session_id, sess.created_at, sess.updated_at, sess.expired_at)

        # すでにログインしているユーザはログイン処理をしない
        if complete_today_login?(user.last_activated_at, request_at)
          user.updated_at = request_at
          user.last_activated_at = request_at

          query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?'
          db.xquery(query, request_at, request_at, json_params[:userId])

          next json(
            viewerId: json_params[:viewerId],
            sessionId: sess.session_id,
            updatedResources: UpdatedResources.new(request_at, user, nil, nil, nil, nil, nil, nil).as_json,
          )
        end

        # login process
        user, login_bonuses, presents = login_process(json_params[:userId], request_at)

        json(
          viewerId: json_params[:viewerId],
          sessionId: sess.session_id,
          updatedResources: UpdatedResources.new(request_at, user, nil, nil, nil, nil, login_bonuses, presents).as_json,
        )
      end
    end

    # list_gacha ガチャ一覧
    get '/user/:user_id/gacha/index', check_session: true do
      user_id = get_user_id()
      request_at = get_request_time()

      query = 'SELECT * FROM gacha_masters WHERE start_at <= ? AND end_at >= ? ORDER BY display_order ASC'
      gacha_master_list = db.xquery(query, request_at, request_at).to_a

      if gacha_master_list.empty?
        next json(
          oneIimeToken: '',
          gachas: [],
        )
      end

      # ガチャ排出アイテム取得
      gacha_data_list = []
      query = 'SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC'
      gacha_master_list.each do |v|
        gacha_item = db.xquery(query, v.fetch(:id)).to_a
        raise HttpError.new(404, 'not found gacha item') if gacha_item.empty?

        gacha_data_list.push(
          gacha: GachaMaster.new(v).as_json,
          gachaItemList: gacha_item.map { GachaItemMaster.new(_1).as_json },
        )
      end
      # generate one time token
      query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
      db.xquery(query, request_at, user_id)
      token_id = generate_id()
      tk = generate_uuid()
      token = UserOneTimeToken.new(
        id: token_id,
        user_id:,
        token: tk,
        token_type: 1,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 600,
      )
      query = 'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
      db.xquery(query, token.id, token.user_id, token.token, token.token_type, token.created_at, token.updated_at, token.expired_at)

      json(
        oneTimeToken: token.token,
        gachas: gacha_data_list,
      )
    end

    # draw_gacha ガチャを引く
    post '/user/:user_id/gacha/draw/:gacha_id/:n', check_session: true do
      user_id = get_user_id()
      gacha_id = params[:gacha_id]
      raise HttpError.new(403, 'invalid gachaID') if !gacha_id.is_a?(String) || gacha_id.empty?

      
      gacha_count = begin
        Integer(params[:n], 10)
      rescue ArgumentError => e
        raise HttpError.new(400, e.inspect)
      end
      if gacha_count != 1 && gacha_count != 10
        raise HttpError.new(400, 'invalid draw gacha times')
      end

      request_at = get_request_time()

      check_one_time_token!(json_params[:oneTimeToken], 1, request_at)
      check_viewer_id!(user_id, json_params[:viewerId])

      consumed_coin = gacha_count * 1000

      # userのisuconが足りるか
      query = 'SELECT * FROM users WHERE id=?'
      user = db.xquery(query, user_id).first
      raise HttpError.new(404, 'not found user') unless user
      raise HttpError.new(409, 'not enough isucon') if user.fetch(:isu_coin) < consumed_coin

      # gachaIDからガチャマスタの取得
      query = 'SELECT * FROM gacha_masters WHERE id=? AND start_at <= ? AND end_at >= ?'
      gacha_info = db.xquery(query, gacha_id, request_at, request_at).first
      raise HttpError.new(404, 'not found gacha') unless gacha_info

      # gachaItemMasterからアイテムリスト取得
      gacha_item_list = db.xquery('SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC', gacha_id).to_a
      raise HttpError.new(404, 'not found gacha item') if gacha_item_list.empty?

      # weightの合計値を算出
      sum = db.xquery('SELECT SUM(weight) as sum FROM gacha_item_masters WHERE gacha_id=?', gacha_id).first.fetch(:sum)
      raise HttpError.new(404, '') unless sum

      # random値の導出 & 抽選
      result = []
      gacha_count.times do
        random = rand(sum)
        boundary = 0
        gacha_item_list.each do |v|
          boundary += v.fetch(:weight)
          if random < boundary
            result.push(v)
            break
          end
        end
      end

      db_transaction do
        # 直付与 => プレゼントに入れる
        presents = []

        result.each do |v|
          present_id = generate_id()
          present = UserPresent.new(
            id: present_id,
            user_id: user_id,
            sent_at: request_at,
            item_type: v.fetch(:item_type),
            item_id: v.fetch(:item_id),
            amount: v.fetch(:amount),
            present_message: "#{gacha_info.fetch(:name)}の付与アイテムです",
            created_at: request_at,
            updated_at: request_at,
          )
          query = 'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)'
          db.xquery(query, present.id, present.user_id, present.sent_at, present.item_type, present.item_id, present.amount, present.present_message, present.created_at, present.updated_at)

          presents.push(present)
        end

        # isuconをへらす
        query = 'UPDATE users SET isu_coin=? WHERE id=?'
        total_coin = user.fetch(:isu_coin) - consumed_coin
        db.xquery(query, total_coin, user_id)

        json(
          presents: presents.map(&:as_json),
        )
      end
    end

    # list_present
    get '/user/:user_id/present/index/:n', check_session: true do
      n = begin
        Integer(params[:n], 10)
      rescue ArgumentError
        raise HttpError.new(400, 'invalid index number (n) parameter')
      end
      if n == 0 
        raise HttpError.new(400, 'index number (n) should be more than or equal to 1')
      end

      user_id = begin
        get_user_id()
      rescue HttpError
        raise HttpError.new(400, 'invalid userID parameter')
      end

      offset = PRESENT_COUNT_PER_PAGE * (n - 1)
      query = <<~EOF
        SELECT * FROM user_presents
        WHERE user_id = ? AND deleted_at IS NULL
        ORDER BY created_at DESC, id
        LIMIT #{PRESENT_COUNT_PER_PAGE} OFFSET #{offset}
      EOF
      present_list = db.xquery(query, user_id).to_a

      present_count = db.xquery('SELECT COUNT(*) as cnt FROM user_presents WHERE user_id = ? AND deleted_at IS NULL', user_id).first.fetch(:cnt)

      is_next = present_count > (offset + PRESENT_COUNT_PER_PAGE)

      json(
        presents: present_list.map { UserPresent.new(_1).as_json },
        isNext: is_next,
      )
    end

    # receive_present プレゼント受け取り
    post '/user/:user_id/present/receive', check_session: true do
      user_id = get_user_id()
      request_at = get_request_time()

      if !json_params[:presentIds].is_a?(Array) || json_params[:presentIds].empty?
        raise HttpError.new(429, 'presentIds is empty')
      end

      check_viewer_id!(user_id, json_params[:viewerId])

      # user_presentsに入っているが未取得のプレゼント取得
      query = 'SELECT * FROM user_presents WHERE id IN (?) AND deleted_at IS NULL'
      obtain_present = db.xquery(query, json_params[:presentIds]).map { UserPresent.new(_1) }
      
      if obtain_present.empty?
        next json(
          updatedResources: UpdatedResources.new(request_at, nil, nil, nil, nil, nil, nil, []).as_json,
        )
      end

      db_transaction do
        # 配布処理
        obtain_present.each do |v|
          raise HttpError.new(500, 'received present') if v.deleted_at
          v.updated_at = request_at
          v.deleted_at = request_at

          query = 'UPDATE user_presents SET deleted_at=?, updated_at=? WHERE id=?'
          db.xquery(query, request_at, request_at, v.id)

          obtain_item(v.user_id, v.item_id, v.item_type, v.amount, request_at)
        end
      end

      json(
        updatedResources: UpdatedResources.new(request_at, nil, nil, nil, nil, nil, nil, obtain_present).as_json,
      )
    end

    # list_item アイテムリスト
    get '/user/:user_id/item', check_session: true do
      user_id = get_user_id()

      request_at = get_request_time()

      query = 'SELECT * FROM users WHERE id=?'
      user = db.xquery(query, user_id).first
      raise HttpError.new(404, 'not found user') unless user

      query = 'SELECT * FROM user_items WHERE user_id = ?'
      item_list = db.xquery(query, user_id).to_a

      query = 'SELECT * FROM user_cards WHERE user_id=?'
      card_list = db.xquery(query, user_id).to_a

      # generate one time token
      query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
      db.xquery(query, request_at, user_id)

      token_id = generate_id()
      tk = generate_uuid()
      token = UserOneTimeToken.new(
        id: token_id,
        user_id: user_id,
        token: tk,
        token_type: 2,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 600,
      )
      query = 'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
      db.xquery(query, token.id, token.user_id, token.token, token.token_type, token.created_at, token.updated_at, token.expired_at)

      json(
        oneTimeToken: token.token,
        items: item_list.map { UserItem.new(_1).as_json },
        user: User.new(user).as_json,
        cards: card_list.map { UserCard.new(_1).as_json },
      )
    end

    ConsumeUserItemData = Struct.new(
      :id,
      :user_id,
      :item_id,
      :item_type,
      :amount,
      :created_at,
      :updated_at,
      :gained_exp,
      :consume_amount, # 消費量

      keyword_init: true,
    )

    TargetUserCardData = Struct.new(
      :id,
      :user_id,
      :card_id,
      :amount_per_sec,
      :level,
      :total_exp,

      # lv1のときの生産性
      :base_amount_per_sec,
      # 最高レベル
      :max_level,
      # lv maxのときの生産性
      :max_amount_per_sec,
      # lv1 -> lv2に上がるときのexp
      :base_exp_per_level,

      keyword_init: true,
    )

    # add_exp_to_card 装備強化
    post '/user/:user_id/card/addexp/:card_id', check_session: true do
      card_id = begin
        Integer(params[:card_id], 10)
      rescue ArgumentError => e
        raise HttpError.new(400, e.inspect)
      end

      user_id = get_user_id()

      request_at = get_request_time()

      check_one_time_token!(json_params[:oneTimeToken], 2, request_at)

      check_viewer_id!(user_id, json_params[:viewerId])

      # get target card
      query = <<~EOF
        SELECT uc.id , uc.user_id , uc.card_id , uc.amount_per_sec , uc.level, uc.total_exp, im.amount_per_sec as 'base_amount_per_sec', im.max_level , im.max_amount_per_sec , im.base_exp_per_level
        FROM user_cards as uc
        INNER JOIN item_masters as im ON uc.card_id = im.id
        WHERE uc.id = ? AND uc.user_id=?
      EOF
      card = db.xquery(query, card_id, user_id).first&.then { TargetUserCardData.new(_1) }
      raise HttpError.new(404, '') unless card

      if card.level == card.max_level
        raise HttpError.new(400, 'target card is max level')
      end

      # 消費アイテムの所持チェック
      items = []
      query = <<~EOF
        SELECT ui.id, ui.user_id, ui.item_id, ui.item_type, ui.amount, ui.created_at, ui.updated_at, im.gained_exp
        FROM user_items as ui
        INNER JOIN item_masters as im ON ui.item_id = im.id
        WHERE ui.item_type = 3 AND ui.id=? AND ui.user_id=?
      EOF
      json_params[:items].each do |v|
        item = db.xquery(query, v[:id], user_id).first&.then { ConsumeUserItemData.new(_1) }
        raise HttpError.new(404, '') unless item

        raise HttpError.new(400, 'item not enough') if v[:amount] > item.amount

        item.consume_amount = v[:amount]
        items.push(item)
      end

      # 経験値付与
      # 経験値をカードに付与
      items.each do |v|
        card.total_exp += v.gained_exp * v.consume_amount
      end

      # lvup判定(lv upしたら生産性を加算)
      loop do
        next_lv_threshold = (card.base_exp_per_level.to_f * (1.2 ** (card.level-1))).to_i
        break if next_lv_threshold > card.total_exp

        # lv up処理
        card.level += 1
        card.amount_per_sec += (card.max_amount_per_sec - card.base_amount_per_sec) / (card.max_level - 1)
      end

      db_transaction do
        # cardのlvと経験値の更新、itemの消費
        query = 'UPDATE user_cards SET amount_per_sec=?, level=?, total_exp=?, updated_at=? WHERE id=?'
        db.xquery(query, card.amount_per_sec, card.level, card.total_exp, request_at, card.id)

        query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?'
        items.each do |v|
          db.xquery(query, v.amount - v.consume_amount, request_at, v.id)
        end

        # get response data
        query = 'SELECT * FROM user_cards WHERE id=?'
        result_card = db.xquery(query, card.id).first&.then { UserCard.new(_1) }
        raise HttpError.new(404, 'not found card') unless result_card
        result_items = items.map do |v|
          UserItem.new(
            id: v.id,
            user_id: v.user_id,
            item_id: v.item_id,
            item_type: v.item_type,
            amount: v.amount - v.consume_amount,
            created_at: v.created_at,
            updated_at: request_at,
          )
        end

        json(
          updatedResources: UpdatedResources.new(request_at, nil, nil, [result_card], nil, result_items, nil, nil).as_json,
        )
      end
    end

    # update_deck 装備変更
    post '/user/:user_id/card', check_session: true do
      user_id = get_user_id()

      if !json_params[:cardIds].is_a?(Array) || json_params[:cardIds].size != DECK_CARD_NUMBER
        raise HttpError.new(400, 'invalid number of cards')
      end

      request_at = get_request_time()

      check_viewer_id!(user_id, json_params[:viewerId])

      # カード所持情報のバリデーション
      query = 'SELECT * FROM user_cards WHERE id IN (?)'
      cards = db.xquery(query, json_params.fetch(:cardIds)).to_a
      raise HttpError.new(400, 'invalid card ids') if cards.size != DECK_CARD_NUMBER

      db_transaction do
        # update data
        query = 'UPDATE user_decks SET updated_at=?, deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
        db.xquery(query, request_at, request_at, user_id)

        user_deck_id = generate_id()
        new_deck = UserDeck.new(
          id: user_deck_id,
          user_id:,
          user_card_id_1: json_params.fetch(:cardIds).fetch(0),
          user_card_id_2: json_params.fetch(:cardIds).fetch(1),
          user_card_id_3: json_params.fetch(:cardIds).fetch(2),
          created_at: request_at,
          updated_at: request_at,
        )
        query = 'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
        db.xquery(query, new_deck.id, new_deck.user_id, new_deck.user_card_id_1, new_deck.user_card_id_2, new_deck.user_card_id_3, new_deck.created_at, new_deck.updated_at)

        json(
          updatedResources: UpdatedResources.new(request_at, nil, nil, nil, [new_deck], nil, nil, nil).as_json,
        )
      end
    end

    # reward ゲーム報酬受取
    post '/user/:user_id/reward', check_session: true do
      user_id = get_user_id()

      request_at = get_request_time()

      check_viewer_id!(user_id, json_params[:viewerId])

      # 最後に取得した報酬時刻取得
      query = 'SELECT * FROM users WHERE id=?'
      user = db.xquery(query, user_id).first&.then { User.new(_1) }
      raise HttpError.new(404, 'not found user') unless user

      # 使っているデッキの取得
      query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL'
      deck = db.xquery(query, user_id).first
      raise HttpError.new(404, '') unless deck

      query = 'SELECT * FROM user_cards WHERE id IN (?, ?, ?)'
      cards = db.xquery(query, deck.fetch(:user_card_id_1), deck.fetch(:user_card_id_2), deck.fetch(:user_card_id_3)).to_a
      raise HttpError.new(400, 'invalid card length') if cards.size != 3

      # 経過時間*生産性のcoin (1椅子 = 1coin)
      past_time = request_at - user.last_getreward_at
      get_coin = past_time * (cards[0].fetch(:amount_per_sec) + cards[1].fetch(:amount_per_sec) + cards[2].fetch(:amount_per_sec))

      # 報酬の保存(ゲームない通貨を保存)(users)
      user.isu_coin += get_coin
      user.last_getreward_at = request_at
      query = 'UPDATE users SET isu_coin=?, last_getreward_at=? WHERE id=?'
      db.xquery(query, user.isu_coin, user.last_getreward_at, user.id)

      json(
        updatedResources: UpdatedResources.new(request_at, user, nil, nil, nil, nil, nil).as_json,
      )
    end

    # home
    get '/user/:user_id/home', check_session: true do
      user_id = get_user_id()

      request_at = get_request_time()

      # 装備情報
      query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL'
      deck = db.xquery(query, user_id).first&.then { UserDeck.new(_1) }

      # 生産性
      cards = []
      if deck
        card_ids = [deck.user_card_id_1, deck.user_card_id_2, deck.user_card_id_3]
        cards = db.xquery('SELECT * FROM user_cards WHERE id IN (?)', card_ids).map { UserCard.new(_1) }
      end
      total_amount_per_sec = cards.sum(&:amount_per_sec)

      # 経過時間
      query = 'SELECT * FROM users WHERE id=?'
      user = db.xquery(query, user_id).first&.then { User.new(_1) }
      raise HttpError.new(404, 'not found user') unless user
      past_time = request_at - user.last_getreward_at

      json(
        now: request_at,
        user: user.as_json,
        deck: deck.as_json,
        totalAmountPerSec: total_amount_per_sec,
        pastTime: past_time,
      )
    end

    register Admin
  end

  UpdatedResources = Struct.new(:now, :user, :user_device, :user_cards, :user_decks, :user_items, :user_login_bonuses, :user_presents) do
    def as_json
      {
        now: now,
      }.tap do |j|
        j[:user] = user.as_json if user
        j[:userDevice] = user_device.as_json if user_device
        j[:userCards] = user_cards.map(&:as_json) if user_cards && !user_cards.empty?
        j[:userDecks] = user_decks.map(&:as_json) if user_decks && !user_decks.empty?
        j[:userItems] = user_items.map(&:as_json) if user_items && !user_items.empty?
        j[:userLoginBonuses] = user_login_bonuses.map(&:as_json) if user_login_bonuses && !user_login_bonuses.empty?
        j[:userPresents] = user_presents.map(&:as_json) if user_presents && !user_presents.empty?
      end
    end
  end

  ## ######################################
  ## entity

  User = Struct.new(:id, :isu_coin, :last_getreward_at, :last_activated_at, :registered_at, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        isuCoin: isu_coin,
        lastGetRewardAt: last_getreward_at,
        lastActivatedAt: last_activated_at,
        registeredAt: registered_at,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserDevice = Struct.new(:id, :user_id, :platform_id, :platform_type, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        platformId: platform_id,
        platformType: platform_type,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserBan = Struct.new(:id, :user_id, :created_at, :updated_at, :deleted_at, keyword_init: true)

  UserCard = Struct.new(:id, :user_id, :card_id, :amount_per_sec, :level, :total_exp, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        cardId: card_id,
        amountPerSec: amount_per_sec,
        level: level,
        totalExp: total_exp,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserDeck = Struct.new(:id, :user_id, :user_card_id_1, :user_card_id_2, :user_card_id_3, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        cardId1: user_card_id_1,
        cardId2: user_card_id_2,
        cardId3: user_card_id_3,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserItem = Struct.new(:id, :user_id, :item_type, :item_id, :amount, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        itemType: item_type,
        itemId: item_id,
        amount: amount,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserLoginBonus = Struct.new(:id, :user_id, :login_bonus_id, :last_reward_sequence, :loop_count, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        loginBonusId: login_bonus_id,
        lastRewardSequence: last_reward_sequence,
        loopCount: loop_count,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserPresent = Struct.new(:id, :user_id, :sent_at, :item_type, :item_id, :amount, :present_message, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        sentAt: sent_at,
        itemType: item_type,
        itemId: item_id,
        amount: amount,
        presentMessage: present_message,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserPresentAllReceivedHistory = Struct.new(:id, :user_id, :present_all_id, :received_at, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        presentAllId: present_all_id,
        receivedAt: received_at,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  Session = Struct.new(:id, :user_id, :session_id, :expired_at, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        sessionId: session_id,
        expiredAt: expired_at,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  UserOneTimeToken = Struct.new(:id, :user_id, :token, :token_type, :expired_at, :created_at, :updated_at, :deleted_at, keyword_init: true) do
    def as_json
      {
        id: id,
        userId: user_id,
        token: token,
        tokenType: token_type,
        expiredAt: expired_at,
        createdAt: created_at,
        updatedAt: updated_at,
      }.tap do |j|
        j[:deletedAt] = deleted_at if deleted_at
      end
    end
  end

  ## ######################################
  ## master

  GachaMaster = Struct.new(:id, :name, :start_at, :end_at, :display_order, :created_at, keyword_init: true) do
    def as_json
      {
        id: id,
        name: name,
        startAt: start_at,
        endAt: end_at,
        displayOrder: display_order,
        createdAt: created_at,
      }
    end
  end

  GachaItemMaster = Struct.new(:id, :gacha_id, :item_type, :item_id, :amount, :weight, :created_at, keyword_init: true) do
    def as_json
      {
        id: id,
        gachaId: gacha_id,
        itemType: item_type,
        itemId: item_id,
        amount: amount,
        weight: weight,
        createdAt: created_at,
      }
    end
  end

  ItemMaster = Struct.new(:id, :item_type, :name, :description, :amount_per_sec, :max_level, :max_amount_per_sec, :base_exp_per_level, :gained_exp, :shortening_min, keyword_init: true) do
    def as_json
      {
        id: id,
        itemType: item_type,
        name: name,
        description: description,
        amountPerSec: amount_per_sec,
        maxLevel: max_level,
        maxAmountPerSec: max_amount_per_sec,
        baseExpPerLevel: base_exp_per_level,
        gainedExp: gained_exp,
        shorteningMin: shortening_min,
      }
    end
  end

  LoginBonusMaster = Struct.new(:id, :start_at, :end_at, :column_count, :looped, :created_at, keyword_init: true) do
    def as_json
      {
        id: id,
        startAt: start_at,
        endAt: end_at,
        columnCount: column_count,
        looped: looped,
        createdAt: created_at,
      }
    end
  end

  LoginBonusRewardMaster = Struct.new(:id, :login_bonus_id, :reward_sequence, :item_type, :item_id, :amount, :created_at, keyword_init: true) do
    def as_json
      {
        id: id,
        loginBonusId: login_bonus_id,
        rewardSequence: reward_sequence,
        itemType: item_type,
        itemId: item_id,
        amount: amount,
        createdAt: created_at,
      }
    end
  end

  PresentAllMaster = Struct.new(:id, :registered_start_at, :registered_end_at, :item_type, :item_id, :amount, :present_message, :created_at, keyword_init: true) do
    def as_json
      {
        id: id,
        registeredStartAt: registered_start_at,
        registeredEndAt: registered_end_at,
        itemType: item_type,
        itemId: item_id,
        amount: amount,
        presentMessage: present_message,
        createdAt: created_at,
      }
    end
  end

  VersionMaster = Struct.new(:id, :status, :master_version, keyword_init: true) do
    def as_json
      {
        id: id,
        status: status,
        masterVersion: master_version,
      }
    end
  end
end
