use actix_web::web;
use bytes::BytesMut;
use futures_util::TryStreamExt as _;
use itertools::Itertools as _;
use sqlx::mysql::MySqlArguments;
use sqlx::Arguments as _;
use sqlx::MySqlPool;

// //////////////////////////////////////
// admin

pub struct AdminSessionCheckMiddlewareFactory;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest>
    for AdminSessionCheckMiddlewareFactory
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::Error,
    >,
    S::Future: 'static,
    B: 'static,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type InitError = ();
    type Transform = AdminSessionCheckMiddleware<S>;
    type Future = std::future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        std::future::ready(Ok(AdminSessionCheckMiddleware { service }))
    }
}
pub struct AdminSessionCheckMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest>
    for AdminSessionCheckMiddleware<S>
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::Error,
    >,
    S::Future: 'static,
    B: 'static,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future =
        futures_util::future::LocalBoxFuture<'static, Result<Self::Response, Self::Error>>;

    actix_web::dev::forward_ready!(service);

    fn call(&self, mut req: actix_web::dev::ServiceRequest) -> Self::Future {
        let sess_id = req
            .headers()
            .get("x-session")
            .map(|v| v.to_str().unwrap().to_owned())
            .unwrap_or_default();

        let pool_fut = req.extract::<web::Data<MySqlPool>>();
        let request_at_fut = req.extract::<crate::RequestTime>();
        let fut = self.service.call(req);

        Box::pin(async move {
            let pool = pool_fut.await?;
            let query = "SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL";
            let admin_session: crate::Session = sqlx::query_as(query)
                .bind(&sess_id)
                .fetch_optional(&**pool)
                .await
                .map_err(crate::Error::Sqlx)?
                .ok_or(crate::Error::Unauthorized)?;

            let request_at = request_at_fut.await?.0;

            if admin_session.expired_at < request_at {
                let query = "UPDATE admin_sessions SET deleted_at=? WHERE session_id=?";
                sqlx::query(query)
                    .bind(request_at)
                    .bind(sess_id)
                    .execute(&**pool)
                    .await
                    .map_err(crate::Error::Sqlx)?;
                return Err(crate::Error::ExpiredSession.into());
            }

            // next
            fut.await
        })
    }
}

