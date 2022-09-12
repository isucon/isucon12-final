package IsuConquest::Admin;
use v5.36;
use utf8;
use experimental qw(defer builtin for_list);
use builtin qw(indexed);

use Kossy;
use HTTP::Status qw(:constants);
use HTTP::Date qw(str2time);

use Cpanel::JSON::XS;
use Cpanel::JSON::XS::Type;

use Crypt::Bcrypt qw/bcrypt_check/;
use Text::CSV_XS;
use SQL::Maker;
SQL::Maker->load_plugin('InsertMulti');

$Kossy::JSON_SERIALIZER = Cpanel::JSON::XS->new()->ascii(0)->canonical;

use constant DEBUG => 1;

use IsuConquest::ErrorMessage qw(
    ErrExpiredSession
    ErrUserNotFound
    ErrUserDeviceNotFound
    ErrNoFormFile
    ErrUnauthorized
);

use IsuConquest::Response qw(
    fail
    success
    normalize_response_keys

    Session
    VersionMaster
    ItemMaster
    GachaMaster
    GachaItemMaster
    PresentAllMaster
    LoginBonusMaster
    LoginBonusRewardMaster
    User
    UserDevice
    UserCard
    UserDeck
    UserItem
    UserLoginBonus
    UserPresent
    UserPresentAllReceivedHistory
);

use IsuConquest::IDGenerator qw(generate_id generate_uuid);
use IsuConquest::DBConnector qw(connect_db);

sub db($self) {
    $self->{dbh} ||= connect_db
}

filter 'admin_filter'               => \&admin_filter;
filter 'admin_check_session_filter' => \&admin_check_session_filter;
filter 'allow_json_request_filter'  => \&allow_json_request_filter;

my $admin_filters = [qw/allow_json_request_filter admin_filter/];
router 'POST', '/login' => $admin_filters, \&admin_login_handler;

my $admin_session_filters = [$admin_filters->@*, 'admin_check_session_filter'];
router 'DELETE', '/logout'                 => $admin_session_filters, \&admin_logout_handler;
router 'GET',    '/master'                 => $admin_session_filters, \&admin_list_master_handler;
router 'PUT',    '/master',                => $admin_session_filters, \&admin_update_master_handler;
router 'GET',    '/user/{user_id:\d+}'     => $admin_session_filters, \&admin_user_handler;
router 'POST',   '/user/{user_id:\d+}/ban' => $admin_session_filters, \&admin_ban_user_handler;

sub allow_json_request_filter($app) {
    return sub ($self, $c) {
        $c->env->{'kossy.request.parse_json_body'} = 1;
        $app->($self, $c);
    };
}

sub admin_filter($app) {
    return sub ($self, $c) {
        my $request_at = str2time( $c->request->header('x-isu-date') );
        unless ($request_at) {
            $request_at = time;
        }
        $c->stash->{request_time} = $request_at;

        $app->($self, $c);
    }
}

sub admin_check_session_filter($app) {
    return sub ($self, $c) {
        my $session_id = $c->request->header('x-session');

        my $admin_session = $self->db->select_row(
            "SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL",
            $session_id
        );
        unless ($admin_session) {
            fail($c, HTTP_UNAUTHORIZED, ErrUnauthorized);
        }

        my $request_at = time;

        if ($admin_session->{expired_at} < $request_at) {
            $self->db->query(
                "UPDATE admin_sessions SET deleted_at=? WHERE session_id=?",
                $request_at, $session_id
            );
            fail($c, HTTP_UNAUTHORIZED, ErrExpiredSession);
        }

        $app->($self, $c);
    }
}

use constant AdminLoginResponse => normalize_response_keys {
    session => Session
};

