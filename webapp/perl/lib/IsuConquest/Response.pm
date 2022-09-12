package IsuConquest::Response;
use v5.36;
use experimental qw(for_list);

use Exporter 'import';

use Carp qw(carp);
use Cpanel::JSON::XS::Type;
use String::CamelCase qw(camelize);

our @EXPORT_OK = qw(
    fail
    success
    normalize_response_keys

    User
    UserDevice
    UserCard
    UserItem
    UserDeck
    UserLoginBonus
    UserPresent
    UserPresentAllReceivedHistory
    Session
    VersionMaster
    ItemMaster
    GachaMaster
    GachaItemMaster
    PresentAllMaster
    LoginBonusRewardMaster
    LoginBonusMaster
);


sub fail($c, $code, $message) {
    carp sprintf("error at %s: code=%d, err=%s", $c->request->uri, $code, $message);

    my $res = $c->render_json({
        status_code => $code,
        message     => $message,
    }, {
        status_code => JSON_TYPE_INT,
        message     => JSON_TYPE_STRING,
    });

    die Kossy::Exception->new($code, response => $res);
}

sub success($c, $data, $spec) {
    my $normalized_data = normalize_response_keys($data);
    $c->render_json($normalized_data, $spec);
}

# JSONレスポンスのキーを正規化する
#
# おこなうこと:
# 1. snake_caseのキーを、lowerCamelCaseのキーに変換する
#   例: { user_id => 123 } -> { userId => 123 }
#
# 2. データとJSONレスポンスのキー名が異なる場合に変換する
#   例: { user_card_id_1 => 123 } -> { cardId1 => 123 }
sub normalize_response_keys($data) {

    state $key_mapping = {
        user_card_id_1 => 'card_id1',
        user_card_id_2 => 'card_id2',
        user_card_id_3 => 'card_id3',
    };

    my $ref = ref $data||'';
    if ($ref eq 'HASH') {
        my $ndata = {};
        for my ($key, $val) ($data->%*) {
            my $nkey = lcfirst(camelize($key_mapping->{$key} // $key));
            my $nval = normalize_response_keys($val);
            $ndata->{$nkey} = $nval;
        }
        return $ndata;
    }
    elsif ($ref eq 'ARRAY') {
        return [ map { normalize_response_keys($_) } $data->@* ];
    }
    else {
        return $data;
    }
}

use constant User => normalize_response_keys {
    id                 => JSON_TYPE_INT,
    isu_coin           => JSON_TYPE_INT,
    last_getreward_at  => JSON_TYPE_INT, # XXX: last_get_reward_atではない
    last_activated_at  => JSON_TYPE_INT,
    registered_at      => JSON_TYPE_INT,
    created_at         => JSON_TYPE_INT,
    updated_at         => JSON_TYPE_INT,
    deleted_at         => JSON_TYPE_INT_OR_NULL,
};

use constant UserDevice => normalize_response_keys {
    id            => JSON_TYPE_INT,
    user_id       => JSON_TYPE_INT,
    platform_id   => JSON_TYPE_STRING,
    platform_type => JSON_TYPE_INT,
    created_at    => JSON_TYPE_INT,
    updated_at    => JSON_TYPE_INT,
    deleted_at    => JSON_TYPE_INT_OR_NULL,
};

use constant UserCard => normalize_response_keys {
    id             => JSON_TYPE_INT,
    user_id        => JSON_TYPE_INT,
    card_id        => JSON_TYPE_INT,
    amount_per_sec => JSON_TYPE_INT,
    level          => JSON_TYPE_INT,
    total_exp      => JSON_TYPE_INT,
    created_at     => JSON_TYPE_INT,
    updated_at     => JSON_TYPE_INT,
    deleted_at     => JSON_TYPE_INT_OR_NULL,
};

use constant UserItem => normalize_response_keys {
    id         => JSON_TYPE_INT,
    user_id    => JSON_TYPE_INT,
    item_type  => JSON_TYPE_INT,
    item_id    => JSON_TYPE_INT,
    amount     => JSON_TYPE_INT,
    created_at => JSON_TYPE_INT,
    updated_at => JSON_TYPE_INT,
    deleted_at => JSON_TYPE_INT_OR_NULL,
};

use constant UserDeck => normalize_response_keys {
    id         => JSON_TYPE_INT,
    user_id    => JSON_TYPE_INT,
    card_id1   => JSON_TYPE_INT, # db: user_card_id_1
    card_id2   => JSON_TYPE_INT, # db: user_card_id_2
    card_id3   => JSON_TYPE_INT, # db: user_card_id_3
    created_at => JSON_TYPE_INT,
    updated_at => JSON_TYPE_INT,
    deleted_at => JSON_TYPE_INT_OR_NULL,
};

use constant UserLoginBonus => normalize_response_keys {
    id                   => JSON_TYPE_INT,
    user_id              => JSON_TYPE_INT,
    login_bonus_id       => JSON_TYPE_INT,
    last_reward_sequence => JSON_TYPE_INT,
    loop_count           => JSON_TYPE_INT,
    created_at           => JSON_TYPE_INT,
    updated_at           => JSON_TYPE_INT,
    deleted_at           => JSON_TYPE_INT_OR_NULL,
};

use constant UserPresent => normalize_response_keys {
    id              => JSON_TYPE_INT,
    user_id         => JSON_TYPE_INT,
    sent_at         => JSON_TYPE_INT,
    item_type       => JSON_TYPE_INT,
    item_id         => JSON_TYPE_INT,
    amount          => JSON_TYPE_INT,
    present_message => JSON_TYPE_STRING,
    created_at      => JSON_TYPE_INT,
    updated_at      => JSON_TYPE_INT,
    deleted_at      => JSON_TYPE_INT_OR_NULL,
};

use constant UserPresentAllReceivedHistory => normalize_response_keys {
    id             => JSON_TYPE_INT,
    user_id        => JSON_TYPE_INT,
    present_all_id => JSON_TYPE_INT,
    received_at    => JSON_TYPE_INT,
    created_at     => JSON_TYPE_INT,
    updated_at     => JSON_TYPE_INT,
    deleted_at     => JSON_TYPE_INT_OR_NULL,
};

use constant Session => normalize_response_keys {
    id         => JSON_TYPE_INT,
    user_id    => JSON_TYPE_INT,
    session_id => JSON_TYPE_STRING,
    created_at => JSON_TYPE_INT,
    updated_at => JSON_TYPE_INT,
    expired_at => JSON_TYPE_INT,
    deleted_at => JSON_TYPE_INT_OR_NULL,
};

use constant VersionMaster => normalize_response_keys {
    id            => JSON_TYPE_INT,
    status        => JSON_TYPE_INT,
    master_version => JSON_TYPE_STRING,
};

use constant ItemMaster => normalize_response_keys {
    id                 => JSON_TYPE_INT,
    item_type          => JSON_TYPE_INT,
    name               => JSON_TYPE_STRING,
    description        => JSON_TYPE_STRING,
    amount_per_sec     => JSON_TYPE_INT_OR_NULL,
    max_level          => JSON_TYPE_INT_OR_NULL,
    max_amount_per_sec => JSON_TYPE_INT_OR_NULL,
    base_exp_per_level => JSON_TYPE_INT_OR_NULL,
    gained_exp         => JSON_TYPE_INT_OR_NULL,
    shortening_min     => JSON_TYPE_INT_OR_NULL,
    # created_at         => JSON_TYPE_INT,
};

use constant GachaMaster => normalize_response_keys {
    id            => JSON_TYPE_INT,
    name          => JSON_TYPE_STRING,
    start_at      => JSON_TYPE_INT,
    end_at        => JSON_TYPE_INT,
    display_order => JSON_TYPE_INT,
    created_at    => JSON_TYPE_INT,
};

use constant GachaItemMaster => normalize_response_keys {
    id         => JSON_TYPE_INT,
    gacha_id   => JSON_TYPE_INT,
    item_type  => JSON_TYPE_INT,
    item_id    => JSON_TYPE_INT,
    amount     => JSON_TYPE_INT,
    weight     => JSON_TYPE_INT,
    created_at => JSON_TYPE_INT,
};

use constant PresentAllMaster => normalize_response_keys {
    id                  => JSON_TYPE_INT,
    registered_start_at => JSON_TYPE_INT,
    registered_end_at   => JSON_TYPE_INT,
    item_type           => JSON_TYPE_INT,
    item_id             => JSON_TYPE_INT,
    amount              => JSON_TYPE_INT,
    present_message     => JSON_TYPE_STRING,
    created_at          => JSON_TYPE_INT,
};

use constant LoginBonusRewardMaster => normalize_response_keys {
    id              => JSON_TYPE_INT,
    login_bonus_id  => JSON_TYPE_INT,
    reward_sequence => JSON_TYPE_INT,
    item_type       => JSON_TYPE_INT,
    item_id         => JSON_TYPE_INT,
    amount          => JSON_TYPE_INT,
    created_at      => JSON_TYPE_INT,
};

use constant LoginBonusMaster => normalize_response_keys {
    id           => JSON_TYPE_INT,
    start_at     => JSON_TYPE_INT,
    end_at       => JSON_TYPE_INT,
    column_count => JSON_TYPE_INT,
    looped       => JSON_TYPE_BOOL,
    created_at   => JSON_TYPE_INT,
};

1;