/// 管理者権限ログイン
/// POST /admin/login
pub async fn admin_login(
    req: web::Json<AdminLoginRequest>,
    request_at: crate::RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AdminLoginResponse>, crate::Error> {
    let req = req.into_inner();
    let request_at = request_at.0;

    let mut tx = pool.begin().await?;

    // userの存在確認
    let query = "SELECT * FROM admin_users WHERE id=?";
    let user: AdminUser = sqlx::query_as(query)
        .bind(req.user_id)
        .fetch_optional(&mut tx)
        .await?
        .ok_or(crate::Error::UserNotFound)?;

    // verify password
    verify_password(&user.password, &req.password)?;

    let query = "UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?";
    sqlx::query(query)
        .bind(request_at)
        .bind(request_at)
        .bind(req.user_id)
        .execute(&mut tx)
        .await?;

    // すでにあるsessionをdeleteにする
    let query = "UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(req.user_id)
        .execute(&mut tx)
        .await?;

    // create session
    let s_id = crate::generate_id(&pool).await?;
    let sess_id = crate::generate_uuid();
    let sess = crate::Session {
        id: s_id,
        user_id: req.user_id,
        session_id: sess_id,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 86400,
        deleted_at: None,
    };

    let query = "INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(sess.id)
        .bind(sess.user_id)
        .bind(&sess.session_id)
        .bind(sess.created_at)
        .bind(sess.updated_at)
        .bind(sess.expired_at)
        .execute(&mut tx)
        .await?;

    tx.commit().await?;

    Ok(web::Json(AdminLoginResponse {
        admin_session: sess,
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminLoginRequest {
    user_id: i64,
    password: String,
}

#[derive(Debug, serde::Serialize)]
pub struct AdminLoginResponse {
    #[serde(rename = "session")]
    admin_session: crate::Session,
}

/// 管理者権限ログアウト
/// DELETE /admin/logout
pub async fn admin_logout(
    request: actix_web::HttpRequest,
    request_at: crate::RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<actix_web::HttpResponse, crate::Error> {
    let request_at = request_at.0;

    let sess_id = request
        .headers()
        .get("x-session")
        .unwrap()
        .to_str()
        .unwrap();
    // すでにあるsessionをdeleteにする
    let query = "UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(sess_id)
        .execute(&**pool)
        .await?;

    Ok(actix_web::HttpResponse::NoContent().body(()))
}

/// マスタデータ閲覧
/// GET /admin/master
pub async fn admin_list_master(
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AdminListMasterResponse>, crate::Error> {
    let master_versions: Vec<crate::VersionMaster> =
        sqlx::query_as("SELECT * FROM version_masters")
            .fetch_all(&**pool)
            .await?;

    let items: Vec<crate::ItemMaster> = sqlx::query_as("SELECT * FROM item_masters")
        .fetch_all(&**pool)
        .await?;

    let gachas: Vec<crate::GachaMaster> = sqlx::query_as("SELECT * FROM gacha_masters")
        .fetch_all(&**pool)
        .await?;

    let gacha_items: Vec<crate::GachaItemMaster> =
        sqlx::query_as("SELECT * FROM gacha_item_masters")
            .fetch_all(&**pool)
            .await?;

    let present_alls: Vec<crate::PresentAllMaster> =
        sqlx::query_as("SELECT * FROM present_all_masters")
            .fetch_all(&**pool)
            .await?;

    let login_bonuses: Vec<crate::LoginBonusMaster> =
        sqlx::query_as("SELECT * FROM login_bonus_masters")
            .fetch_all(&**pool)
            .await?;

    let login_bonus_rewards: Vec<crate::LoginBonusRewardMaster> =
        sqlx::query_as("SELECT * FROM login_bonus_reward_masters")
            .fetch_all(&**pool)
            .await?;

    Ok(web::Json(AdminListMasterResponse {
        version_master: master_versions,
        items,
        gachas,
        gacha_items,
        present_alls,
        login_bonuses,
        login_bonus_rewards,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminListMasterResponse {
    version_master: Vec<crate::VersionMaster>,
    items: Vec<crate::ItemMaster>,
    gachas: Vec<crate::GachaMaster>,
    gacha_items: Vec<crate::GachaItemMaster>,
    present_alls: Vec<crate::PresentAllMaster>,
    login_bonus_rewards: Vec<crate::LoginBonusRewardMaster>,
    login_bonuses: Vec<crate::LoginBonusMaster>,
}

/// マスタデータ更新
/// PUT /admin/master
pub async fn admin_update_master(
    mut payload: actix_multipart::Multipart,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AdminUpdateMasterResponse>, crate::Error> {
    let mut tx = pool.begin().await?;

    while let Some(item) = payload.try_next().await? {
        let name = item.content_disposition().get_name().unwrap();
        match name {
            "versionMaster" => {
                // version master
                let version_master_recs = read_form_file_to_csv(item).await?;
                let placeholders = version_master_recs.iter().map(|_| "(?, ?, ?)").join(",");
                let query = format!(
                    "INSERT INTO version_masters(id, status, master_version) VALUES {} ON DUPLICATE KEY UPDATE status=VALUES(status), master_version=VALUES(master_version)",
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in version_master_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "itemMaster" => {
                // item
                let item_master_recs = read_form_file_to_csv(item).await?;
                let placeholders = item_master_recs
                    .iter()
                    .map(|_| "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
                    .join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO item_masters(id, item_type, name, description, amount_per_sec, max_level, max_amount_per_sec, base_exp_per_level, gained_exp, shortening_min)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE item_type=VALUES(item_type), name=VALUES(name), description=VALUES(description), amount_per_sec=VALUES(amount_per_sec), max_level=VALUES(max_level), max_amount_per_sec=VALUES(max_amount_per_sec), base_exp_per_level=VALUES(base_exp_per_level), gained_exp=VALUES(gained_exp), shortening_min=VALUES(shortening_min)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in item_master_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(&v[4]);
                    args.add(&v[5]);
                    args.add(&v[6]);
                    args.add(&v[7]);
                    args.add(&v[8]);
                    args.add(&v[9]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "gachaMaster" => {
                // gacha
                let gacha_recs = read_form_file_to_csv(item).await?;
                let placeholders = gacha_recs.iter().map(|_| "(?, ?, ?, ?, ?, ?)").join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in gacha_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(&v[4]);
                    args.add(&v[5]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "gachaItemMaster" => {
                // gacha item
                let gacha_item_recs = read_form_file_to_csv(item).await?;
                let placeholders = gacha_item_recs
                    .iter()
                    .map(|_| "(?, ?, ?, ?, ?, ?, ?)")
                    .join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO gacha_item_masters(id, gacha_id, item_type, item_id, amount, weight, created_at)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE gacha_id=VALUES(gacha_id), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), weight=VALUES(weight), created_at=VALUES(created_at)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in gacha_item_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(&v[4]);
                    args.add(&v[5]);
                    args.add(&v[6]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "presentAllMaster" => {
                // present all
                let present_all_recs = read_form_file_to_csv(item).await?;
                let placeholders = present_all_recs
                    .iter()
                    .map(|_| "(?, ?, ?, ?, ?, ?, ?, ?)")
                    .join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO present_all_masters(id, registered_start_at, registered_end_at, item_type, item_id, amount, present_message, created_at)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE registered_start_at=VALUES(registered_start_at), registered_end_at=VALUES(registered_end_at), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), present_message=VALUES(present_message), created_at=VALUES(created_at)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in present_all_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(&v[4]);
                    args.add(&v[5]);
                    args.add(&v[6]);
                    args.add(&v[7]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "loginBonusMaster" => {
                // login bonuses
                let login_bonus_recs = read_form_file_to_csv(item).await?;
                let placeholders = login_bonus_recs
                    .iter()
                    .map(|_| "(?, ?, ?, ?, ?, ?)")
                    .join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO login_bonus_masters(id, start_at, end_at, column_count, looped, created_at)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE start_at=VALUES(start_at), end_at=VALUES(end_at), column_count=VALUES(column_count), looped=VALUES(looped), created_at=VALUES(created_at)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in login_bonus_recs {
                    let looped = &v[4] == "TRUE";
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(looped);
                    args.add(&v[5]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            "loginBonusRewardMaster" => {
                // login bonus rewards
                let login_bonus_reward_recs = read_form_file_to_csv(item).await?;
                let placeholders = login_bonus_reward_recs
                    .iter()
                    .map(|_| "(?, ?, ?, ?, ?, ?, ?)")
                    .join(",");
                let query = format!(
                    concat!(
                        "INSERT INTO login_bonus_reward_masters(id, login_bonus_id, reward_sequence, item_type, item_id, amount, created_at)",
                        "VALUES {}",
                        "ON DUPLICATE KEY UPDATE login_bonus_id=VALUES(login_bonus_id), reward_sequence=VALUES(reward_sequence), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), created_at=VALUES(created_at)",
                    ),
                    placeholders
                );
                let mut args = MySqlArguments::default();
                for v in login_bonus_reward_recs {
                    args.add(&v[0]);
                    args.add(&v[1]);
                    args.add(&v[2]);
                    args.add(&v[3]);
                    args.add(&v[4]);
                    args.add(&v[5]);
                    args.add(&v[6]);
                }
                sqlx::query_with(&query, args).execute(&mut tx).await?;
            }
            _ => {}
        }
    }

    let active_master: crate::VersionMaster =
        sqlx::query_as("SELECT * FROM version_masters WHERE status=1")
            .fetch_one(&mut tx)
            .await?;

    tx.commit().await?;

    Ok(web::Json(AdminUpdateMasterResponse {
        version_master: active_master,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminUpdateMasterResponse {
    version_master: crate::VersionMaster,
}

/// ファイルからcsvレコードを取得する
async fn read_form_file_to_csv(
    field: actix_multipart::Field,
) -> Result<Vec<csv::StringRecord>, crate::Error> {
    let src = field
        .map_ok(|chunk| BytesMut::from(&chunk[..]))
        .try_concat()
        .await?;

    let reader = csv::ReaderBuilder::new().from_reader(src.as_ref());
    let mut records = Vec::new();
    for r in reader.into_records() {
        records.push(r?);
    }
    Ok(records)
}

/// ユーザの詳細画面
/// GET /admin/user/{userId}
pub async fn admin_user(
    path: web::Path<(i64,)>,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AdminUserResponse>, crate::Error> {
    let (user_id,) = path.into_inner();

    let query = "SELECT * FROM users WHERE id=?";
    let user: crate::User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(crate::Error::UserNotFound)?;

    let query = "SELECT * FROM user_devices WHERE user_id=?";
    let devices: Vec<crate::UserDevice> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_cards WHERE user_id=?";
    let cards: Vec<crate::UserCard> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_decks WHERE user_id=?";
    let decks: Vec<crate::UserDeck> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_items WHERE user_id=?";
    let items: Vec<crate::UserItem> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_login_bonuses WHERE user_id=?";
    let login_bonuses: Vec<crate::UserLoginBonus> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_presents WHERE user_id=?";
    let presents: Vec<crate::UserPresent> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_present_all_received_history WHERE user_id=?";
    let present_history: Vec<crate::UserPresentAllReceivedHistory> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    Ok(web::Json(AdminUserResponse {
        user,
        user_devices: devices,
        user_cards: cards,
        user_decks: decks,
        user_items: items,
        user_login_bonuses: login_bonuses,
        user_presents: presents,
        user_present_all_received_history: present_history,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminUserResponse {
    user: crate::User,

    user_devices: Vec<crate::UserDevice>,
    user_cards: Vec<crate::UserCard>,
    user_decks: Vec<crate::UserDeck>,
    user_items: Vec<crate::UserItem>,
    user_login_bonuses: Vec<crate::UserLoginBonus>,
    user_presents: Vec<crate::UserPresent>,
    user_present_all_received_history: Vec<crate::UserPresentAllReceivedHistory>,
}

/// ユーザBAN処理
/// POST /admin/user/{userId}/ban
pub async fn admin_ban_user(
    path: web::Path<(i64,)>,
    request_at: crate::RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AdminBanUserResponse>, crate::Error> {
    let (user_id,) = path.into_inner();
    let request_at = request_at.0;

    let query = "SELECT * FROM users WHERE id=?";
    let user: crate::User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(crate::Error::UserNotFound)?;

    let ban_id = crate::generate_id(&pool).await?;
    let query = "INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?";
    sqlx::query(query)
        .bind(ban_id)
        .bind(user_id)
        .bind(request_at)
        .bind(request_at)
        .bind(request_at)
        .execute(&**pool)
        .await?;

    Ok(web::Json(AdminBanUserResponse { user }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminBanUserResponse {
    user: crate::User,
}

fn verify_password(hash: &str, pw: &str) -> Result<(), crate::Error> {
    if bcrypt::verify(pw, hash).unwrap_or(false) {
        Ok(())
    } else {
        Err(crate::Error::Unauthorized)
    }
}

#[derive(Debug, sqlx::FromRow)]
struct AdminUser {
    id: i64,
    password: String,
    last_activated_at: i64,
    created_at: i64,
    updated_at: i64,
    deleted_at: Option<i64>,
}
