package IsuConquest::API;
use v5.36;
use utf8;
use experimental qw(defer builtin);
use builtin qw(true false);

use Kossy;
use HTTP::Status qw(:constants);
use HTTP::Date qw(str2time);

use Cpanel::JSON::XS;
use Cpanel::JSON::XS::Type;

$Kossy::JSON_SERIALIZER = Cpanel::JSON::XS->new()->ascii(0)->canonical;

use IsuConquest::ErrorMessage qw(
    ErrInvalidMasterVersion
    ErrUserNotFound
    ErrUnauthorized
    ErrForbidden
    ErrExpiredSession
    ErrInvalidToken
    ErrUserDeviceNotFound
    ErrLoginBonusRewardNotFound
    ErrItemNotFound
    ErrInvalidItemType
    ErrInvalidRequestBody
);

use IsuConquest::Response qw(
    fail
    success
    normalize_response_keys

    User
    UserDevice
    UserCard
    UserDeck
    UserItem
    UserLoginBonus
    UserPresent
    GachaMaster
    GachaItemMaster
);

use IsuConquest::IDGenerator qw(generate_id generate_uuid);
use IsuConquest::DBConnector qw(connect_db);

use constant {
    DECK_CARD_NUMBER => 3,
    PRESENT_COUNT_PER_PAGE => 100,
};

use constant SQL_DIRECTORY => "../sql/";

sub db($self) {
    $self->{dbh} ||= connect_db
}

# utility
post '/initialize' => \&initialize_handler;
get  '/health'     => sub ($self, $c) { $c->halt_text(HTTP_OK, "OK") };
get  '/'           => sub ($self, $c) { $c->halt_no_content(HTTP_OK) };

filter 'api_filter'                => \&api_filter;
filter 'check_session_filter'      => \&check_session_filter;
filter 'allow_json_request_filter' => \&allow_json_request_filter;

my $api_filters = [qw/allow_json_request_filter api_filter/];
post '/user',  $api_filters => \&create_user_handler;
post '/login', $api_filters => \&login_handler;

my $api_session_filters = [$api_filters->@*, 'check_session_filter'];
get  '/user/{user_id:\d+}/gacha/index'                       => $api_session_filters, \&list_gacha_handler;
post '/user/{user_id:\d+}/gacha/draw/{gacha_id:\d+}/{n:\d+}' => $api_session_filters, \&draw_gacha_handler;
get  '/user/{user_id:\d+}/present/index/{n:\d+}'             => $api_session_filters, \&list_present_handler;
post '/user/{user_id:\d+}/present/receive'                   => $api_session_filters, \&receive_present_handler;
get  '/user/{user_id:\d+}/item'                              => $api_session_filters, \&list_item_handler;
post '/user/{user_id:\d+}/card/addexp/{card_id:\d+}'         => $api_session_filters, \&add_exp_to_card_handler;
post '/user/{user_id:\d+}/card'                              => $api_session_filters, \&update_deck_handler;
post '/user/{user_id:\d+}/reward'                            => $api_session_filters, \&reward_handler;
get  '/user/{user_id:\d+}/home'                              => $api_session_filters, \&home_handler;

sub allow_json_request_filter($app) {
    return sub ($self, $c) {
        $c->env->{'kossy.request.parse_json_body'} = 1;
        $app->($self, $c);
    };
}

sub api_filter($app) {
    return sub ($self, $c) {
        my $request_at = str2time( $c->request->header('x-isu-date') );
        unless ($request_at) {
            $request_at = time;
        }
        $c->stash->{request_time} = $request_at;

        # マスタ確認
        my $query = "SELECT * FROM version_masters WHERE status=1";
        my $version_master = $self->db->select_row($query);
        unless ($version_master) {
            fail($c, HTTP_NOT_FOUND, "active master version is not found")
        }

        unless ($version_master->{master_version} eq $c->request->header('x-master-version')) {
            fail($c, HTTP_UNPROCESSABLE_ENTITY, ErrInvalidMasterVersion);
        }

        # check ban
        my $user_id = $c->args->{user_id};
        if ($user_id) {
            my $is_ban = $self->check_ban($user_id);
            if ($is_ban) {
                fail($c, HTTP_FORBIDDEN, ErrForbidden)
            }
        }

        $app->($self, $c);
    }
}

sub check_session_filter($app) {
    return sub ($self, $c) {
        my $session_id = $c->request->header('x-session');
        unless ($session_id) {
            fail($c, HTTP_UNAUTHORIZED, ErrUnauthorized)
        }

        my $user_id = $c->args->{user_id};
        unless ($user_id) {
            fail($c, HTTP_BAD_REQUEST, 'badrequest');
        }

        my $request_at = $c->stash->{request_time};

        my $query = "SELECT * FROM user_sessions WHERE session_id=? AND deleted_at IS NULL";
        my $user_session = $self->db->select_row($query, $session_id);
        unless ($user_session) {
            fail($c, HTTP_UNAUTHORIZED, ErrUnauthorized)
        }

        unless ($user_session->{user_id} == $user_id) {
            fail($c, HTTP_FORBIDDEN, ErrForbidden)
        }

        if ($user_session->{expired_at} < $request_at) {
            my $query = "UPDATE user_sessions SET deleted_at=? WHERE session_id=?";
            $self->db->query($query, $request_at, $session_id);
            fail($c, HTTP_UNAUTHORIZED, ErrExpiredSession)
        }

        $app->($self, $c);
    }
}