# 管理者権限ログイン
# POST /admin/login
sub admin_login_handler($self, $c) {
    my $user_id = $c->request->body_parameters->{userId};
    my $password = $c->request->body_parameters->{password};
    unless ($user_id && $password) {
        fail($c, HTTP_BAD_REQUEST, 'bad request');
    }

    my $request_at = time;

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    # userの存在確認
    my $user = $self->db->select_row(
        "SELECT * FROM admin_users WHERE id=?",
        $user_id
    );
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }

    # verify password
    if (my $err = verify_password($user->{password}, $password)) {
        fail($c, HTTP_UNAUTHORIZED, $err)
    }

    $self->db->query(
        "UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?",
        $request_at, $request_at, $user_id
    );

    # すでにあるsessionをdeleteにする
    $self->db->query(
        "UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL",
        $request_at, $user_id
    );

    # create session
    my $session = {
        id         => generate_id(),
        user_id    => $user_id,
        session_id => generate_uuid(),
        created_at => $request_at,
        updated_at => $request_at,
        expired_at => $request_at + 86400,
    };

    $self->db->query(
        "INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (:id, :user_id, :session_id, :created_at, :updated_at, :expired_at)",
        $session
    );

    $self->db->commit;

    return success($c, {
        session => $session
    }, AdminLoginResponse);
}

# 管理者権限ログアウト
# DELETE /admin/logout
sub admin_logout_handler($self, $c) {
    my $session_id = $c->request->header('x-session');

    my $request_at = time;

    # すでにあるsessionをdeleteにする
    $self->db->query(
        "UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL",
        $request_at, $session_id
    );

    $c->halt_no_content(HTTP_NO_CONTENT)
}

use constant AdminListMasterResponse => normalize_response_keys {
    version_master      => json_type_arrayof(VersionMaster),
    items               => json_type_arrayof(ItemMaster),
    gachas              => json_type_arrayof(GachaMaster),
    gacha_items         => json_type_arrayof(GachaItemMaster),
    present_alls        => json_type_arrayof(PresentAllMaster),
    login_bonuses       => json_type_arrayof(LoginBonusMaster),
    login_bonus_rewards => json_type_arrayof(LoginBonusRewardMaster),
};

# マスタデータ閲覧
# GET /admin/master
sub admin_list_master_handler($self, $c) {
    my $version_masters     = $self->db->select_all("SELECT * FROM version_masters");
    my $items               = $self->db->select_all("SELECT * FROM item_masters");
    my $gachas              = $self->db->select_all("SELECT * FROM gacha_masters");
    my $gacha_items         = $self->db->select_all("SELECT * FROM gacha_item_masters");
    my $present_alls        = $self->db->select_all("SELECT * FROM present_all_masters");
    my $login_bonuses       = $self->db->select_all("SELECT * FROM login_bonus_masters");
    my $login_bonus_rewards = $self->db->select_all("SELECT * FROM login_bonus_reward_masters");

    return success($c, {
        version_master      => $version_masters,
        items               => $items,
        gachas              => $gachas,
        gacha_items         => $gacha_items,
        present_alls        => $present_alls,
        login_bonuses       => $login_bonuses,
        login_bonus_rewards => $login_bonus_rewards,
    }, AdminListMasterResponse);
}

use constant AdminUpdateMasterResponse => normalize_response_keys {
    version_master => VersionMaster,
};

