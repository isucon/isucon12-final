USE `isucon`;

DROP TABLE IF EXISTS `admin_sessions`;
DROP TABLE IF EXISTS `user_sessions`;
DROP TABLE IF EXISTS `user_one_time_tokens`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `user_decks`;
DROP TABLE IF EXISTS `user_bans`;
DROP TABLE IF EXISTS `user_devices`;
DROP TABLE IF EXISTS `login_bonus_masters`;
DROP TABLE IF EXISTS `login_bonus_reward_masters`;
DROP TABLE IF EXISTS `user_login_bonuses`;
DROP TABLE IF EXISTS `present_all_masters`;
DROP TABLE IF EXISTS `user_present_all_received_history`;
DROP TABLE IF EXISTS `user_presents`;
DROP TABLE IF EXISTS `gacha_masters`;
DROP TABLE IF EXISTS `gacha_item_masters`;
DROP TABLE IF EXISTS `user_items`;
DROP TABLE IF EXISTS `user_cards`;
DROP TABLE IF EXISTS `item_masters`;
DROP TABLE IF EXISTS `version_masters`;
DROP TABLE IF EXISTS `admin_users`;
DROP TABLE IF EXISTS `id_generator`;

CREATE TABLE `users` (
  `id` bigint NOT NULL,
  `isu_coin` bigint NOT NULL default 0 comment '所持ISU-COIN',
  `last_getreward_at` bigint NOT NULL comment '最後にリワードを取得した日時',
  `last_activated_at` bigint NOT NULL comment '最終アクティブ日時',
  `registered_at` bigint NOT NULL comment '登録日時',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_decks` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID', 
  `user_card_id_1` bigint NOT NULL comment '装備枠1',
  `user_card_id_2` bigint NOT NULL comment '装備枠2',
  `user_card_id_3` bigint NOT NULL comment '装備枠3',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_user_id ( `user_id`,  `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_bans` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID', 
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_user_id (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_devices` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID', 
  `platform_id` varchar(255) NOT NULL comment 'プラットフォームのviewer_id',
  `platform_type` int(1) NOT NULL comment 'PC:1,iOS:2,Android:3', 
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY(`id`),
  UNIQUE uniq_user_id ( `user_id`, `platform_type`, `deleted_at`),
  UNIQUE uniq_platform_id (`platform_id`, `platform_type`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;


/* ログインボーナスマスタ */

CREATE TABLE `login_bonus_masters` (
  `id` bigint NOT NULL,
  `start_at` bigint NOT NULL comment '開始日時',
  `end_at` bigint comment '終了日時。Nullの場合、終了しない。',
  `column_count` int(2) NOT NULL comment '何日分用意するかの日数。例:7日のスタートダッシュ、20日の通常ログイン',
  `looped` boolean NOT NULL comment 'ループするかどうか',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `login_bonus_reward_masters` (
  `id` bigint NOT NULL,
  `login_bonus_id` bigint NOT NULL comment 'ログインボーナスID',
  `reward_sequence` int(2) NOT NULL comment '何日目の報酬か',
  `item_type` int(1) NOT NULL comment '付与するアイテム種別',
  `item_id` int NOT NULL comment '付与するアイテムID',
  `amount` bigint NOT NULL comment '個数',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_login_bonuses` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID', 
  `login_bonus_id` int NOT NULL comment 'ログインボーナスID',
  `last_reward_sequence` int NOT NULL comment '最終受け取り報酬番号',
  `loop_count` int NOT NULL comment 'ループ回数',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_user_id (`user_id`, `login_bonus_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/*  全員プレゼントマスタ */

CREATE TABLE `present_all_masters` (
  `id` bigint NOT NULL,
  `registered_start_at` bigint NOT NULL comment '配布対象のユーザの登録日時の起点日。この日付以降のユーザが対象',
  `registered_end_at` bigint NOT NULL  comment '配布対象のユーザの登録日時の終点日。この日付以前のユーザが対象',
  `item_type` int(1) NOT NULL comment 'アイテム種別',
  `item_id` int NOT NULL comment 'アイテムID',
  `amount` int NOT NULL comment 'アイテム数',
  `present_message` varchar(255) comment 'プレゼント(お詫び)メッセージ',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/* 全員プレゼント履歴テーブル */

CREATE TABLE `user_present_all_received_history` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment '受けとったユーザID',
  `present_all_id` bigint NOT NULL comment '全員プレゼントマスタのID',
  `received_at` bigint NOT NULL comment '受け取った日時',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_presents` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID',
  `sent_at` bigint NOT NULL comment 'プレゼント送付日時',
  `item_type` int(1) NOT NULL comment 'アイテム種別',
  `item_id` int NOT NULL comment 'アイテムID',
  `amount` int NOT NULL comment 'アイテム数',
  `present_message` varchar(255) comment 'プレゼントメッセージ',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  INDEX userid_idx (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/* ガチャマスタ */

CREATE TABLE `gacha_masters` (
  `id` bigint NOT NULL,
  `name` varchar(255) comment 'ガチャ名',
  `start_at` bigint NOT NULL comment '開始日時',
  `end_at` bigint NOT NULL comment '終了日時',
  `display_order` int(2) comment 'ガチャ台の表示順,小さいほど左に表示',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `gacha_item_masters` (
  `id` bigint NOT NULL,
  `gacha_id` bigint NOT NULL comment 'ガチャ台のID',
  `item_type` int(1) NOT NULL comment 'アイテム種別',
  `item_id` int NOT NULL comment 'アイテムID',
  `amount` int NOT NULL comment 'アイテム数',
  `weight` int NOT NULL comment '確率。万分率で表示',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_item_id (`gacha_id`, `item_type`, `item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_items` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID',
  `item_type` int(1) NOT NULL comment 'アイテム種別:1はusersテーブル、2はuser_cardsへ。3,4をこのテーブルへ保存',
  `item_id` int NOT NULL comment 'アイテムID',
  `amount` int NOT NULL comment 'アイテム数',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  INDEX userid_idx (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_cards` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL comment 'ユーザID',
  `card_id` int NOT NULL comment '装備のID',
  `amount_per_sec` int NOT NULL comment '生産性（ISU/sec)',
  `level` int NOT NULL comment 'カードレベル',
  `total_exp` bigint NOT NULL comment '累計経験値',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_card_id (`user_id`, `card_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/*　アイテムマスタ、カードマスタ */

CREATE TABLE `item_masters` (
  `id` bigint NOT NULL,
  `item_type` int(2) NOT NULL comment '1:ISUCOIN、2:ハンマー（カード)、3:強化素材、4:時短アイテム（タイマー）',
  `name` varchar(128) NOT NULL comment 'アイテム名',
  `description` varchar(255) comment 'アイテム説明文',
  `amount_per_sec` int comment 'TYPE2:level1の時の生産性(ISU/sec)',
  `max_level` int comment 'TYPE2:生産性(ISU/sec)',
  `max_amount_per_sec` int comment 'TYPE2:level max時の生産性(ISU/sec)',
  `base_exp_per_level` int comment 'TYP2:level1 -> 2に必要な経験値、以降、前のlevelの1.2倍(切り上げ)必要',
  `gained_exp` int comment 'TYPE3:獲得経験値',
  `shortening_min` bigint comment 'TYPE4:短縮時間(分)',
  -- `created_at` bigint,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;


/*　マスタバージョンを管理するテーブル */
CREATE TABLE `version_masters` (
  `id` bigint NOT NULL,
  `status` int(2) NOT NULL comment 'ステータス 1: available、2:not_available',
  `master_version` varchar(128) NOT NULL comment 'マスタバージョン',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `user_sessions` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL,
  `session_id` varchar(128) NOT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  `expired_at` bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_session_id (`user_id`, `session_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/* 更新処理について利用するone time tokenの管理 */
CREATE TABLE `user_one_time_tokens` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL,
  `token` varchar(128) NOT NULL,
  `token_type` int(2) NOT NULL comment '1:ガチャ用、2:カード強化用',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  `expired_at` bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_token (`user_id`, `token`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

/* 管理者権限のセッション管理 */
CREATE TABLE `admin_sessions` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL,
  `session_id` varchar(128) NOT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  `expired_at` bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`),
  UNIQUE uniq_admin_session_id (`user_id`, `session_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `admin_users` (
  `id` bigint NOT NULL,
  `password` varchar(255) NOT NULL,
  `last_activated_at` bigint NOT NULL comment '最終アクティブ日時',
  `created_at` bigint NOT NULL,
  `updated_at`bigint NOT NULL,
  `deleted_at` bigint default NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `id_generator` (
  `id` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