sub check_one_time_token($self, $token, $token_type, $request_at) {
    my $query = "SELECT * FROM user_one_time_tokens WHERE token=? AND token_type=? AND deleted_at IS NULL";
    my $tk = $self->db->select_row($query, $token, $token_type);
    unless ($tk) {
        return ErrInvalidToken
    }

    if ($tk->{expired_at} < $request_at) {
        my $query = "UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?";
        $self->db->query($query, $request_at, $token);
        return ErrInvalidToken
    }

    # 使ったトークンを失効する
    $query = "UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?";
    my $effected = $self->db->query($query, $request_at, $token);
    unless ($effected) {
        return ErrInvalidToken
    }

    return;
}

sub check_viewer_id($self, $user_id, $viewer_id) {
    my $query = "SELECT * FROM user_devices WHERE user_id=? AND platform_id=?";
    my $device = $self->db->select_row($query, $user_id, $viewer_id);
    unless ($device) {
        return ErrUserDeviceNotFound
    }
    return;
}

sub check_ban($self, $user_id) {
    my $user_bans = $self->db->select_row("SELECT * FROM user_bans WHERE user_id=?", $user_id);
    my $is_ban = $user_bans ? true : false;
    return $is_ban;
}

# ログイン処理
sub login_process($self, $user_id, $request_at) {
    my $query = "SELECT * FROM users WHERE id=?";
    my $user = $self->db->select_row($query, $user_id);
    unless ($user) {
        return undef, ErrUserNotFound
    }

    # ログインボーナス処理
    my ($login_bonuses, $err) = $self->obtain_login_bonus($user_id, $request_at);
    if ($err) {
        return undef, $err
    }

    # 全員プレゼント取得
    (my $presents, $err) = $self->obtain_present($user_id, $request_at);
    if ($err) {
        return undef, $err
    }

    my $isu_coin = $self->db->select_row("SELECT isu_coin FROM users WHERE id=?", $user->{id});
    unless ($isu_coin) {
        return undef, ErrUserNotFound
    }

    $user->{updated_at} = $request_at;
    $user->{last_activated_at} = $request_at;

    $query = "UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?";
    $self->db->query($query, $request_at, $request_at, $request_at);

    return {
        user          => $user,
        login_bonuses => $login_bonuses,
        presents      => $presents,
    }, undef;
}

# ログイン処理が終わっているか
sub is_complete_today_login($last_activated_at, $request_at) {
    my (undef, undef, undef, $la_day, $la_month, $la_year) = localtime($last_activated_at);
    my (undef, undef, undef, $req_day, $req_month, $req_year) = localtime($request_at);
    return $la_year  == $req_year &&
           $la_month == $req_month &&
           $la_day   == $req_day;
}

sub obtain_login_bonus($self, $user_id, $request_at) {
    # login bonus masterから有効なログインボーナスを取得
    my $query = "SELECT * FROM login_bonus_masters WHERE start_at <= ? AND end_at >= ?";
    my $login_bonuses = $self->db->select_all($query, $request_at, $request_at);

    my $send_login_bonuses = [];

    for my $bonus ($login_bonuses->@*) {
        my $init_bonus = false;

        # ボーナスの進捗取得
        my $user_bonus = $self->db->select_row(
            "SELECT * FROM user_login_bonuses WHERE user_id=? AND login_bonus_id=?",
            $user_id, $bonus->{id}
        );
        unless ($user_bonus) { # ボーナス初期化
            $init_bonus = true;

            $user_bonus = {
                id                   => generate_id(),
                user_id              => $user_id,
                login_bonus_id       => $bonus->{id},
                last_reward_sequence => 0,
                loop_count           => 1,
                created_at           => $request_at,
                updated_at           => $request_at,
            }
        }

        # ボーナス進捗更新
        if ($user_bonus->{last_reward_sequence} < $bonus->{column_count}) {
            $user_bonus->{last_reward_sequence}++
        }
        else {
            if ($bonus->{looped}) {
                $user_bonus->{loop_count} += 1;
                $user_bonus->{last_reward_sequence} = 1;
            }
            else {
                # 上限まで付与完了
                next;
            }
        }
        $user_bonus->{updated_at} = $request_at;

        # 今回付与するリソース取得
        my $reward_item = $self->db->select_row(
            "SELECT * FROM login_bonus_reward_masters WHERE login_bonus_id=? AND reward_sequence=?",
            $bonus->{id},
            $user_bonus->{last_reward_sequence},
        );
        unless ($reward_item) {
            return undef, ErrLoginBonusRewardNotFound
        }

        my (undef, $err) = $self->obtain_item($user_id, $reward_item->{item_id}, $reward_item->{item_type}, $reward_item->{amount}, $request_at);
        if ($err) {
            return undef, $err
        }

        # 進捗の保存
        if ($init_bonus) {
            $self->db->query(
                "INSERT INTO user_login_bonuses(id, user_id, login_bonus_id, last_reward_sequence, loop_count, created_at, updated_at) VALUES (:id, :user_id, :login_bonus_id, :last_reward_sequence, :loop_count, :created_at, :updated_at)",
                $user_bonus
            );
        }
        else {
            $self->db->query(
                "UPDATE user_login_bonuses SET last_reward_sequence=:last_reward_sequence, loop_count=:loop_count, updated_at=:updated_at WHERE id=:id",
                $user_bonus
            );
        }

        push $send_login_bonuses->@* => $user_bonus;
    }
    return $send_login_bonuses, undef;
}