# マスタデータ更新
# PUT /admin/master
sub admin_update_master_handler($self, $c) {

    state $query_builder = SQL::Maker->new(driver => 'mysql');

    my $txn = $self->db->txn_scope;
    defer { $txn->rollback }

    my $err;

    # version master
    (my $version_recs, $err) = read_form_file_to_csv($c, 'versionMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($version_recs) {
        my $data = [];
        for my ($i, $v) (indexed $version_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id             => $v->[0],
                status         => $v->[1],
                master_version => $v->[2],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(version_masters => $data, {
                update => { status => \'VALUES(status)', master_version => \'VALUES(master_version)' }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: versionMaster" if DEBUG;
    }

    # item
    (my $item_recs, $err) = read_form_file_to_csv($c, 'itemMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($item_recs) {
        my $data = [];
        for my ($i, $v) (indexed $item_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id                 => $v->[0],
                item_type          => $v->[1],
                name               => $v->[2],
                description        => $v->[3],
                amount_per_sec     => $v->[4],
                max_level          => $v->[5],
                max_amount_per_sec => $v->[6],
                base_exp_per_level => $v->[7],
                gained_exp         => $v->[8],
                shortening_min     => $v->[9],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(item_masters => $data, {
                update => {
                    item_type          => \'VALUES(item_type)',
                    name               => \'VALUES(name)',
                    description        => \'VALUES(description)',
                    amount_per_sec     => \'VALUES(amount_per_sec)',
                    max_level          => \'VALUES(max_level)',
                    max_amount_per_sec => \'VALUES(max_amount_per_sec)',
                    base_exp_per_level => \'VALUES(base_exp_per_level)',
                    gained_exp         => \'VALUES(gained_exp)',
                    shortening_min     => \'VALUES(shortening_min)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: itemMaster" if DEBUG;
    }

    # gacha
    (my $gacha_recs, $err) = read_form_file_to_csv($c, 'gachaMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($gacha_recs) {
        my $data = [];
        for my ($i, $v) (indexed $gacha_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id            => $v->[0],
                name          => $v->[1],
                start_at      => $v->[2],
                end_at        => $v->[3],
                display_order => $v->[4],
                created_at    => $v->[5],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(gacha_masters => $data, {
                update => {
                    name          => \'VALUES(name)',
                    start_at      => \'VALUES(start_at)',
                    end_at        => \'VALUES(end_at)',
                    display_order => \'VALUES(display_order)',
                    created_at    => \'VALUES(created_at)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: gachaMaster" if DEBUG;
    }

    # gacha_item
    (my $gacha_item_recs, $err) = read_form_file_to_csv($c, 'gachaItemMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($gacha_item_recs) {
        my $data = [];
        for my ($i, $v) (indexed $gacha_item_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id         => $v->[0],
                gacha_id   => $v->[1],
                item_type  => $v->[2],
                item_id    => $v->[3],
                amount     => $v->[4],
                weight     => $v->[5],
                created_at => $v->[6],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(gacha_item_masters => $data, {
                update => {
                    gacha_id   => \'VALUES(gacha_id)',
                    item_type  => \'VALUES(item_type)',
                    item_id    => \'VALUES(item_id)',
                    amount     => \'VALUES(amount)',
                    weight     => \'VALUES(weight)',
                    created_at => \'VALUES(created_at)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: gachaItemMaster" if DEBUG;
    }

    # present_all
    (my $present_all_recs, $err) = read_form_file_to_csv($c, 'presentAllMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($present_all_recs) {
        my $data = [];
        for my ($i, $v) (indexed $present_all_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id                  => $v->[0],
                registered_start_at => $v->[1],
                registered_end_at   => $v->[2],
                item_type           => $v->[3],
                item_id             => $v->[4],
                amount              => $v->[5],
                present_message     => $v->[6],
                created_at          => $v->[7],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(present_all_masters => $data, {
                update => {
                    registered_start_at => \'VALUES(registered_start_at)',
                    registered_end_at   => \'VALUES(registered_end_at)',
                    item_type           => \'VALUES(item_type)',
                    item_id             => \'VALUES(item_id)',
                    amount              => \'VALUES(amount)',
                    present_message     => \'VALUES(present_message)',
                    created_at          => \'VALUES(created_at)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: presentAllMaster" if DEBUG;
    }

    # login_bonus
    (my $login_bonus_recs, $err) = read_form_file_to_csv($c, 'loginBonusMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($login_bonus_recs) {
        my $data = [];
        for my ($i, $v) (indexed $login_bonus_recs->@*) {
            next if $i == 0;
            my $looped = $v->[4] eq 'TRUE';
            push $data->@* => {
                id           => $v->[0],
                start_at     => $v->[1],
                end_at       => $v->[2],
                column_count => $v->[3],
                looped       => $looped,
                created_at   => $v->[5],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(login_bonus_masters => $data, {
                update => {
                    start_at     => \'VALUES(start_at)',
                    end_at       => \'VALUES(end_at)',
                    column_count => \'VALUES(column_count)',
                    looped       => \'VALUES(looped)',
                    created_at   => \'VALUES(created_at)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: loginBonusMaster" if DEBUG;
    }

    # login_bonus_reward
    (my $login_bonus_reward_recs, $err) = read_form_file_to_csv($c, 'loginBonusRewardMaster');
    if ($err && $err ne ErrNoFormFile) {
        fail($c, HTTP_BAD_REQUEST, $err)
    }
    if ($login_bonus_reward_recs) {
        my $data = [];
        for my ($i, $v) (indexed $login_bonus_reward_recs->@*) {
            next if $i == 0;
            push $data->@* => {
                id              => $v->[0],
                login_bonus_id  => $v->[1],
                reward_sequence => $v->[2],
                item_type       => $v->[3],
                item_id         => $v->[4],
                amount          => $v->[5],
                created_at      => $v->[6],
            };
        }

        if ($data->@*) {
            my ($query, @binds) = $query_builder->insert_multi(login_bonus_reward_masters => $data, {
                update => {
                    login_bonus_id  => \'VALUES(login_bonus_id)',
                    reward_sequence => \'VALUES(reward_sequence)',
                    item_type       => \'VALUES(item_type)',
                    item_id         => \'VALUES(item_id)',
                    amount          => \'VALUES(amount)',
                    created_at      => \'VALUES(created_at)',
                }
            });
            $self->db->query($query, @binds);
        }
    }
    else {
        warn "Skip Update Master: loginBonusRewardMaster" if DEBUG;
    }

    my $active_master = $self->db->select_row("SELECT * FROM version_masters WHERE status=1");
    unless ($active_master) {
        fail($c, HTTP_INTERNAL_SERVER_ERROR, 'invalid active_master')
    }

    $txn->commit;

    return success($c, {
        version_master => $active_master
    }, AdminUpdateMasterResponse)
}

# ファイルからcsvレコードを取得する
sub read_form_file_to_csv($c, $name) {
    my $file = $c->request->uploads->{$name};
    unless ($file) {
        return undef, ErrNoFormFile
    }

    open my $fh, '<', $file->path
        or return undef, $!;

    state $csv = Text::CSV_XS->new({binary => 1});
    my $records = $csv->getline_all($fh);

    return $records, undef;
}

use constant AdminUserResponse => normalize_response_keys {
    user                              => User,
    user_devices                      => json_type_arrayof(UserDevice),
    user_cards                        => json_type_arrayof(UserCard),
    user_decks                        => json_type_arrayof(UserDeck),
    user_items                        => json_type_arrayof(UserItem),
    user_login_bonuses                => json_type_arrayof(UserLoginBonus),
    user_presents                     => json_type_arrayof(UserPresent),
    user_present_all_received_history => json_type_arrayof(UserPresentAllReceivedHistory),
};

# ユーザの詳細画面
# GET /admin/user/{user_id}
sub admin_user_handler ($self, $c) {
    my $user_id = $c->args->{user_id};

    my $user = $self->db->select_row("SELECT * FROM users WHERE id=?", $user_id);
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }

    my $devices = $self->db->select_all("SELECT * FROM user_devices WHERE user_id=?", $user_id);
    unless ($devices->@*) {
        fail($c, HTTP_NOT_FOUND, ErrUserDeviceNotFound)
    }

    my $cards           = $self->db->select_all("SELECT * FROM user_cards WHERE user_id=?", $user_id);
    my $decks           = $self->db->select_all("SELECT * FROM user_decks WHERE user_id=?", $user_id);
    my $items           = $self->db->select_all("SELECT * FROM user_items WHERE user_id=?", $user_id);
    my $login_bonuses   = $self->db->select_all("SELECT * FROM user_login_bonuses WHERE user_id=?", $user_id);
    my $presents        = $self->db->select_all("SELECT * FROM user_presents WHERE user_id=?", $user_id);
    my $present_history = $self->db->select_all("SELECT * FROM user_present_all_received_history WHERE user_id=?", $user_id);

    return success($c, {
        user                              => $user,
        user_devices                      => $devices,
        user_cards                        => $cards,
        user_decks                        => $decks,
        user_items                        => $items,
        user_login_bonuses                => $login_bonuses,
        user_presents                     => $presents,
        user_present_all_received_history => $present_history,
    }, AdminUserResponse);
}

use constant AdminBanUserResponse => normalize_response_keys {
    user => User,
};

# ユーザBAN処理
# POST /admin/user/{user_id}/ban
sub admin_ban_user_handler($self, $c) {
    my $user_id = $c->args->{user_id};
    my $request_at = time;

    my $user = $self->db->select_row("SELECT * FROM users WHERE id=?", $user_id);
    unless ($user) {
        fail($c, HTTP_NOT_FOUND, ErrUserNotFound)
    }

    my $ban_id = generate_id();
    $self->db->query(
        "INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?",
        $ban_id, $user_id, $request_at, $request_at, $request_at
    );

    return success($c, {
        user => $user,
    }, AdminBanUserResponse)
}

sub verify_password($hash, $password) {
    unless (bcrypt_check($password, $hash)) {
        return ErrUnauthorized
    }
    return;
}

1;
