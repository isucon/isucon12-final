# frozen_string_literal: true

require 'csv'
require 'bcrypt'

module Isuconquest
  module Admin
    def self.registered(app)
      app.set(:admin_check_session) do |_bool|
        condition do
          sess_id = request.get_header('HTTP_X_SESSION')
          raise HttpError.new(401, 'unauthorized user') if sess_id.nil? || sess_id.empty?

          query = 'SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL'
          admin_session = db.xquery(query, sess_id).first
          raise HttpError.new(401, 'unauthorized user') unless admin_session

          request_at = get_request_time()

          if admin_session.fetch(:expired_at) < request_at
            query = 'UPDATE admin_session SET deleted_at=? WHERE session_id=?'
            db.xquery(query, request_at, sess_id)
            raise HttpError.new(401, 'session expired')
          end
        end
      end

      # admin_login 管理者権限ログイン
      app.post '/admin/login' do
        request_at = get_request_time()

        db_transaction do
          # userの存在確認
          query = 'SELECT * FROM admin_users WHERE id=?'
          user = db.xquery(query, json_params[:userId]).first
          raise HttpError.new(404, 'not found user') unless user

          # verify password
          verify_password(user.fetch(:password), json_params[:password])

          query = 'UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?'
          db.xquery(query, request_at, request_at, json_params[:userId])

          # すでにあるsessionをdeleteにする
          query = 'UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
          db.xquery(query, request_at, json_params[:userId])

          # create session
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

          query = 'INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
          db.xquery(query, sess.id, sess.user_id, sess.session_id, sess.created_at, sess.updated_at, sess.expired_at)

          json(
            session: sess.as_json,
          )
        end
      end

      # admin_logout 管理者権限ログアウト
      app.delete '/admin/logout', admin_check_session: true do
        sess_id = request.get_header('HTTP_X_SESSION')
        request_at = get_request_time()
        # すでにあるsessionをdeleteにする
        query = 'UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL'
        db.xquery(query, request_at, sess_id)

        status 204
        nil
      end

      # admin_list_master マスタデータ閲覧
      app.get '/admin/master', admin_check_session: true do
        master_versions = db.xquery('SELECT * FROM version_masters').to_a.map { VersionMaster.new(_1).as_json }
        items = db.xquery('SELECT * FROM item_masters').to_a.map { ItemMaster.new(_1).as_json }
        gachas = db.xquery('SELECT * FROM gacha_masters').to_a.map { GachaMaster.new(_1).as_json }
        gacha_items = db.xquery('SELECT * FROM gacha_item_masters').to_a.map { GachaItemMaster.new(_1).as_json }
        present_alls = db.xquery('SELECT * FROM present_all_masters').to_a.map { PresentAllMaster.new(_1).as_json }
        login_bonuses = db.xquery('SELECT * FROM login_bonus_masters').to_a.map { LoginBonusMaster.new(_1).as_json }
        login_bonus_rewards = db.xquery('SELECT * FROM login_bonus_reward_masters').to_a.map { LoginBonusRewardMaster.new(_1).as_json }

        json(
          versionMaster: master_versions,
          items:,
          gachas:,
          gachaItems: gacha_items,
          presentAlls: present_alls,
          loginBonuses: login_bonuses,
          loginBonusRewards: login_bonus_rewards,
        )
      end

      # admin_update_master マスタデータ更新
      app.put '/admin/master', admin_check_session: true do
        db_transaction do
          # version master
          version_master_recs = read_form_file_to_csv(:versionMaster)
          if version_master_recs
            placeholders = version_master_recs.map { '(?, ?, ?)' }.join(',')
            query = "INSERT INTO version_masters(id, status, master_version) VALUES #{placeholders} ON DUPLICATE KEY UPDATE status=VALUES(status), master_version=VALUES(master_version)"
            db.xquery(query, *version_master_recs.flatten)
          end

          # item
          item_master_recs = read_form_file_to_csv(:itemMaster)
          if item_master_recs
            placeholders = item_master_recs.map { '(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)' }.join(',')
            query = [
              'INSERT INTO item_masters(id, item_type, name, description, amount_per_sec, max_level, max_amount_per_sec, base_exp_per_level, gained_exp, shortening_min)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE item_type=VALUES(item_type), name=VALUES(name), description=VALUES(description), amount_per_sec=VALUES(amount_per_sec), max_level=VALUES(max_level), max_amount_per_sec=VALUES(max_amount_per_sec), base_exp_per_level=VALUES(base_exp_per_level), gained_exp=VALUES(gained_exp), shortening_min=VALUES(shortening_min)',
            ].join(' ')
            db.xquery(query, *item_master_recs.flatten)
          end

          # gacha
          gacha_recs = read_form_file_to_csv(:gachaMaster)
          if gacha_recs
            placeholders = item_master_recs.map { '(?, ?, ?, ?, ?, ?)' }.join(',')
            query = [
              'INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)',
            ].join(' ')
            db.xquery(query, *gacha_recs.flatten)
          end

          # gacha item
          gacha_item_recs = read_form_file_to_csv(:gachaItemMaster)
          if gacha_item_recs
            placeholders = gacha_item_recs.map { '(?, ?, ?, ?, ?, ?, ?)' }.join(',')
            query = [
              'INSERT INTO gacha_item_masters(id, gacha_id, item_type, item_id, amount, weight, created_at)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE gacha_id=VALUES(gacha_id), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), weight=VALUES(weight), created_at=VALUES(created_at)',
            ].join(' ')
            db.xquery(query, *gacha_item_recs.flatten)
          end

          # present all
          present_all_recs = read_form_file_to_csv(:presentAllMaster)
          if present_all_recs
            placeholders = present_all_recs.map { '(?, ?, ?, ?, ?, ?, ?, ?)' }.join(',')
            query = [
              'INSERT INTO present_all_masters(id, registered_start_at, registered_end_at, item_type, item_id, amount, present_message, created_at)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE registered_start_at=VALUES(registered_start_at), registered_end_at=VALUES(registered_end_at), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), present_message=VALUES(present_message), created_at=VALUES(created_at)',
            ].join(' ')
            db.xquery(query, *present_all_recs.flatten)
          end

          # login bonuses
          login_bonus_recs = read_form_file_to_csv(:loginBonusMaster)
          if login_bonus_recs
            placeholders = login_bonus_recs.map { '(?, ?, ?, ?, ?, ?)' }.join(',')
            data = login_bonus_recs.flat_map  { looped = _1[4] == 'TRUE'; [_1[0], _1[1], _1[2], _1[3], looped, _1[5]] }
            query = [
              'INSERT INTO login_bonus_masters(id, start_at, end_at, column_count, looped, created_at)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE start_at=VALUES(start_at), end_at=VALUES(end_at), column_count=VALUES(column_count), looped=VALUES(looped), created_at=VALUES(created_at)',
            ].join(' ')
            db.xquery(query, *data)
          end

          # login bonus rewards
          login_bonus_reward_recs = read_form_file_to_csv(:loginBonusRewardMaster)
          if login_bonus_reward_recs
            placeholders = login_bonus_reward_recs.map { '(?, ?, ?, ?, ?, ?, ?)' }.join(',')
            query = [
              'INSERT INTO login_bonus_reward_masters(id, login_bonus_id, reward_sequence, item_type, item_id, amount, created_at)',
              "VALUES #{placeholders}",
              'ON DUPLICATE KEY UPDATE login_bonus_id=VALUES(login_bonus_id), reward_sequence=VALUES(reward_sequence), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), created_at=VALUES(created_at)',
            ].join(' ')
            db.xquery(query, *login_bonus_reward_recs.flatten)
          end

          active_master = db.query('SELECT * FROM version_masters WHERE status=1').first&.then { VersionMaster.new(_1) }
          raise HttpError.new(500, 'invalid active_master') unless active_master
          json(
            versionMaster: active_master.as_json,
          )
        end
      end

      # admin_user ユーザの詳細画面
      app.get '/admin/user/:user_id', admin_check_session: true do
        user_id = get_user_id()

        query = 'SELECT * FROM users WHERE id=?'
        user = db.xquery(query, user_id).first&.then { User.new(_1) }
        raise HttpError.new(404, 'not found user') unless user

        query = 'SELECT * FROM user_devices WHERE user_id=?'
        devices = db.xquery(query, user_id).map { UserDevice.new(_1) }

        query = 'SELECT * FROM user_cards WHERE user_id=?'
        cards = db.xquery(query, user_id).map { UserCard.new(_1) }

        query = 'SELECT * FROM user_decks WHERE user_id=?'
        decks = db.xquery(query, user_id).map { UserDeck.new(_1) }

        query = 'SELECT * FROM user_items WHERE user_id=?'
        items = db.xquery(query, user_id).map { UserItem.new(_1) }

        query = 'SELECT * FROM user_login_bonuses WHERE user_id=?'
        login_bonuses = db.xquery(query, user_id).map { UserLoginBonus.new(_1) }

        query = 'SELECT * FROM user_presents WHERE user_id=?'
        presents = db.xquery(query, user_id).map { UserPresent.new(_1) }

        query = 'SELECT * FROM user_present_all_received_history WHERE user_id=?'
        present_history = db.xquery(query, user_id).map { UserPresentAllReceivedHistory.new(_1) }

        json(
          user: user.as_json,
          userDevices: devices.map(&:as_json),
          userCards: cards.map(&:as_json),
          userDecks: decks.map(&:as_json),
          userItems: items.map(&:as_json),
          userLoginBonuses: login_bonuses.map(&:as_json),
          userPresents: presents.map(&:as_json),
          userPresentAllReceivedHistory: present_history.map(&:as_json),
        )
      end

      # admin_ban_user ユーザBAN処理
      app.post '/admin/user/:user_id/ban', admin_check_session: true do
        user_id = get_user_id()

        request_at = get_request_time()

        query = 'SELECT * FROM users WHERE id=?'
        user = db.xquery(query, user_id).first&.then { User.new(_1) }
        raise HttpError.new(404, 'not found user') unless user

        ban_id = generate_id()
        query = 'INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?'
        db.xquery(query, ban_id, user_id, request_at, request_at, request_at)

        json(
          user: user.as_json,
        )
      end

      app.helpers do
        def verify_password(hash, pw)
          unless BCrypt::Password.new(hash) == pw
            raise HttpError.new(401, 'unauthorized user')
          end
        end

        def read_form_file_to_csv(name)
          file = params.dig(name, :tempfile)
          return nil unless file
          file.set_encoding(Encoding::UTF_8)
          csv = CSV.new(file, headers: true, return_headers: false)
          csv.map(&:fields)
        rescue CSV::MalformedCSVError => e
          raise HttpError.new(400, e.inspect)
        end
      end
    end
  end
end