# プレゼント付与処理
sub obtain_present($self, $user_id, $request_at) {
    my $normal_presents = $self->db->select_all(
        "SELECT * FROM present_all_masters WHERE registered_start_at <= ? AND registered_end_at >= ?",
        $request_at, $request_at,
    );

    # 全員プレゼント取得情報更新
    my $obtain_presents = [];
    for my $normal_present ($normal_presents->@*) {
        my $received = $self->db->select_row(
            "SELECT * FROM user_present_all_received_history WHERE user_id=? AND present_all_id=?",
            $user_id,
            $normal_present->{id}
        );
        if ($received) {
            # プレゼント配布済
            next;
        }

        # user present boxに入れる
        my $user_present = {
            id              => generate_id(),
            user_id         => $user_id,
            sent_at         => $request_at,
            item_type       => $normal_present->{item_type},
            item_id         => $normal_present->{item_id},
            amount          => $normal_present->{amount},
            present_message => $normal_present->{present_message},
            created_at      => $request_at,
            updated_at      => $request_at,
        };

        $self->db->query(
            "INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (:id, :user_id, :sent_at, :item_type, :item_id, :amount, :present_message, :created_at, :updated_at)",
            $user_present
        );

        # historyに入れる
        my $history = {
            id             => generate_id(),
            user_id        => $user_id,
            present_all_id => $normal_present->{id},
            received_at    => $request_at,
            created_at     => $request_at,
            updated_at     => $request_at,
        };

        $self->db->query(
            "INSERT INTO user_present_all_received_history(id, user_id, present_all_id, received_at, created_at, updated_at) VALUES (:id, :user_id, :present_all_id, :received_at, :created_at, :updated_at)",
            $history
        );
        push $obtain_presents->@* => $user_present;
    }

    return $obtain_presents, undef;
}

# アイテム付与処理
sub obtain_item($self, $user_id, $item_id, $item_type, $obtain_amount, $request_at) {

    my $obtain_coins = [];
    my $obtain_cards = [];
    my $obtain_items = [];

    if ($item_type == 1) { # coin
        my $user = $self->db->select_row("SELECT * FROM users WHERE id=?", $user_id);
        unless ($user) {
            return undef, ErrUserNotFound
        }

        my $query = "UPDATE users SET isu_coin=? WHERE id=?";
        my $total_coin = $user->{isu_coin} + $obtain_amount;
        $self->db->query($query, $total_coin, $user->{id});
        push $obtain_coins->@* => $obtain_amount;
    }
    elsif ($item_type == 2) { # card(ハンマー)
        my $item = $self->db->select_row(
            "SELECT * FROM item_masters WHERE id=? AND item_type=?",
            $item_id, $item_type
        );
        unless ($item) {
            return undef, ErrItemNotFound
        }

        my $card = {
            id             => generate_id(),
            user_id        => $user_id,
            card_id        => $item->{id},
            amount_per_sec => $item->{amount_per_sec},
            level          => 1,
            total_exp      => 0,
            created_at     => $request_at,
            updated_at     => $request_at,
        };

        $self->db->query(
            "INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (:id, :user_id, :card_id, :amount_per_sec, :level, :total_exp, :created_at, :updated_at)",
            $card
        );
        push $obtain_cards->@* => $card;
    }
    elsif ($item_type == 3 or $item_type == 4) { # 強化素材
        my $item = $self->db->select_row(
            "SELECT * FROM item_masters WHERE id=? AND item_type=?",
            $item_id, $item_type
        );
        unless ($item) {
            return undef, ErrItemNotFound
        }

        # 所持数取得
        my $user_item = $self->db->select_row(
            "SELECT * FROM user_items WHERE user_id=? AND item_id=?",
            $user_id, $item->{id},
        );
        if ($user_item) { # 更新
            $user_item->{amount} += $obtain_amount;
            $user_item->{updated_at} = $request_at;

            $self->db->query(
                "UPDATE user_items SET amount=?, updated_at=? WHERE id=?",
                $user_item->{amount},
                $user_item->{updated_at},
                $user_item->{id},
            );
        }
        else { # 新規作成
            $user_item = {
                id         => generate_id(),
                user_id    => $user_id,
                item_type  => $item->{item_type},
                item_id    => $item->{id},
                amount     => $obtain_amount,
                created_at => $request_at,
                updated_at => $request_at,
            };
            $self->db->query(
                "INSERT INTO user_items(id, user_id, item_id, item_type, amount, created_at, updated_at) VALUES (:id, :user_id, :item_id, :item_type, :amount, :created_at, :updated_at)",
                $user_item
            );
        }
        push $obtain_items->@* => $user_item;
    }
    else {
        return undef, ErrInvalidItemType
    }

    return {
        obtain_coins => $obtain_coins,
        obtain_cards => $obtain_cards,
        obtain_items => $obtain_items,
    }, undef;
}


use constant InitializeResponse => normalize_response_keys {
    language => JSON_TYPE_STRING,
};

# initialize 初期化処理
# POST /initialize
sub initialize_handler($self, $c) {
    my $err = system("/bin/sh", "-c", SQL_DIRECTORY . "init.sh");
    if ($err) {
        warn sprintf('Failed to initialize %s', $err);
        fail($c, HTTP_INTERNAL_SERVER_ERROR, $err);
    }

    return success($c, {
        language => "perl"
    }, InitializeResponse)
};

use constant UpdatedResource => normalize_response_keys {
    now                => JSON_TYPE_INT,
    user               => json_type_null_or_anyof(User),
    user_device        => json_type_null_or_anyof(UserDevice),
    user_cards         => json_type_null_or_anyof(json_type_arrayof(UserCard)),
    user_decks         => json_type_null_or_anyof(json_type_arrayof(UserDeck)),
    user_items         => json_type_null_or_anyof(json_type_arrayof(UserItem)),
    user_login_bonuses => json_type_null_or_anyof(json_type_arrayof(UserLoginBonus)),
    user_presents      => json_type_null_or_anyof(json_type_arrayof(UserPresent)),
};

use constant CreateUserResponse => normalize_response_keys {
    user_id           => JSON_TYPE_INT,
    viewer_id         => JSON_TYPE_STRING,
    session_id        => JSON_TYPE_STRING,
    created_at        => JSON_TYPE_INT,
    updated_resources => UpdatedResource
};

# createUser ユーザの作成
# POST /user
sub create_user_handler($self, $c) {
    my $viewer_id     = $c->request->body_parameters->{viewerId};
    my $platform_type = $c->request->body_parameters->{platformType};
    unless ($viewer_id && $platform_type) {
        fail($c, HTTP_BAD_REQUEST, ErrInvalidRequestBody);
    }
    if ($platform_type < 1 || $platform_type > 3) {
        fail($c, HTTP_BAD_REQUEST, ErrInvalidRequestBody);
    }

    my $request_at = $c->stash->{request_time};

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # ユーザ作成
    my $user = {
        id                 => generate_id(),
        isu_coin           => 0,
        last_getreward_at  => $request_at,
        last_activated_at  => $request_at,
        registered_at      => $request_at,
        created_at         => $request_at,
        updated_at         => $request_at,
    };

    $self->db->query(
        "INSERT INTO users(id, last_activated_at, registered_at, last_getreward_at, created_at, updated_at) VALUES (:id, :last_activated_at, :registered_at, :last_getreward_at, :created_at, :updated_at)",
        $user
    );

    my $user_device = {
        id            => generate_id(),
        user_id       => $user->{id},
        platform_id   => $viewer_id,
        platform_type => $platform_type,
        created_at    => $request_at,
        updated_at    => $request_at,
    };

    $self->db->query(
        "INSERT INTO user_devices(id, user_id, platform_id, platform_type, created_at, updated_at) VALUES (:id, :user_id, :platform_id, :platform_type, :created_at, :updated_at)",
        $user_device
    );

    # 初期デッキ付与
    my $init_card = $self->db->select_row("SELECT * FROM item_masters WHERE id=?", 2);
    unless ($init_card) {
        fail($c, HTTP_NOT_FOUND, ErrItemNotFound);
    }

    my $init_cards = [];
    for (1 .. DECK_CARD_NUMBER) {
        my $card = {
            id             => generate_id(),
            user_id        => $user->{id},
            card_id        => $init_card->{id},
            amount_per_sec => $init_card->{amount_per_sec},
            level          => 1,
            total_exp      => 0,
            created_at     => $request_at,
            updated_at     => $request_at,
        };

        $self->db->query(
            "INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (:id, :user_id, :card_id, :amount_per_sec, :level, :total_exp, :created_at, :updated_at)",
            $card
        );
        push $init_cards->@* => $card;
    }

    my $init_deck = {
        id              => generate_id(),
        user_id         => $user->{id},
        user_card_id_1  => $init_cards->[0]{id},
        user_card_id_2  => $init_cards->[1]{id},
        user_card_id_3  => $init_cards->[2]{id},
        created_at      => $request_at,
        updated_at      => $request_at,
    };

    $self->db->query(
        "INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (:id, :user_id, :user_card_id_1, :user_card_id_2, :user_card_id_3, :created_at, :updated_at)",
        $init_deck
    );

    # ログイン処理
    my ($login_result, $err) = $self->login_process($user->{id}, $request_at);
    if ($err) {
        if ($err eq ErrUserNotFound || $err eq ErrItemNotFound || $err eq ErrLoginBonusRewardNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
        if ($err eq ErrInvalidItemType) {
            fail($c, HTTP_BAD_REQUEST, $err)
        }
    }

    ($user, my $login_bonuses, my $presents) = $login_result->@{qw/user login_bonuses presents/};

    # generate session
    my $session = {
        id         => generate_id(),
        user_id    => $user->{id},
        session_id => generate_uuid(),
        created_at => $request_at,
        updated_at => $request_at,
        expired_at => $request_at + 864000,
    };

    $self->db->query(
        "INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (:id, :user_id, :session_id, :created_at, :updated_at, :expired_at)",
        $session
    );

    $txn->commit;

    return success($c, {
        user_id           => $user->{id},
        viewer_id         => $viewer_id,
        session_id        => $session->{session_id},
        created_at        => $request_at,
        updated_resources => {
            now                => $request_at,
            user               => $user,
            user_device        => $user_device,
            user_cards         => $init_cards,
            user_decks         => [$init_deck],
            user_login_bonuses => $login_bonuses,
            user_presents      => $presents,
        },
    }, CreateUserResponse);
}


use constant LoginResponse => normalize_response_keys {
    viewer_id         => JSON_TYPE_STRING,
    session_id        => JSON_TYPE_STRING,
    updated_resources => UpdatedResource
};

# login ログイン
# POST /login
sub login_handler($self, $c) {
    my $viewer_id = $c->request->body_parameters->{viewerId};
    my $user_id   = $c->request->body_parameters->{userId};
    unless ($viewer_id && $user_id) {
        fail($c, HTTP_BAD_REQUEST, ErrInvalidRequestBody);
    }

    my $request_at = $c->stash->{request_time};

    my $user = $self->db->select_row(
        "SELECT * FROM users WHERE id=?",
        $user_id
    );
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }

    # check ban
    my $is_ban = $self->check_ban($user_id);
    if ($is_ban) {
        fail($c, HTTP_FORBIDDEN, ErrForbidden);
    }

    # viewer id check
    if (my $err = $self->check_viewer_id($user->{id}, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err);
        }
    }

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # sessionを更新
    $self->db->query(
        "UPDATE user_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL",
        $request_at, $user_id
    );

    my $session = {
        id         => generate_id(),
        user_id    => $user_id,
        session_id => generate_uuid(),
        created_at => $request_at,
        updated_at => $request_at,
        expired_at => $request_at + 864000,
    };

    $self->db->query(
        "INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (:id, :user_id, :session_id, :created_at, :updated_at, :expired_at)",
        $session
    );

    # すでにログインしているユーザはログイン処理をしない
    if (is_complete_today_login($user->{last_activated_at}, $request_at)) {
        $user->{updated_at} = $request_at;
        $user->{last_activated_at} = $request_at;

        $self->db->query(
            "UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?",
            $request_at,
            $request_at,
            $user_id,
        );

        $txn->commit;

        return success($c, {
            viewer_id  => $viewer_id,
            session_id => $session->{session_id},
            updated_resources => {
                now  => $request_at,
                user => $user
            }
        }, LoginResponse);
    }
    else {
        my ($login_result, $err) = $self->login_process($user_id, $request_at);
        if ($err) {
            if ($err eq ErrUserNotFound || $err eq ErrItemNotFound || $err eq ErrLoginBonusRewardNotFound) {
                fail($c, HTTP_NOT_FOUND, $err)
            }
            if ($err eq ErrInvalidItemType) {
                fail($c, HTTP_BAD_REQUEST, $err)
            }
        }
        my ($user, $login_bonuses, $presents) = $login_result->@{qw/user login_bonuses presents/};

        $txn->commit;

        return success($c, {
            viewer_id         => $viewer_id,
            session_id        => $session->{session_id},
            updated_resources => {
                now                => $request_at,
                user               => $user,
                user_login_bonuses => $login_bonuses,
                user_presents      => $presents,
            }
        }, LoginResponse);
    }
}

use constant GachaData => normalize_response_keys {
    gacha           => GachaMaster,
    gacha_item_list => json_type_arrayof(GachaItemMaster),
};

use constant ListGachaResponse => normalize_response_keys {
    one_time_token => JSON_TYPE_STRING,
    gachas         => json_type_arrayof(GachaData),
};

# ガチャ一覧
# GET /user/{user_id}/gacha/index
sub list_gacha_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $request_at = $c->stash->{request_time};

    my $gacha_master_list = $self->db->select_all(
        "SELECT * FROM gacha_masters WHERE start_at <= ? AND end_at >= ? ORDER BY display_order ASC",
        $request_at,
        $request_at,
    );
    unless ($gacha_master_list->@*) { # 0件
        return success($c, {
            gachas => [],
        }, ListGachaResponse);
    }

    # ガチャ排出アイテム取得
    my $gacha_data_list = [];
    my $query = "SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC";
    for my $gacha_master ($gacha_master_list->@*) {
        my $gacha_item_list = $self->db->select_all($query, $gacha_master->{id});
        unless ($gacha_item_list->@*) {
            fail($c, HTTP_NOT_FOUND, "not found gacha item");
        }

        push $gacha_data_list->@* => {
            gacha           => $gacha_master,
            gacha_item_list => $gacha_item_list,
        }
    }

    # genearte one time token
    $self->db->query(
        "UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL",
        $request_at,
        $user_id,
    );
    my $token = {
        id         => generate_id(),
        user_id    => $user_id,
        token      => generate_uuid(),
        token_type => 1,
        created_at => $request_at,
        updated_at => $request_at,
        expired_at => $request_at + 600,
    };

    $self->db->query(
        "INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (:id, :user_id, :token, :token_type, :created_at, :updated_at, :expired_at)",
        $token
    );

    return success($c, {
        one_time_token => $token->{token},
        gachas         => $gacha_data_list,
    }, ListGachaResponse);
}

use constant DrawGachaResponse => normalize_response_keys {
    presents => json_type_arrayof(UserPresent),
};

# ガチャを引く
# POST /user/{user_id}/gacha/draw/{gacha_id}/{n}
sub draw_gacha_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $gacha_id = $c->args->{gacha_id};
    my $gacha_count = $c->args->{n};

    if ($gacha_count != 1 && $gacha_count != 10) {
        fail($c, HTTP_BAD_REQUEST, "invalid draw gacha times");
    }

    my $viewer_id = $c->request->body_parameters->{viewerId};
    my $one_time_token = $c->request->body_parameters->{oneTimeToken};
    unless ($viewer_id && $one_time_token) {
        fail($c, HTTP_BAD_REQUEST, ErrInvalidRequestBody);
    }

    my $request_at = $c->stash->{request_time};

    if (my $err = $self->check_one_time_token($one_time_token, 1, $request_at)) {
        if ($err eq ErrInvalidToken) {
            fail($c, HTTP_BAD_REQUEST, $err)
        }
    }

    if (my $err = $self->check_viewer_id($user_id, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
    }

    my $consumed_coin = $gacha_count * 1000;

    # userのisu_coinが足りるか
    my $user = $self->db->select_row(
        "SELECT * FROM users WHERE id=?",
        $user_id
    );
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }
    if ($user->{isu_coin} < $consumed_coin) {
        fail($c, HTTP_CONFLICT, "not enough isucon")
    }

    # gachaIDからガチャマスタの取得
    my $gacha_info = $self->db->select_row(
        "SELECT * FROM gacha_masters WHERE id=? AND start_at <= ? AND end_at >= ?",
        $gacha_id, $request_at, $request_at,
    );
    unless ($gacha_info) {
        fail($c, HTTP_NOT_FOUND, "not found gacha")
    }

    # gachaItemMasterからアイテムリスト取得
    my $gacha_item_list = $self->db->select_all(
        "SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC",
        $gacha_id
    );
    unless ($gacha_item_list->@*) { # 0件
        fail($c, HTTP_NOT_FOUND, "not found gacha items");
    }

    # weightの合計値を算出
    my $sum = $self->db->select_one(
        "SELECT SUM(weight) FROM gacha_item_masters WHERE gacha_id=?",
        $gacha_id,
    );

    # random値の導出 & 抽選
    my $result = [];
    for my $i (0 .. $gacha_count - 1) {
        my $random = int rand $sum;
        my $boundary = 0;
        for my $v ($gacha_item_list->@*) {
            $boundary += $v->{weight};
            if ($random < $boundary) {
                push $result->@* => $v;
                last;
            }
        }
    }

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # 直付与 => プレゼントに入れる
    my $presents = [];
    for my $v ($result->@*) {
        my $present = {
            id              => generate_id(),
            user_id         => $user_id,
            sent_at         => $request_at,
            item_type       => $v->{item_type},
            item_id         => $v->{item_id},
            amount          => $v->{amount},
            present_message => sprintf("%sの付与アイテムです", $gacha_info->{name}),
            created_at      => $request_at,
            updated_at      => $request_at,
        };

        $self->db->query(
            "INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (:id, :user_id, :sent_at, :item_type, :item_id, :amount, :present_message, :created_at, :updated_at)",
            $present
        );

        push $presents->@* => $present;
    }

    # isu_coinをへらす
    my $total_coin = $user->{isu_coin} - $consumed_coin;
    $self->db->query(
        "UPDATE users SET isu_coin=? WHERE id=?",
        $total_coin, $user->{id}
    );

    $txn->commit;

    return success($c, {
        presents => $presents,
    }, DrawGachaResponse);
}


use constant ListPresentResponse => normalize_response_keys {
    presents  => json_type_arrayof(UserPresent),
    is_next   => JSON_TYPE_BOOL,
};

# プレゼント一覧
# GET /user/{user_id}/present/index/{n}
sub list_present_handler($self, $c) {
    my $n = $c->args->{n};
    unless ($n >= 1) {
        fail($c, HTTP_BAD_REQUEST, "index number is more than 1");
    }

    my $user_id = $c->args->{user_id};

    my $offset = PRESENT_COUNT_PER_PAGE * ($n - 1);
    my $query = <<~'SQL';
        SELECT * FROM user_presents
        WHERE user_id = ? AND deleted_at IS NULL
        ORDER BY created_at DESC, id
        LIMIT ? OFFSET ?
    SQL

    my $present_list = $self->db->select_all($query, $user_id, PRESENT_COUNT_PER_PAGE, $offset);

    my $present_count = $self->db->select_one(
        "SELECT COUNT(*) FROM user_presents WHERE user_id = ? AND deleted_at IS NULL",
        $user_id,
    );

    my $is_next = $present_count > ($offset + PRESENT_COUNT_PER_PAGE);

    return success($c, {
        presents => $present_list,
        is_next  => $is_next,
    }, ListPresentResponse);
}

use constant ReceivePresentResponse => normalize_response_keys {
    updated_resources => UpdatedResource
};

# プレゼント受け取り
# POST /user/{user_id}/present/receive
sub receive_present_handler($self, $c) {
    my $viewer_id = $c->request->body_parameters->{viewerId};
    my $present_ids = $c->request->body_parameters_raw->{presentIds};
    unless ($viewer_id && $present_ids) {
        fail($c, HTTP_BAD_REQUEST, 'badrequest');
    }

    my $user_id = $c->args->{user_id};

    my $request_at = $c->stash->{request_time};

    if ($present_ids->@* == 0) {
        fail($c, HTTP_UNPROCESSABLE_ENTITY, "presentIds is empty");
    }

    if (my $err = $self->check_viewer_id($user_id, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
    }

    # user_presentsに入っているが未取得のプレゼント取得
    my $obtain_presents = $self->db->select_all(
        "SELECT * FROM user_presents WHERE id IN (?) AND deleted_at IS NULL",
        $present_ids
    );
    unless ($obtain_presents->@*) { # 0件
        return success($c, {
            updated_resources => {
                now           => $request_at,
                user_presents => [],
            }
        }, ReceivePresentResponse);
    }

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # 配布処理
    for my $op ($obtain_presents->@*) {
        if ($op->{deleted_at}) {
            fail($c, HTTP_INTERNAL_SERVER_ERROR, 'received present')
        }

        $op->{updated_at} = $request_at;
        $op->{deleted_at} = $request_at;

        $self->db->query(
            "UPDATE user_presents SET deleted_at=?, updated_at=? WHERE id=?",
            $request_at, $request_at, $op->{id}
        );

        my (undef, $err) = $self->obtain_item($op->{user_id}, $op->{item_id}, $op->{item_type}, $op->{amount}, $request_at);
        if ($err) {
            if ($err eq ErrUserNotFound || $err eq ErrItemNotFound) {
                fail($c, HTTP_NOT_FOUND, $err)
            }
            if ($err eq ErrInvalidItemType) {
                fail($c, HTTP_BAD_REQUEST, $err)
            }
        }
    }

    $txn->commit;

    return success($c, {
        updated_resources => {
            now           => $request_at,
            user_presents => $obtain_presents,
        }
    }, ReceivePresentResponse);
}


use constant ListItemResponse => normalize_response_keys {
    one_time_token => JSON_TYPE_STRING,
    user           => User,
    items          => json_type_arrayof(UserItem),
    cards          => json_type_arrayof(UserCard),
};

# アイテムリスト
# GET /user/{user_id}/item
sub list_item_handler($self, $c) {
    my $user_id = $c->args->{user_id};

    my $request_at = $c->stash->{request_time};

    my $user = $self->db->select_row(
        "SELECT * FROM users WHERE id=?",
        $user_id
    );
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound);
    }

    my $item_list = $self->db->select_all(
        "SELECT * FROM user_items WHERE user_id = ?",
        $user_id
    );

    my $card_list = $self->db->select_all(
        "SELECT * FROM user_cards WHERE user_id=?",
        $user_id
    );

    # genearte one time token
    $self->db->query(
        "UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL",
        $request_at, $user_id
    );

    my $token = {
        id         => generate_id(),
        user_id    => $user_id,
        token      => generate_uuid(),
        token_type => 2,
        created_at => $request_at,
        updated_at => $request_at,
        expired_at => $request_at + 600,
    };

    $self->db->query(
        "INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (:id, :user_id, :token, :token_type, :created_at, :updated_at, :expired_at)",
        $token
    );

    return success($c, {
        one_time_token => $token->{token},
        items          => $item_list,
        user           => $user,
        cards          => $card_list,
    }, ListItemResponse);
}

use constant AddExpToCardResponse => normalize_response_keys {
    updated_resources => UpdatedResource,
};

# 装備強化
# POST /user/{user_id}/card/addexp/{card_id}
sub add_exp_to_card_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $card_id = $c->args->{card_id};

    my $viewer_id      = $c->request->body_parameters->{viewerId};
    my $one_time_token = $c->request->body_parameters->{oneTimeToken};

    # [ { id => JSON_TYPE_INT, amount => JSON_TYPE_INT } ]
    my $request_items = $c->request->body_parameters_raw->{items};

    my $request_at = $c->stash->{request_time};

    if (my $err = $self->check_one_time_token($one_time_token, 2, $request_at)) {
        if ($err eq ErrInvalidToken) {
            fail($c, HTTP_BAD_REQUEST, $err)
        }
    }

    if (my $err = $self->check_viewer_id($user_id, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
    }

    # get target card
    #
    #    base_amount_per_sec lv1のときの生産性
    #    max_level           最高レベル
    #    max_amount_per_sec  lv maxのときの生産性
    #    base_exp_per_level  lv1 -> lv2に上がるときのexp
    my $query = <<~'SQL';
        SELECT uc.id , uc.user_id , uc.card_id , uc.amount_per_sec , uc.level, uc.total_exp, im.amount_per_sec as 'base_amount_per_sec', im.max_level , im.max_amount_per_sec , im.base_exp_per_level
        FROM user_cards as uc
        INNER JOIN item_masters as im ON uc.card_id = im.id
        WHERE uc.id = ? AND uc.user_id=?
    SQL

    my $card = $self->db->select_row($query, $card_id, $user_id);
    unless ($card) {
        fail($c, HTTP_NOT_FOUND, 'not found card');
    }

    if ($card->{level} == $card->{max_level}) {
        fail($c, HTTP_BAD_REQUEST, "target card is max level")
    }

    # 消費アイテムの所持チェック
    $query = <<~'SQL';
        SELECT ui.id, ui.user_id, ui.item_id, ui.item_type, ui.amount, ui.created_at, ui.updated_at, im.gained_exp
        FROM user_items as ui
        INNER JOIN item_masters as im ON ui.item_id = im.id
        WHERE ui.item_type = 3 AND ui.id=? AND ui.user_id=?
    SQL

    my $items = [];
    for my $v ($request_items->@*) {
        my $item = $self->db->select_row($query, $v->{id}, $user_id);
       unless ($item) {
            fail($c, HTTP_NOT_FOUND, 'not found item')
        }

        if ($v->{amount} > $item->{amount}) {
            fail($c, HTTP_BAD_REQUEST, 'item not enough')
        }

        # 消費量
        $item->{consume_amount} = $v->{amount};
        push $items->@* => $item;
    }

    # 経験値付与
    # 経験値をカードに付与
    for my $v ($items->@*) {
        $card->{total_exp} += $v->{gained_exp} * $v->{consume_amount}
    }

    # lvup判定(lv upしたら生産性を加算)
    while(1) {
        my $next_lv_threshold = int($card->{base_exp_per_level} * (1.2 ** ($card->{level}-1)));
        if ($next_lv_threshold > $card->{total_exp}) {
            last;
        }

        # lv up処理
        $card->{level} += 1;
        $card->{amount_per_sec} += ($card->{max_amount_per_sec} - $card->{base_amount_per_sec}) / ($card->{max_level} - 1);
    }

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # cardのlvと経験値の更新、itemの消費
    $self->db->query(
        "UPDATE user_cards SET amount_per_sec=?, level=?, total_exp=?, updated_at=? WHERE id=?",
        $card->{amount_per_sec}, $card->{level}, $card->{total_exp}, $request_at, $card->{id},
    );

    $query = "UPDATE user_items SET amount=?, updated_at=? WHERE id=?";
    for my $v ($items->@*) {
        $self->db->query($query, $v->{amount} - $v->{consume_amount}, $request_at, $v->{id});
    }

    # get response data
    my $result_card = $self->db->select_row(
        "SELECT * FROM user_cards WHERE id=?",
        $card->{id}
    );
    unless ($result_card) {
        fail($c, HTTP_NOT_FOUND, "not found card")
    }

    my $result_items = [];
    for my $v ($items->@*) {
        push $result_items->@* => {
            id         => $v->{id},
            user_id    => $v->{user_id},
            item_id    => $v->{item_id},
            item_type  => $v->{item_type},
            amount     => $v->{amount} - $v->{consume_amount},
            created_at => $v->{created_at},
            updated_at => $request_at,
        };
    }

    $txn->commit;

    return success($c, {
        updated_resources => {
            now        => $request_at,
            user_cards => [$result_card],
            user_items => $result_items,
        }
    }, AddExpToCardResponse);
}

use constant UpdateDeckResponse => normalize_response_keys {
    updated_resources => UpdatedResource,
};

# 装備変更
# POST /user/{user_id}/card
sub update_deck_handler($self, $c) {
    my $user_id = $c->args->{user_id};

    my $viewer_id = $c->request->body_parameters->{viewerId};
    my $card_ids  = $c->request->body_parameters_raw->{cardIds};

    if ($card_ids->@* != DECK_CARD_NUMBER) {
        fail($c, HTTP_BAD_REQUEST, "invalid number of cards");
    }

    my $request_at = $c->stash->{request_time};

    if (my $err = $self->check_viewer_id($user_id, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
    }

    # カード所持情報のバリデーション
    my $cards = $self->db->select_all(
        "SELECT * FROM user_cards WHERE id IN (?)",
        $card_ids
    );
    unless ($cards->@*) { # 0件
        fail($c, HTTP_NOT_FOUND, 'not found cards')
    }
    unless ($cards->@* == DECK_CARD_NUMBER) {
        fail($c, HTTP_BAD_REQUEST, 'invalid card ids')
    }

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # update data
    $self->db->query(
        "UPDATE user_decks SET updated_at=?, deleted_at=? WHERE user_id=? AND deleted_at IS NULL",
        $request_at, $request_at, $user_id
    );

    my $new_deck = {
        id             => generate_id(),
        user_id        => $user_id,
        user_card_id_1 => $card_ids->[0],
        user_card_id_2 => $card_ids->[1],
        user_card_id_3 => $card_ids->[2],
        created_at     => $request_at,
        updated_at     => $request_at,
    };

    $self->db->query(
        "INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (:id, :user_id, :user_card_id_1, :user_card_id_2, :user_card_id_3, :created_at, :updated_at)",
        $new_deck
    );

    $txn->commit;

    return success($c, {
        updated_resources => {
            now        => $request_at,
            user_decks => [$new_deck],
        }
    }, UpdateDeckResponse);
}

use constant RewardResponse => normalize_response_keys {
    updated_resources => UpdatedResource,
};

# ゲーム報酬受取
# POST /user/{user_id}/reward
sub reward_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $viewer_id = $c->request->body_parameters->{viewerId};
    my $request_at = $c->stash->{request_time};

    if (my $err = $self->check_viewer_id($user_id, $viewer_id)) {
        if ($err eq ErrUserDeviceNotFound) {
            fail($c, HTTP_NOT_FOUND, $err)
        }
    }

    # 最後に取得した報酬時刻取得
    my $user = $self->db->select_row(
        "SELECT * FROM users WHERE id=?",
        $user_id
    );
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }

    # 使っているデッキの取得
    my $deck = $self->db->select_row(
        "SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL",
        $user_id
    );
    unless ($deck) {
        fail($c, HTTP_NOT_FOUND, 'not found deck')
    }

    my $cards = $self->db->select_all(
        "SELECT * FROM user_cards WHERE id IN (?, ?, ?)",
        $deck->{user_card_id_1}, $deck->{user_card_id_2}, $deck->{user_card_id_3}
    );
    unless ($cards->@*) {
        fail($c, HTTP_NOT_FOUND, 'not found cards')
    }
    unless ($cards->@* == DECK_CARD_NUMBER) {
        fail($c, HTTP_BAD_REQUEST, 'invalid cards length')
    }

    # 経過時間*生産性のcoin (1椅子 = 1coin)
    my $past_time = $request_at - $user->{last_getreward_at};
    my $get_coin = int($past_time) * ($cards->[0]->{amount_per_sec} + $cards->[1]->{amount_per_sec} + $cards->[2]->{amount_per_sec});

    # 報酬の保存(ゲームない通貨を保存)(users)
    $user->{isu_coin} += $get_coin;
    $user->{last_getreward_at} = $request_at;

    $self->db->query(
        "UPDATE users SET isu_coin=?, last_getreward_at=? WHERE id=?",
        $user->{isu_coin}, $user->{last_getreward_at}, $user->{id}
    );

    return success($c, {
        updated_resources => {
            now  => $request_at,
            user => $user,
        }
    }, RewardResponse);
}

use constant HomeResponse => normalize_response_keys {
    now                  => JSON_TYPE_INT,
    user                 => User,
    deck                 => json_type_null_or_anyof(UserDeck),
    total_amount_per_sec => JSON_TYPE_INT,
    past_time            => JSON_TYPE_INT, # 経過時間を秒単位で
};

# ホーム取得
# GET /user/{user_id}/home
sub home_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $request_at = $c->stash->{request_time};

    # 装備情報
    my $deck = $self->db->select_row(
        "SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL",
        $user_id,
    );

    # 生産性
    my $cards = [];
    if ($deck) {
        my $card_ids = [$deck->{user_card_id_1}, $deck->{user_card_id_2}, $deck->{user_card_id_3}];
        $cards = $self->db->select_all(
            "SELECT * FROM user_cards WHERE id IN (?)",
            $card_ids,
        );
        unless ($cards->@*) {
            fail($c, HTTP_NOT_FOUND, 'not found cards')
        }
    }
    my $total_amount_per_sec = 0;
    for my $v ($cards->@*) {
        $total_amount_per_sec += $v->{amount_per_sec}
    }

    # 経過時間
    my $user = $self->db->select_row("SELECT * FROM users WHERE id=?", $user_id);
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound);
    }
    my $past_time = $request_at - $user->{last_getreward_at};

    return success($c, {
        now                  => $request_at,
        user                 => $user,
        deck                 => $deck,
        total_amount_per_sec => $total_amount_per_sec,
        past_time            => $past_time,
    }, HomeResponse);
}

1;
