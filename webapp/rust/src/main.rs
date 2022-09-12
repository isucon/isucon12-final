use actix_web::http::StatusCode;
use actix_web::web;
use actix_web::HttpMessage as _;
use actix_web::HttpResponse;
use chrono::TimeZone as _;
use chrono::{DateTime, FixedOffset, NaiveDateTime};
use itertools::Itertools as _;
use sqlx::mysql::{MySqlArguments, MySqlDatabaseError};
use sqlx::Arguments as _;
use sqlx::MySqlPool;
use std::borrow::Cow;

mod admin;

const DECK_CARD_NUMBER: usize = 3;
pub const PRESENT_COUNT_PER_PAGE: i64 = 100;

static SQL_DIRECTORY: once_cell::sync::Lazy<std::path::PathBuf> =
    once_cell::sync::Lazy::new(|| std::path::PathBuf::from("../sql"));

static JST_OFFSET: once_cell::sync::Lazy<FixedOffset> =
    once_cell::sync::Lazy::new(|| FixedOffset::east(9 * 60 * 60));

#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("invalid request body")]
    InvalidRequestBody,
    #[error("invalid master version")]
    InvalidMasterVersion,
    #[error("invalid item type")]
    InvalidItemType,
    #[error("invalid token")]
    InvalidToken,
    #[error("failed to get request time")]
    GetRequestTime,
    #[error("session expired")]
    ExpiredSession,
    #[error("not found user")]
    UserNotFound,
    #[error("not found user device")]
    UserDeviceNotFound,
    #[error("not found item")]
    ItemNotFound,
    #[error("not found login bonus reward")]
    LoginBonusRewardNotFound,
    #[error("no such file")]
    NoFormFile,
    #[error("unauthorized user")]
    Unauthorized,
    #[error("forbidden")]
    Forbidden,
    #[error("SQLx error: {0}")]
    Sqlx(#[from] sqlx::Error),
    #[error("multipart error: {0}")]
    Multipart(#[from] actix_multipart::MultipartError),
    #[error("CSV error: {0}")]
    Csv(#[from] csv::Error),
    #[error("status={status_code}, message={message}")]
    Custom {
        status_code: StatusCode,
        message: Cow<'static, str>,
    },
}
#[derive(Debug, serde::Serialize)]
struct ErrorResponse {
    status_code: u16,
    message: String,
}
impl actix_web::ResponseError for Error {
    fn error_response(&self) -> HttpResponse {
        let message = format!("{}", self);
        let status_code = self.status_code().as_u16();
        tracing::error!("status={}, err={}", status_code, message);
        HttpResponse::build(self.status_code()).json(ErrorResponse {
            status_code,
            message,
        })
    }

    fn status_code(&self) -> StatusCode {
        match *self {
            Self::InvalidRequestBody
            | Self::InvalidItemType
            | Self::InvalidToken
            | Self::NoFormFile => StatusCode::BAD_REQUEST,
            Self::Forbidden => StatusCode::FORBIDDEN,
            Self::ExpiredSession | Self::Unauthorized => StatusCode::UNAUTHORIZED,
            Self::UserNotFound
            | Self::UserDeviceNotFound
            | Self::ItemNotFound
            | Self::LoginBonusRewardNotFound => StatusCode::NOT_FOUND,
            Self::InvalidMasterVersion => StatusCode::UNPROCESSABLE_ENTITY,
            Self::GetRequestTime | Self::Sqlx(_) | Self::Multipart(_) | Self::Csv(_) => {
                StatusCode::INTERNAL_SERVER_ERROR
            }
            Self::Custom { status_code, .. } => status_code,
        }
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    if std::env::var_os("RUST_LOG").is_none() {
        std::env::set_var("RUST_LOG", "info,sqlx=warn");
    }
    tracing_subscriber::fmt::init();

    // connect db
    let pool = sqlx::mysql::MySqlPoolOptions::new()
        .max_connections(50)
        .after_connect(|conn, _metadata| {
            Box::pin(async {
                use sqlx::Executor as _;
                conn.execute("set time_zone = '+09:00'").await?;
                Ok(())
            })
        })
        .connect(&format!(
            "mysql://{}:{}@{}:{}/{}",
            get_env("ISUCON_DB_USER", "isucon"),
            get_env("ISUCON_DB_PASSWORD", "isucon"),
            get_env("ISUCON_DB_HOST", "127.0.0.1"),
            get_env("ISUCON_DB_PORT", "3306"),
            get_env("ISUCON_DB_NAME", "isucon")
        ))
        .await
        .expect("failed to connect to db");

    let server = actix_web::HttpServer::new(move || {
        actix_web::App::new()
            .wrap(actix_web::middleware::Logger::default())
            .wrap(
                actix_cors::Cors::default()
                    .send_wildcard()
                    .allow_any_origin()
                    .allowed_methods([actix_web::http::Method::GET, actix_web::http::Method::POST])
                    .allowed_header(actix_web::http::header::CONTENT_TYPE)
                    .allowed_header("x-master-version")
                    .allowed_header("x-session"),
            )
            .app_data(web::Data::new(pool.clone()))
            // utility
            .route("/initialize", web::post().to(initialize))
            .route("/health", web::get().to(health))
            // feature
            .route(
                "/user",
                web::post().to(create_user).wrap(ApiMiddlewareFactory),
            )
            .route("/login", web::post().to(login).wrap(ApiMiddlewareFactory))
            .route(
                "/user/{userId}/gacha/index",
                web::get()
                    .to(list_gacha)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/gacha/draw/{gachaId}/{n}",
                web::post()
                    .to(draw_gacha)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/present/index/{n}",
                web::get()
                    .to(list_present)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/present/receive",
                web::post()
                    .to(receive_present)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/item",
                web::get()
                    .to(list_item)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/card/addexp/{cardId}",
                web::post()
                    .to(add_exp_to_card)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/card",
                web::post()
                    .to(update_deck)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/reward",
                web::post()
                    .to(reward)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            .route(
                "/user/{userId}/home",
                web::get()
                    .to(home)
                    .wrap(CheckSessionMiddlewareFactory)
                    .wrap(ApiMiddlewareFactory),
            )
            // admin
            .route(
                "/admin/login",
                web::post()
                    .to(admin::admin_login)
                    .wrap(AdminMiddlewareFactory),
            )
            .route(
                "/admin/logout",
                web::delete()
                    .to(admin::admin_logout)
                    .wrap(admin::AdminSessionCheckMiddlewareFactory)
                    .wrap(AdminMiddlewareFactory),
            )
            .route(
                "/admin/master",
                web::get()
                    .to(admin::admin_list_master)
                    .wrap(admin::AdminSessionCheckMiddlewareFactory)
                    .wrap(AdminMiddlewareFactory),
            )
            .route(
                "/admin/master",
                web::put()
                    .to(admin::admin_update_master)
                    .wrap(admin::AdminSessionCheckMiddlewareFactory)
                    .wrap(AdminMiddlewareFactory),
            )
            .route(
                "/admin/user/{userId}",
                web::get()
                    .to(admin::admin_user)
                    .wrap(admin::AdminSessionCheckMiddlewareFactory)
                    .wrap(AdminMiddlewareFactory),
            )
            .route(
                "/admin/user/{userId}/ban",
                web::post()
                    .to(admin::admin_ban_user)
                    .wrap(admin::AdminSessionCheckMiddlewareFactory)
                    .wrap(AdminMiddlewareFactory),
            )
    });

    if let Some(l) = listenfd::ListenFd::from_env().take_tcp_listener(0)? {
        server.listen(l)?
    } else {
        server.bind(("0.0.0.0", 8080))?
    }
    .run()
    .await
}

#[derive(Debug, Clone, Copy)]
pub struct RequestTime(i64);
impl actix_web::FromRequest for RequestTime {
    type Error = Error;
    type Future = std::future::Ready<Result<Self, Self::Error>>;

    fn from_request(
        req: &actix_web::HttpRequest,
        _payload: &mut actix_web::dev::Payload,
    ) -> Self::Future {
        std::future::ready(
            req.extensions()
                .get::<Self>()
                .copied()
                .ok_or(Error::GetRequestTime),
        )
    }
}

struct AdminMiddlewareFactory;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for AdminMiddlewareFactory
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
    type Transform = AdminMiddleware<S>;
    type Future = std::future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        std::future::ready(Ok(AdminMiddleware { service }))
    }
}
struct AdminMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for AdminMiddleware<S>
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
    type Future = S::Future;

    actix_web::dev::forward_ready!(service);

    fn call(&self, req: actix_web::dev::ServiceRequest) -> Self::Future {
        let request_at = chrono::Utc::now();
        req.extensions_mut()
            .insert(RequestTime(request_at.timestamp()));
        self.service.call(req)
    }
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct UserId {
    user_id: i64,
}

struct ApiMiddlewareFactory;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for ApiMiddlewareFactory
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
    type Transform = ApiMiddleware<S>;
    type Future = std::future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        std::future::ready(Ok(ApiMiddleware { service }))
    }
}
struct ApiMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for ApiMiddleware<S>
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
        let request_at = req
            .headers()
            .get("x-isu-date")
            .and_then(|header_value| header_value.to_str().ok())
            .and_then(|x_isu_date| DateTime::parse_from_rfc2822(x_isu_date).ok())
            .map(|t| t.timestamp())
            .unwrap_or_else(|| chrono::Utc::now().timestamp());
        req.extensions_mut().insert(RequestTime(request_at));

        let x_master_version = req
            .headers()
            .get("x-master-version")
            .map(|val| val.to_str().unwrap().to_owned());
        let pool_fut = req.extract::<web::Data<MySqlPool>>();
        let user_id_fut = req.extract::<web::Path<UserId>>();
        let fut = self.service.call(req);

        Box::pin(async move {
            let pool = pool_fut.await?;

            // マスタ確認
            let query = "SELECT * FROM version_masters WHERE status=1";
            let master_version: VersionMaster = sqlx::query_as(query)
                .fetch_optional(&**pool)
                .await
                .map_err(Error::Sqlx)?
                .ok_or_else(|| Error::Custom {
                    status_code: StatusCode::NOT_FOUND,
                    message: "active master version is not found".into(),
                })?;

            if master_version.master_version
                != x_master_version.ok_or(Error::InvalidMasterVersion)?
            {
                return Err(Error::InvalidMasterVersion.into());
            }

            // check ban
            if let Ok(user_id) = user_id_fut.await {
                let user_id = user_id.user_id;
                let is_ban = check_ban(&pool, user_id).await.map_err(Error::Sqlx)?;
                if is_ban {
                    return Err(Error::Forbidden.into());
                }
            }

            // next
            fut.await
        })
    }
}

struct CheckSessionMiddlewareFactory;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest>
    for CheckSessionMiddlewareFactory
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
    type Transform = CheckSessionMiddleware<S>;
    type Future = std::future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        std::future::ready(Ok(CheckSessionMiddleware { service }))
    }
}
struct CheckSessionMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for CheckSessionMiddleware<S>
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
        let sess_id = req.headers().get("x-session");
        if sess_id.is_none() {
            return Box::pin(async { Err(Error::Unauthorized.into()) });
        }
        let sess_id = sess_id.unwrap().to_str().unwrap().to_owned();

        let user_id_fut = req.extract::<web::Path<UserId>>();
        let request_at_fut = req.extract::<RequestTime>();
        let pool_fut = req.extract::<web::Data<MySqlPool>>();
        let fut = self.service.call(req);

        Box::pin(async move {
            let user_id = user_id_fut
                .await
                .map_err(|e| Error::Custom {
                    status_code: StatusCode::BAD_REQUEST,
                    message: format!("{}", e).into(),
                })?
                .user_id;

            let request_at = request_at_fut.await?.0;

            let pool = pool_fut.await?;
            let query = "SELECT * FROM user_sessions WHERE session_id=? AND deleted_at IS NULL";
            let user_session: Session = sqlx::query_as(query)
                .bind(&sess_id)
                .fetch_optional(&**pool)
                .await
                .map_err(Error::Sqlx)?
                .ok_or(Error::Unauthorized)?;

            if user_session.user_id != user_id {
                return Err(Error::Forbidden.into());
            }

            if user_session.expired_at < request_at {
                let query = "UPDATE user_sessions SET deleted_at=? WHERE session_id=?";
                sqlx::query(query)
                    .bind(request_at)
                    .bind(sess_id)
                    .execute(&**pool)
                    .await
                    .map_err(Error::Sqlx)?;
                return Err(Error::ExpiredSession.into());
            }

            // next
            fut.await
        })
    }
}

async fn check_one_time_token(
    pool: &MySqlPool,
    token: &str,
    token_type: i64,
    request_at: i64,
) -> Result<(), Error> {
    let query =
        "SELECT * FROM user_one_time_tokens WHERE token=? AND token_type=? AND deleted_at IS NULL";
    let tk: UserOneTimeToken = sqlx::query_as(query)
        .bind(token)
        .bind(token_type)
        .fetch_optional(pool)
        .await?
        .ok_or(Error::InvalidToken)?;

    if tk.expired_at < request_at {
        let query = "UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?";
        sqlx::query(query)
            .bind(request_at)
            .bind(token)
            .execute(pool)
            .await?;
        return Err(Error::InvalidToken);
    }

    // 使ったトークンを失効する
    let query = "UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?";
    sqlx::query(query)
        .bind(request_at)
        .bind(token)
        .execute(pool)
        .await?;

    Ok(())
}

async fn check_viewer_id(pool: &MySqlPool, user_id: i64, viewer_id: &str) -> Result<(), Error> {
    let query = "SELECT * FROM user_devices WHERE user_id=? AND platform_id=?";
    let device: Option<UserDevice> = sqlx::query_as(query)
        .bind(user_id)
        .bind(viewer_id)
        .fetch_optional(pool)
        .await?;

    if device.is_none() {
        Err(Error::UserDeviceNotFound)
    } else {
        Ok(())
    }
}

async fn check_ban(pool: &MySqlPool, user_id: i64) -> sqlx::Result<bool> {
    let query = "SELECT * FROM user_bans WHERE user_id=?";
    let ban_user: Option<UserBan> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(pool)
        .await?;
    Ok(ban_user.is_some())
}

/// ログイン処理
async fn login_process<'c>(
    pool: &MySqlPool,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
    user_id: i64,
    request_at: i64,
) -> Result<(User, Vec<UserLoginBonus>, Vec<UserPresent>), Error> {
    let query = "SELECT * FROM users WHERE id=?";
    let mut user: User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&mut *tx)
        .await?
        .ok_or(Error::UserNotFound)?;

    // ログインボーナス処理
    let login_bonuses = obtain_login_bonus(pool, &mut *tx, user_id, request_at).await?;

    // 全員プレゼント取得
    let all_presents = obtain_present(pool, &mut *tx, user_id, request_at).await?;

    user.isu_coin = sqlx::query_scalar("SELECT isu_coin FROM users WHERE id=?")
        .bind(user.id)
        .fetch_optional(&mut *tx)
        .await?
        .ok_or(Error::UserNotFound)?;

    user.updated_at = request_at;
    user.last_activated_at = request_at;

    let query = "UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?";
    sqlx::query(query)
        .bind(request_at)
        .bind(request_at)
        .bind(user_id)
        .execute(&mut *tx)
        .await?;

    Ok((user, login_bonuses, all_presents))
}

/// ログイン処理が終わっているか
fn is_complete_today_login(
    last_activated_at: DateTime<FixedOffset>,
    request_at: DateTime<FixedOffset>,
) -> bool {
    last_activated_at.date() == request_at.date()
}

async fn obtain_login_bonus<'c>(
    pool: &MySqlPool,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
    user_id: i64,
    request_at: i64,
) -> Result<Vec<UserLoginBonus>, Error> {
    // login bonus masterから有効なログインボーナスを取得
    let query = "SELECT * FROM login_bonus_masters WHERE start_at <= ? AND end_at >= ?";
    let login_bonuses: Vec<LoginBonusMaster> = sqlx::query_as(query)
        .bind(request_at)
        .bind(request_at)
        .fetch_all(&mut *tx)
        .await?;

    let mut send_login_bonuses = Vec::new();

    for bonus in login_bonuses {
        let mut init_bonus = false;
        // ボーナスの進捗取得
        let query = "SELECT * FROM user_login_bonuses WHERE user_id=? AND login_bonus_id=?";
        let user_bonus: Option<UserLoginBonus> = sqlx::query_as(query)
            .bind(user_id)
            .bind(bonus.id)
            .fetch_optional(&mut *tx)
            .await?;
        let mut user_bonus = if let Some(user_bonus) = user_bonus {
            user_bonus
        } else {
            init_bonus = true;
            let ub_id = generate_id(pool).await?;
            UserLoginBonus {
                // ボーナス初期化
                id: ub_id,
                user_id,
                login_bonus_id: bonus.id,
                last_reward_sequence: 0,
                loop_count: 1,
                created_at: request_at,
                updated_at: request_at,
                deleted_at: None,
            }
        };

        // ボーナス進捗更新
        if user_bonus.last_reward_sequence < bonus.column_count {
            user_bonus.last_reward_sequence += 1;
        } else if bonus.looped {
            user_bonus.loop_count += 1;
            user_bonus.last_reward_sequence = 1;
        } else {
            // 上限まで付与完了
            continue;
        }
        user_bonus.updated_at = request_at;

        // 今回付与するリソース取得
        let query =
            "SELECT * FROM login_bonus_reward_masters WHERE login_bonus_id=? AND reward_sequence=?";
        let reward_item: LoginBonusRewardMaster = sqlx::query_as(query)
            .bind(bonus.id)
            .bind(user_bonus.last_reward_sequence)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or(Error::LoginBonusRewardNotFound)?;

        obtain_item(
            pool,
            &mut *tx,
            user_id,
            reward_item.item_id,
            reward_item.item_type,
            reward_item.amount,
            request_at,
        )
        .await?;

        // 進捗の保存
        if init_bonus {
            let query = "INSERT INTO user_login_bonuses(id, user_id, login_bonus_id, last_reward_sequence, loop_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
            sqlx::query(query)
                .bind(user_bonus.id)
                .bind(user_bonus.user_id)
                .bind(user_bonus.login_bonus_id)
                .bind(user_bonus.last_reward_sequence)
                .bind(user_bonus.loop_count)
                .bind(user_bonus.created_at)
                .bind(user_bonus.updated_at)
                .execute(&mut *tx)
                .await?;
        } else {
            let query = "UPDATE user_login_bonuses SET last_reward_sequence=?, loop_count=?, updated_at=? WHERE id=?";
            sqlx::query(query)
                .bind(user_bonus.last_reward_sequence)
                .bind(user_bonus.loop_count)
                .bind(user_bonus.updated_at)
                .bind(user_bonus.id)
                .execute(&mut *tx)
                .await?;
        }

        send_login_bonuses.push(user_bonus);
    }

    Ok(send_login_bonuses)
}

/// プレゼント付与処理
async fn obtain_present<'c>(
    pool: &MySqlPool,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
    user_id: i64,
    request_at: i64,
) -> Result<Vec<UserPresent>, Error> {
    let query = "SELECT * FROM present_all_masters WHERE registered_start_at <= ? AND registered_end_at >= ?";
    let normal_presents: Vec<PresentAllMaster> = sqlx::query_as(query)
        .bind(request_at)
        .bind(request_at)
        .fetch_all(&mut *tx)
        .await?;

    // 全員プレゼント取得情報更新
    let mut obtain_presents = Vec::new();
    for np in normal_presents {
        let query =
            "SELECT * FROM user_present_all_received_history WHERE user_id=? AND present_all_id=?";
        let received: Option<UserPresentAllReceivedHistory> = sqlx::query_as(query)
            .bind(user_id)
            .bind(np.id)
            .fetch_optional(&mut *tx)
            .await?;
        if received.is_some() {
            // プレゼント配布済
            continue;
        }

        // user present boxに入れる
        let p_id = generate_id(pool).await?;
        let up = UserPresent {
            id: p_id,
            user_id,
            sent_at: request_at,
            item_type: np.item_type,
            item_id: np.item_id,
            amount: np.amount,
            present_message: np.present_message,
            created_at: request_at,
            updated_at: request_at,
            deleted_at: None,
        };
        let query = "INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)";
        sqlx::query(query)
            .bind(up.id)
            .bind(up.user_id)
            .bind(up.sent_at)
            .bind(up.item_type)
            .bind(up.item_id)
            .bind(up.amount)
            .bind(&up.present_message)
            .bind(up.created_at)
            .bind(up.updated_at)
            .execute(&mut *tx)
            .await?;

        // historyに入れる
        let ph_id = generate_id(pool).await?;
        let history = UserPresentAllReceivedHistory {
            id: ph_id,
            user_id,
            present_all_id: np.id,
            received_at: request_at,
            created_at: request_at,
            updated_at: request_at,
            deleted_at: None,
        };
        let query = "INSERT INTO user_present_all_received_history(id, user_id, present_all_id, received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)";
        sqlx::query(query)
            .bind(history.id)
            .bind(history.user_id)
            .bind(history.present_all_id)
            .bind(history.received_at)
            .bind(history.created_at)
            .bind(history.updated_at)
            .execute(&mut *tx)
            .await?;

        obtain_presents.push(up);
    }

    Ok(obtain_presents)
}

/// アイテム付与処理
async fn obtain_item<'c>(
    pool: &MySqlPool,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
    user_id: i64,
    item_id: i64,
    item_type: i64,
    obtain_amount: i64,
    request_at: i64,
) -> Result<(Vec<i64>, Vec<UserCard>, Vec<UserItem>), Error> {
    let mut obtain_coins = Vec::new();
    let mut obtain_cards = Vec::new();
    let mut obtain_items = Vec::new();

    match item_type {
        1 => {
            // coin
            let query = "SELECT * FROM users WHERE id=?";
            let user: User = sqlx::query_as(query)
                .bind(user_id)
                .fetch_optional(&mut *tx)
                .await?
                .ok_or(Error::UserNotFound)?;

            let query = "UPDATE users SET isu_coin=? WHERE id=?";
            let total_coin = user.isu_coin + obtain_amount;
            sqlx::query(query)
                .bind(total_coin)
                .bind(user.id)
                .execute(&mut *tx)
                .await?;
            obtain_coins.push(obtain_amount);
        }

        2 => {
            // card(ハンマー)
            let query = "SELECT * FROM item_masters WHERE id=? AND item_type=?";
            let item: ItemMaster = sqlx::query_as(query)
                .bind(item_id)
                .bind(item_type)
                .fetch_optional(&mut *tx)
                .await?
                .ok_or(Error::ItemNotFound)?;

            let c_id = generate_id(pool).await?;
            let card = UserCard {
                id: c_id,
                user_id,
                card_id: item.id,
                amount_per_sec: item.amount_per_sec.unwrap(),
                level: 1,
                total_exp: 0,
                created_at: request_at,
                updated_at: request_at,
                deleted_at: None,
            };
            let query = "INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
            sqlx::query(query)
                .bind(card.id)
                .bind(card.user_id)
                .bind(card.card_id)
                .bind(card.amount_per_sec)
                .bind(card.level)
                .bind(card.total_exp)
                .bind(card.created_at)
                .bind(card.updated_at)
                .execute(&mut *tx)
                .await?;
            obtain_cards.push(card);
        }

        3 | 4 => {
            // 強化素材
            let query = "SELECT * FROM item_masters WHERE id=? AND item_type=?";
            let item: ItemMaster = sqlx::query_as(query)
                .bind(item_id)
                .bind(item_type)
                .fetch_optional(&mut *tx)
                .await?
                .ok_or(Error::ItemNotFound)?;
            // 所持数取得
            let query = "SELECT * FROM user_items WHERE user_id=? AND item_id=?";
            let uitem: Option<UserItem> = sqlx::query_as(query)
                .bind(user_id)
                .bind(item.id)
                .fetch_optional(&mut *tx)
                .await?;

            let uitem = if let Some(mut uitem) = uitem {
                // 更新
                uitem.amount += obtain_amount;
                uitem.updated_at = request_at;
                let query = "UPDATE user_items SET amount=?, updated_at=? WHERE id=?";
                sqlx::query(query)
                    .bind(uitem.amount)
                    .bind(uitem.updated_at)
                    .bind(uitem.id)
                    .execute(&mut *tx)
                    .await?;
                uitem
            } else {
                // 新規作成
                let uitem_id = generate_id(pool).await?;
                let uitem = UserItem {
                    id: uitem_id,
                    user_id,
                    item_type: item.item_type,
                    item_id: item.id,
                    amount: obtain_amount,
                    created_at: request_at,
                    updated_at: request_at,
                    deleted_at: None,
                };
                let query = "INSERT INTO user_items(id, user_id, item_id, item_type, amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
                sqlx::query(query)
                    .bind(uitem.id)
                    .bind(user_id)
                    .bind(uitem.item_id)
                    .bind(uitem.item_type)
                    .bind(uitem.amount)
                    .bind(request_at)
                    .bind(request_at)
                    .execute(&mut *tx)
                    .await?;
                uitem
            };

            obtain_items.push(uitem);
        }
        _ => {
            return Err(Error::InvalidItemType);
        }
    }

    Ok((obtain_coins, obtain_cards, obtain_items))
}

/// initialize 初期化処理
/// POST /initialize
async fn initialize() -> Result<web::Json<InitializeResponse>, Error> {
    let output = tokio::process::Command::new("/bin/sh")
        .arg("-c")
        .arg(SQL_DIRECTORY.join("init.sh"))
        .output()
        .await
        .map_err(|e| Error::Custom {
            status_code: StatusCode::INTERNAL_SERVER_ERROR,
            message: format!("Failed to initialize: {}", e).into(),
        })?;
    if !output.status.success() {
        return Err(Error::Custom {
            status_code: StatusCode::INTERNAL_SERVER_ERROR,
            message: format!(
                "Failed to initialize out={}, err={}",
                String::from_utf8_lossy(&output.stdout),
                String::from_utf8_lossy(&output.stderr)
            )
            .into(),
        });
    }
    Ok(web::Json(InitializeResponse { language: "rust" }))
}

#[derive(Debug, serde::Serialize)]
struct InitializeResponse {
    language: &'static str,
}

/// ユーザの作成
/// POST /user
async fn create_user(
    req: web::Json<CreateUserRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<CreateUserResponse>, Error> {
    let req = req.into_inner();
    let request_at = request_at.0;

    if req.viewer_id.is_empty() || req.platform_type < 1 || req.platform_type > 3 {
        return Err(Error::InvalidRequestBody);
    }

    let mut tx = pool.begin().await?;

    // ユーザ作成
    let u_id = generate_id(&pool).await?;
    let user = User {
        id: u_id,
        isu_coin: 0,
        last_get_reward_at: request_at,
        last_activated_at: request_at,
        registered_at: request_at,
        created_at: request_at,
        updated_at: request_at,
        deleted_at: None,
    };
    let query = "INSERT INTO users(id, last_activated_at, registered_at, last_getreward_at, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(user.id)
        .bind(user.last_activated_at)
        .bind(user.registered_at)
        .bind(user.last_get_reward_at)
        .bind(user.created_at)
        .bind(user.updated_at)
        .execute(&mut tx)
        .await?;

    let ud_id = generate_id(&pool).await?;
    let user_device = UserDevice {
        id: ud_id,
        user_id: user.id,
        platform_id: req.viewer_id,
        platform_type: req.platform_type,
        created_at: request_at,
        updated_at: request_at,
        deleted_at: None,
    };
    let query = "INSERT INTO user_devices(id, user_id, platform_id, platform_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(user_device.id)
        .bind(user.id)
        .bind(&user_device.platform_id)
        .bind(req.platform_type)
        .bind(request_at)
        .bind(request_at)
        .execute(&mut tx)
        .await?;

    // 初期デッキ付与
    let query = "SELECT * FROM item_masters WHERE id=?";
    let init_card: ItemMaster = sqlx::query_as(query)
        .bind(2)
        .fetch_optional(&mut tx)
        .await?
        .ok_or(Error::ItemNotFound)?;

    let mut init_cards = Vec::with_capacity(3);
    for _ in 0..3 {
        let c_id = generate_id(&pool).await?;
        let card = UserCard {
            id: c_id,
            user_id: user.id,
            card_id: init_card.id,
            amount_per_sec: init_card.amount_per_sec.unwrap(),
            level: 1,
            total_exp: 0,
            created_at: request_at,
            updated_at: request_at,
            deleted_at: None,
        };
        let query = "INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
        sqlx::query(query)
            .bind(card.id)
            .bind(card.user_id)
            .bind(card.card_id)
            .bind(card.amount_per_sec)
            .bind(card.level)
            .bind(card.total_exp)
            .bind(card.created_at)
            .bind(card.updated_at)
            .execute(&mut tx)
            .await?;
        init_cards.push(card);
    }

    let deck_id = generate_id(&pool).await?;
    let init_deck = UserDeck {
        id: deck_id,
        user_id: user.id,
        card_id1: init_cards[0].id,
        card_id2: init_cards[1].id,
        card_id3: init_cards[2].id,
        created_at: request_at,
        updated_at: request_at,
        deleted_at: None,
    };
    let query = "INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(init_deck.id)
        .bind(init_deck.user_id)
        .bind(init_deck.card_id1)
        .bind(init_deck.card_id2)
        .bind(init_deck.card_id3)
        .bind(init_deck.created_at)
        .bind(init_deck.updated_at)
        .execute(&mut tx)
        .await?;

    // ログイン処理
    let (user, login_bonuses, presents) =
        login_process(&pool, &mut tx, user.id, request_at).await?;

    // generate session
    let s_id = generate_id(&pool).await?;
    let sess_id = generate_uuid();
    let sess = Session {
        id: s_id,
        user_id: user.id,
        session_id: sess_id,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 86400,
        deleted_at: None,
    };
    let query = "INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)";
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

    Ok(web::Json(CreateUserResponse {
        user_id: user.id,
        viewer_id: user_device.platform_id.clone(),
        session_id: sess.session_id,
        created_at: request_at,
        updated_resources: make_updated_resources(
            request_at,
            Some(user),
            Some(user_device),
            Some(init_cards),
            Some(vec![init_deck]),
            None,
            Some(login_bonuses),
            Some(presents),
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct CreateUserRequest {
    viewer_id: String,
    platform_type: i64,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct CreateUserResponse {
    user_id: i64,
    viewer_id: String,
    session_id: String,
    created_at: i64,
    updated_resources: UpdatedResource,
}

/// ログイン
/// POST /login
async fn login(
    req: web::Json<LoginRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<LoginResponse>, Error> {
    let req = req.into_inner();
    let request_at = request_at.0;

    let query = "SELECT * FROM users WHERE id=?";
    let user: User = sqlx::query_as(query)
        .bind(req.user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(Error::UserNotFound)?;

    // check ban
    let is_ban = check_ban(&pool, user.id).await?;
    if is_ban {
        return Err(Error::Forbidden);
    }

    // viewer id check
    check_viewer_id(&pool, user.id, &req.viewer_id).await?;

    let mut tx = pool.begin().await?;

    // sessionを更新
    let query = "UPDATE user_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(req.user_id)
        .execute(&mut tx)
        .await?;
    let s_id = generate_id(&pool).await?;
    let sess_id = generate_uuid();
    let sess = Session {
        id: s_id,
        user_id: req.user_id,
        session_id: sess_id,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 86400,
        deleted_at: None,
    };
    let query = "INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(sess.id)
        .bind(sess.user_id)
        .bind(&sess.session_id)
        .bind(sess.created_at)
        .bind(sess.updated_at)
        .bind(sess.expired_at)
        .execute(&mut tx)
        .await?;

    // すでにログインしているユーザはログイン処理をしない
    if is_complete_today_login(
        JST_OFFSET.from_utc_datetime(&NaiveDateTime::from_timestamp(user.last_activated_at, 0)),
        JST_OFFSET.from_utc_datetime(&NaiveDateTime::from_timestamp(request_at, 0)),
    ) {
        let mut user = user;
        user.updated_at = request_at;
        user.last_activated_at = request_at;

        let query = "UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?";
        sqlx::query(query)
            .bind(request_at)
            .bind(request_at)
            .bind(req.user_id)
            .execute(&mut tx)
            .await?;

        tx.commit().await?;

        return Ok(web::Json(LoginResponse {
            viewer_id: req.viewer_id,
            session_id: sess.session_id,
            updated_resources: make_updated_resources(
                request_at,
                Some(user),
                None,
                None,
                None,
                None,
                None,
                None,
            ),
        }));
    }

    // login process
    let (user, login_bonuses, presents) =
        login_process(&pool, &mut tx, req.user_id, request_at).await?;

    tx.commit().await?;

    Ok(web::Json(LoginResponse {
        viewer_id: req.viewer_id,
        session_id: sess.session_id,
        updated_resources: make_updated_resources(
            request_at,
            Some(user),
            None,
            None,
            None,
            None,
            Some(login_bonuses),
            Some(presents),
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct LoginRequest {
    viewer_id: String,
    user_id: i64,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct LoginResponse {
    viewer_id: String,
    session_id: String,
    updated_resources: UpdatedResource,
}

/// ガチャ一覧
/// GET /user/{userId}/gacha/index
async fn list_gacha(
    path: web::Path<(i64,)>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<ListGachaResponse>, Error> {
    let (user_id,) = path.into_inner();
    let request_at = request_at.0;

    let query = "SELECT * FROM gacha_masters WHERE start_at <= ? AND end_at >= ? ORDER BY display_order ASC";
    let gacha_master_list: Vec<GachaMaster> = sqlx::query_as(query)
        .bind(request_at)
        .bind(request_at)
        .fetch_all(&**pool)
        .await?;

    if gacha_master_list.is_empty() {
        return Ok(web::Json(ListGachaResponse {
            gachas: Vec::new(),
            one_time_token: "".to_owned(),
        }));
    }

    // ガチャ排出アイテム取得
    let mut gacha_data_list = Vec::new();
    let query = "SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC";
    for v in gacha_master_list {
        let gacha_item: Vec<GachaItemMaster> =
            sqlx::query_as(query).bind(v.id).fetch_all(&**pool).await?;

        if gacha_item.is_empty() {
            return Err(Error::Custom {
                status_code: StatusCode::NOT_FOUND,
                message: "not found gacha item".into(),
            });
        }

        gacha_data_list.push(GachaData {
            gacha: v,
            gacha_item,
        });
    }

    // genearte one time token
    let query =
        "UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(user_id)
        .execute(&**pool)
        .await?;
    let t_id = generate_id(&pool).await?;
    let tk = generate_uuid();
    let token = UserOneTimeToken {
        id: t_id,
        user_id,
        token: tk,
        token_type: 1,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 600,
        deleted_at: None,
    };
    let query = "INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(token.id)
        .bind(token.user_id)
        .bind(&token.token)
        .bind(token.token_type)
        .bind(token.created_at)
        .bind(token.updated_at)
        .bind(token.expired_at)
        .execute(&**pool)
        .await?;

    Ok(web::Json(ListGachaResponse {
        one_time_token: token.token,
        gachas: gacha_data_list,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ListGachaResponse {
    one_time_token: String,
    gachas: Vec<GachaData>,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct GachaData {
    gacha: GachaMaster,
    #[serde(rename = "gachaItemList")]
    gacha_item: Vec<GachaItemMaster>,
}

/// ガチャを引く
/// POST /user/{userId}/gacha/draw/{gachaId}/{n}
async fn draw_gacha(
    path: web::Path<(i64, i64, i64)>,
    req: web::Json<DrawGachaRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<DrawGachaResponse>, Error> {
    let (user_id, gacha_id, gacha_count) = path.into_inner();
    let req = req.into_inner();
    let request_at = request_at.0;

    if gacha_count != 1 && gacha_count != 10 {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "invalid draw gacha times".into(),
        });
    }

    check_one_time_token(&pool, &req.one_time_token, 1, request_at).await?;

    check_viewer_id(&pool, user_id, &req.viewer_id).await?;

    let consumed_coin = gacha_count * 1000;

    // userのisuconが足りるか
    let query = "SELECT * FROM users WHERE id=?";
    let user: User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(Error::UserNotFound)?;
    if user.isu_coin < consumed_coin {
        return Err(Error::Custom {
            status_code: StatusCode::CONFLICT,
            message: "not enough isucon".into(),
        });
    }

    // gacha_idからガチャマスタの取得
    let query = "SELECT * FROM gacha_masters WHERE id=? AND start_at <= ? AND end_at >= ?";
    let gacha_info: GachaMaster = sqlx::query_as(query)
        .bind(gacha_id)
        .bind(request_at)
        .bind(request_at)
        .fetch_optional(&**pool)
        .await?
        .ok_or_else(|| Error::Custom {
            status_code: StatusCode::NOT_FOUND,
            message: "not found gacha".into(),
        })?;

    // GachaItemMasterからアイテムリスト取得
    let gacha_item_list: Vec<GachaItemMaster> =
        sqlx::query_as("SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC")
            .bind(gacha_id)
            .fetch_all(&**pool)
            .await?;
    if gacha_item_list.is_empty() {
        return Err(Error::Custom {
            status_code: StatusCode::NOT_FOUND,
            message: "not found gacha item".into(),
        });
    }

    // weightの合計値を算出
    let sum: sqlx::types::Decimal =
        sqlx::query_scalar("SELECT SUM(weight) FROM gacha_item_masters WHERE gacha_id=?")
            .bind(gacha_id)
            .fetch_one(&**pool)
            .await?;
    use num_traits::ToPrimitive as _;
    let sum = sum.to_i64().unwrap();

    // random値の導出 & 抽選
    let mut result = Vec::new();
    for _ in 0..gacha_count {
        use rand::Rng as _;
        let random = rand::thread_rng().gen_range(0..sum);
        let mut boundary = 0;
        for v in &gacha_item_list {
            boundary += v.weight;
            if random < boundary {
                result.push(v);
                break;
            }
        }
    }

    let mut tx = pool.begin().await?;

    // 直付与 => プレゼントに入れる
    let mut presents = Vec::new();
    for v in result {
        let p_id = generate_id(&pool).await?;
        let present = UserPresent {
            id: p_id,
            user_id,
            sent_at: request_at,
            item_type: v.item_type,
            item_id: v.item_id,
            amount: v.amount,
            present_message: format!("{}の付与アイテムです", gacha_info.name),
            created_at: request_at,
            updated_at: request_at,
            deleted_at: None,
        };
        let query = "INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)";
        sqlx::query(query)
            .bind(present.id)
            .bind(present.user_id)
            .bind(present.sent_at)
            .bind(present.item_type)
            .bind(present.item_id)
            .bind(present.amount)
            .bind(&present.present_message)
            .bind(present.created_at)
            .bind(present.updated_at)
            .execute(&mut tx)
            .await?;

        presents.push(present);
    }

    // isuconをへらす
    let query = "UPDATE users SET isu_coin=? WHERE id=?";
    let total_coin = user.isu_coin - consumed_coin;
    sqlx::query(query)
        .bind(total_coin)
        .bind(user.id)
        .execute(&mut tx)
        .await?;

    tx.commit().await?;

    Ok(web::Json(DrawGachaResponse { presents }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct DrawGachaRequest {
    viewer_id: String,
    one_time_token: String,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct DrawGachaResponse {
    presents: Vec<UserPresent>,
}

/// プレゼント一覧
/// GET /user/{userId}/present/index/{n}
async fn list_present(
    path: web::Path<(i64, i64)>,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<ListPresentResponse>, Error> {
    let (user_id, n) = path.into_inner();

    if n == 0 {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "index number is more than 1".into(),
        });
    }

    let offset = PRESENT_COUNT_PER_PAGE * (n - 1);
    let query = r#"
    SELECT * FROM user_presents
    WHERE user_id = ? AND deleted_at IS NULL
    ORDER BY created_at DESC, id
    LIMIT ? OFFSET ?"#;
    let present_list: Vec<UserPresent> = sqlx::query_as(query)
        .bind(user_id)
        .bind(PRESENT_COUNT_PER_PAGE)
        .bind(offset)
        .fetch_all(&**pool)
        .await?;

    let present_count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM user_presents WHERE user_id = ? AND deleted_at IS NULL",
    )
    .bind(user_id)
    .fetch_one(&**pool)
    .await?;

    let is_next = present_count > offset + PRESENT_COUNT_PER_PAGE;

    Ok(web::Json(ListPresentResponse {
        presents: present_list,
        is_next,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ListPresentResponse {
    presents: Vec<UserPresent>,
    is_next: bool,
}

/// プレゼント受け取り
/// POST /user/{userId}/present/receive
async fn receive_present(
    path: web::Path<(i64,)>,
    req: web::Json<ReceivePresentRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<ReceivePresentResponse>, Error> {
    let (user_id,) = path.into_inner();
    let req = req.into_inner();
    let request_at = request_at.0;

    if req.present_ids.is_empty() {
        return Err(Error::Custom {
            status_code: StatusCode::UNPROCESSABLE_ENTITY,
            message: "presentIds is empty".into(),
        });
    }

    check_viewer_id(&pool, user_id, &req.viewer_id).await?;

    // user_presentsに入っているが未取得のプレゼント取得
    let placeholders = req.present_ids.iter().map(|_| "?").join(",");
    let mut args = MySqlArguments::default();
    for present_id in &req.present_ids {
        args.add(present_id);
    }
    let query = format!(
        "SELECT * FROM user_presents WHERE id IN ({}) AND deleted_at IS NULL",
        placeholders
    );
    let mut obtain_present: Vec<UserPresent> =
        sqlx::query_as_with(&query, args).fetch_all(&**pool).await?;

    if obtain_present.is_empty() {
        return Ok(web::Json(ReceivePresentResponse {
            updated_resources: make_updated_resources(
                request_at,
                None,
                None,
                None,
                None,
                None,
                None,
                Some(Vec::new()),
            ),
        }));
    }

    let mut tx = pool.begin().await?;

    // 配布処理
    for v in obtain_present.iter_mut() {
        if v.deleted_at.is_some() {
            return Err(Error::Custom {
                status_code: StatusCode::INTERNAL_SERVER_ERROR,
                message: "received present".into(),
            });
        }

        v.updated_at = request_at;
        v.deleted_at = Some(request_at);
        let query = "UPDATE user_presents SET deleted_at=?, updated_at=? WHERE id=?";
        sqlx::query(query)
            .bind(request_at)
            .bind(request_at)
            .bind(v.id)
            .execute(&mut tx)
            .await?;

        obtain_item(
            &pool,
            &mut tx,
            v.user_id,
            v.item_id,
            v.item_type,
            v.amount,
            request_at,
        )
        .await?;
    }

    tx.commit().await?;

    Ok(web::Json(ReceivePresentResponse {
        updated_resources: make_updated_resources(
            request_at,
            None,
            None,
            None,
            None,
            None,
            None,
            Some(obtain_present),
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct ReceivePresentRequest {
    viewer_id: String,
    present_ids: Vec<i64>,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ReceivePresentResponse {
    updated_resources: UpdatedResource,
}

/// アイテムリスト
/// GET /user/{userId}/item
async fn list_item(
    path: web::Path<(i64,)>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<ListItemResponse>, Error> {
    let (user_id,) = path.into_inner();
    let request_at = request_at.0;

    let query = "SELECT * FROM users WHERE id=?";
    let user: User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(Error::UserNotFound)?;

    let query = "SELECT * FROM user_items WHERE user_id = ?";
    let item_list: Vec<UserItem> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    let query = "SELECT * FROM user_cards WHERE user_id=?";
    let card_list: Vec<UserCard> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_all(&**pool)
        .await?;

    // genearte one time token
    let query =
        "UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(user_id)
        .execute(&**pool)
        .await?;
    let t_id = generate_id(&pool).await?;
    let tk = generate_uuid();
    let token = UserOneTimeToken {
        id: t_id,
        user_id,
        token: tk,
        token_type: 2,
        created_at: request_at,
        updated_at: request_at,
        expired_at: request_at + 600,
        deleted_at: None,
    };
    let query = "INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(token.id)
        .bind(token.user_id)
        .bind(&token.token)
        .bind(token.token_type)
        .bind(token.created_at)
        .bind(token.updated_at)
        .bind(token.expired_at)
        .execute(&**pool)
        .await?;

    Ok(web::Json(ListItemResponse {
        one_time_token: token.token,
        items: item_list,
        user,
        cards: card_list,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ListItemResponse {
    one_time_token: String,
    user: User,
    items: Vec<UserItem>,
    cards: Vec<UserCard>,
}

/// 装備強化
/// POST /user/{userId}/card/addexp/{cardId}
async fn add_exp_to_card(
    path: web::Path<(i64, i64)>,
    req: web::Json<AddExpToCardRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<AddExpToCardResponse>, Error> {
    let (user_id, card_id) = path.into_inner();
    let req = req.into_inner();
    let request_at = request_at.0;

    check_one_time_token(&pool, &req.one_time_token, 2, request_at).await?;

    check_viewer_id(&pool, user_id, &req.viewer_id).await?;

    // get target card
    let query = r#"
    SELECT uc.id , uc.user_id , uc.card_id , uc.amount_per_sec , uc.level, uc.total_exp, im.amount_per_sec as 'base_amount_per_sec', im.max_level , im.max_amount_per_sec , im.base_exp_per_level
    FROM user_cards as uc
    INNER JOIN item_masters as im ON uc.card_id = im.id
    WHERE uc.id = ? AND uc.user_id=?
    "#;
    let mut card: TargetUserCardData = sqlx::query_as(query)
        .bind(card_id)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or_else(|| Error::Custom {
            status_code: StatusCode::NOT_FOUND,
            message: "sql: no rows in result set".into(),
        })?;

    if card.level == card.max_level {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "target card is max level".into(),
        });
    }

    // 消費アイテムの所持チェック
    let mut items = Vec::new();
    let query = r#"
    SELECT ui.id, ui.user_id, ui.item_id, ui.item_type, ui.amount, ui.created_at, ui.updated_at, im.gained_exp
    FROM user_items as ui
    INNER JOIN item_masters as im ON ui.item_id = im.id
    WHERE ui.item_type = 3 AND ui.id=? AND ui.user_id=?
    "#;
    for v in req.items {
        let mut item: ConsumeUserItemData = sqlx::query_as(query)
            .bind(v.id)
            .bind(user_id)
            .fetch_optional(&**pool)
            .await?
            .ok_or_else(|| Error::Custom {
                status_code: StatusCode::NOT_FOUND,
                message: "sql: no rows in result set".into(),
            })?;

        if v.amount > item.amount {
            return Err(Error::Custom {
                status_code: StatusCode::BAD_REQUEST,
                message: "item not enough".into(),
            });
        }
        item.consume_amount = v.amount;
        items.push(item);
    }

    // 経験値付与
    // 経験値をカードに付与
    for v in &items {
        card.total_exp += v.gained_exp * v.consume_amount;
    }

    // lvup判定(lv upしたら生産性を加算)
    loop {
        let next_lv_threshold = card.base_exp_per_level as f64 * 1.2f64.powi(card.level as i32 - 1);
        if next_lv_threshold as i64 > card.total_exp {
            break;
        }

        // lv up処理
        card.level += 1;
        card.amount_per_sec +=
            (card.max_amount_per_sec - card.base_amount_per_sec) / (card.max_level - 1);
    }

    let mut tx = pool.begin().await?;

    // cardのlvと経験値の更新、itemの消費
    let query =
        "UPDATE user_cards SET amount_per_sec=?, level=?, total_exp=?, updated_at=? WHERE id=?";
    sqlx::query(query)
        .bind(card.amount_per_sec)
        .bind(card.level)
        .bind(card.total_exp)
        .bind(request_at)
        .bind(card.id)
        .execute(&mut tx)
        .await?;

    let query = "UPDATE user_items SET amount=?, updated_at=? WHERE id=?";
    for v in &items {
        sqlx::query(query)
            .bind(v.amount - v.consume_amount)
            .bind(request_at)
            .bind(v.id)
            .execute(&mut tx)
            .await?;
    }

    // get response data
    let query = "SELECT * FROM user_cards WHERE id=?";
    let result_card: UserCard = sqlx::query_as(query)
        .bind(card.id)
        .fetch_optional(&mut tx)
        .await?
        .ok_or_else(|| Error::Custom {
            status_code: StatusCode::NOT_FOUND,
            message: "not found card".into(),
        })?;
    let result_items = items
        .into_iter()
        .map(|v| UserItem {
            id: v.id,
            user_id: v.user_id,
            item_id: v.item_id,
            item_type: v.item_type,
            amount: v.amount - v.consume_amount,
            created_at: v.created_at,
            updated_at: request_at,
            deleted_at: None,
        })
        .collect();

    tx.commit().await?;

    Ok(web::Json(AddExpToCardResponse {
        updated_resources: make_updated_resources(
            request_at,
            None,
            None,
            Some(vec![result_card]),
            None,
            Some(result_items),
            None,
            None,
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct AddExpToCardRequest {
    viewer_id: String,
    one_time_token: String,
    items: Vec<ConsumeItem>,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct AddExpToCardResponse {
    updated_resources: UpdatedResource,
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct ConsumeItem {
    id: i64,
    amount: i64,
}

#[derive(Debug, sqlx::FromRow)]
struct ConsumeUserItemData {
    id: i64,
    user_id: i64,
    item_id: i64,
    item_type: i64,
    amount: i64,
    created_at: i64,
    updated_at: i64,
    gained_exp: i64,

    #[sqlx(default)]
    consume_amount: i64, // 消費量
}

#[derive(Debug, sqlx::FromRow)]
struct TargetUserCardData {
    id: i64,
    user_id: i64,
    card_id: i64,
    amount_per_sec: i64,
    level: i64,
    total_exp: i64,

    // lv1のときの生産性
    base_amount_per_sec: i64,
    // 最高レベル
    max_level: i64,
    // lv maxのときの生産性
    max_amount_per_sec: i64,
    // lv1 -> lv2に上がるときのexp
    base_exp_per_level: i64,
}

/// 装備変更
/// POST /user/{userId}/card
async fn update_deck(
    path: web::Path<(i64,)>,
    req: web::Json<UpdateDeckRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<UpdateDeckResponse>, Error> {
    let (user_id,) = path.into_inner();
    let req = req.into_inner();
    let request_at = request_at.0;

    if req.card_ids.len() != DECK_CARD_NUMBER {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "invalid number of cards".into(),
        });
    }

    check_viewer_id(&pool, user_id, &req.viewer_id).await?;

    // カード所持情報のバリデーション
    let query = "SELECT * FROM user_cards WHERE id IN (?, ?, ?)";
    let cards: Vec<UserCard> = sqlx::query_as(query)
        .bind(req.card_ids[0])
        .bind(req.card_ids[1])
        .bind(req.card_ids[2])
        .fetch_all(&**pool)
        .await?;
    if cards.len() != DECK_CARD_NUMBER {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "invalid card ids".into(),
        });
    }

    let mut tx = pool.begin().await?;

    // update data
    let query =
        "UPDATE user_decks SET updated_at=?, deleted_at=? WHERE user_id=? AND deleted_at IS NULL";
    sqlx::query(query)
        .bind(request_at)
        .bind(request_at)
        .bind(user_id)
        .execute(&mut tx)
        .await?;

    let ud_id = generate_id(&pool).await?;
    let new_deck = UserDeck {
        id: ud_id,
        user_id,
        card_id1: req.card_ids[0],
        card_id2: req.card_ids[1],
        card_id3: req.card_ids[2],
        created_at: request_at,
        updated_at: request_at,
        deleted_at: None,
    };
    let query = "INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)";
    sqlx::query(query)
        .bind(new_deck.id)
        .bind(new_deck.user_id)
        .bind(new_deck.card_id1)
        .bind(new_deck.card_id2)
        .bind(new_deck.card_id3)
        .bind(new_deck.created_at)
        .bind(new_deck.updated_at)
        .execute(&mut tx)
        .await?;

    tx.commit().await?;

    Ok(web::Json(UpdateDeckResponse {
        updated_resources: make_updated_resources(
            request_at,
            None,
            None,
            None,
            Some(vec![new_deck]),
            None,
            None,
            None,
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct UpdateDeckRequest {
    viewer_id: String,
    card_ids: Vec<i64>,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UpdateDeckResponse {
    updated_resources: UpdatedResource,
}

/// ゲーム報酬受取
/// POST /user/{userId}/reward
async fn reward(
    path: web::Path<(i64,)>,
    req: web::Json<RewardRequest>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<RewardResponse>, Error> {
    let (user_id,) = path.into_inner();
    let req = req.into_inner();
    let request_at = request_at.0;

    check_viewer_id(&pool, user_id, &req.viewer_id).await?;

    // 最後に取得した報酬時刻取得
    let query = "SELECT * FROM users WHERE id=?";
    let user: User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(Error::UserNotFound)?;

    // 使っているデッキの取得
    let query = "SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL";
    let deck: UserDeck = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or_else(|| Error::Custom {
            status_code: StatusCode::NOT_FOUND,
            message: "sql: no rows in result set".into(),
        })?;

    let query = "SELECT * FROM user_cards WHERE id IN (?, ?, ?)";
    let cards: Vec<UserCard> = sqlx::query_as(query)
        .bind(deck.card_id1)
        .bind(deck.card_id2)
        .bind(deck.card_id3)
        .fetch_all(&**pool)
        .await?;
    if cards.len() != 3 {
        return Err(Error::Custom {
            status_code: StatusCode::BAD_REQUEST,
            message: "invalid cards length".into(),
        });
    }

    // 経過時間*生産性のcoin (1椅子 = 1coin)
    let past_time = request_at - user.last_get_reward_at;
    let get_coin =
        past_time * (cards[0].amount_per_sec + cards[1].amount_per_sec + cards[2].amount_per_sec);

    // 報酬の保存(ゲームない通貨を保存)(users)
    let mut user = user;
    user.isu_coin += get_coin;
    user.last_get_reward_at = request_at;

    let query = "UPDATE users SET isu_coin=?, last_getreward_at=? WHERE id=?";
    sqlx::query(query)
        .bind(user.isu_coin)
        .bind(user.last_get_reward_at)
        .bind(user.id)
        .execute(&**pool)
        .await?;

    Ok(web::Json(RewardResponse {
        updated_resources: make_updated_resources(
            request_at,
            Some(user),
            None,
            None,
            None,
            None,
            None,
            None,
        ),
    }))
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct RewardRequest {
    viewer_id: String,
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct RewardResponse {
    updated_resources: UpdatedResource,
}

/// ホーム取得
/// GET /user/{userId}/home
async fn home(
    path: web::Path<(i64,)>,
    request_at: RequestTime,
    pool: web::Data<MySqlPool>,
) -> Result<web::Json<HomeResponse>, Error> {
    let (user_id,) = path.into_inner();
    let request_at = request_at.0;

    // 装備情報
    let query = "SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL";
    let deck: Option<UserDeck> = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?;

    // 生産性
    let cards: Vec<UserCard> = if let Some(ref deck) = deck {
        let query = "SELECT * FROM user_cards WHERE id IN (?, ?, ?)";
        sqlx::query_as(query)
            .bind(deck.card_id1)
            .bind(deck.card_id2)
            .bind(deck.card_id3)
            .fetch_all(&**pool)
            .await?
    } else {
        Vec::new()
    };
    let total_amount_per_sec = cards.iter().map(|v| v.amount_per_sec).sum();

    // 経過時間
    let query = "SELECT * FROM users WHERE id=?";
    let user: User = sqlx::query_as(query)
        .bind(user_id)
        .fetch_optional(&**pool)
        .await?
        .ok_or(Error::UserNotFound)?;
    let past_time = request_at - user.last_get_reward_at;

    Ok(web::Json(HomeResponse {
        now: request_at,
        user,
        deck,
        total_amount_per_sec,
        past_time,
    }))
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct HomeResponse {
    now: i64,
    user: User,
    #[serde(skip_serializing_if = "Option::is_none")]
    deck: Option<UserDeck>,
    total_amount_per_sec: i64,
    past_time: i64, // 経過時間を秒単位で
}

// //////////////////////////////////////
// util

/// ヘルスチェック
async fn health() -> &'static str {
    "OK"
}

/// uniqueなIDを生成する
pub async fn generate_id(pool: &MySqlPool) -> sqlx::Result<i64> {
    let mut update_err = None;
    for _ in 0..100 {
        match sqlx::query("UPDATE id_generator SET id=LAST_INSERT_ID(id+1)")
            .execute(pool)
            .await
        {
            Ok(res) => {
                return Ok(res.last_insert_id() as i64);
            }
            Err(e) => {
                if let Some(database_error) = e.as_database_error() {
                    if let Some(merr) = database_error.try_downcast_ref::<MySqlDatabaseError>() {
                        if merr.number() == 1213 {
                            update_err = Some(e);
                            continue;
                        }
                    }
                }
                return Err(e);
            }
        }
    }
    Err(update_err.unwrap())
}

pub fn generate_uuid() -> String {
    let id = uuid::Uuid::new_v4();
    id.hyphenated().to_string()
}

/// gets environment variable.
fn get_env(key: &str, default_val: &str) -> String {
    if let Ok(v) = std::env::var(key) {
        if v.is_empty() {
            default_val.to_owned()
        } else {
            v
        }
    } else {
        default_val.to_owned()
    }
}

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UpdatedResource {
    now: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    user: Option<User>,

    #[serde(skip_serializing_if = "Option::is_none")]
    user_device: Option<UserDevice>,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_cards: Option<Vec<UserCard>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_decks: Option<Vec<UserDeck>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_items: Option<Vec<UserItem>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_login_bonuses: Option<Vec<UserLoginBonus>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_presents: Option<Vec<UserPresent>>,
}

#[allow(clippy::too_many_arguments)]
fn make_updated_resources(
    request_at: i64,
    user: Option<User>,
    user_device: Option<UserDevice>,
    user_cards: Option<Vec<UserCard>>,
    user_decks: Option<Vec<UserDeck>>,
    user_items: Option<Vec<UserItem>>,
    user_login_bonuses: Option<Vec<UserLoginBonus>>,
    user_presents: Option<Vec<UserPresent>>,
) -> UpdatedResource {
    UpdatedResource {
        now: request_at,
        user,
        user_device,
        user_cards,
        user_decks,
        user_items,
        user_login_bonuses,
        user_presents,
    }
}

// //////////////////////////////////////
// entity

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct User {
    id: i64,
    isu_coin: i64,
    #[sqlx(rename = "last_getreward_at")]
    last_get_reward_at: i64,
    last_activated_at: i64,
    registered_at: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserDevice {
    id: i64,
    user_id: i64,
    platform_id: String,
    platform_type: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow)]
struct UserBan {
    id: i64,
    user_id: i64,
    created_at: i64,
    updated_at: i64,
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserCard {
    id: i64,
    user_id: i64,
    card_id: i64,
    amount_per_sec: i64,
    level: i64,
    total_exp: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserDeck {
    id: i64,
    user_id: i64,
    #[sqlx(rename = "user_card_id_1")]
    card_id1: i64,
    #[sqlx(rename = "user_card_id_2")]
    card_id2: i64,
    #[sqlx(rename = "user_card_id_3")]
    card_id3: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserItem {
    id: i64,
    user_id: i64,
    item_type: i64,
    item_id: i64,
    amount: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserLoginBonus {
    id: i64,
    user_id: i64,
    login_bonus_id: i64,
    last_reward_sequence: i64,
    loop_count: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserPresent {
    id: i64,
    user_id: i64,
    sent_at: i64,
    item_type: i64,
    item_id: i64,
    amount: i64,
    present_message: String,
    created_at: i64,
    updated_at: i64,
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserPresentAllReceivedHistory {
    id: i64,
    user_id: i64,
    present_all_id: i64,
    received_at: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct Session {
    id: i64,
    user_id: i64,
    session_id: String,
    expired_at: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct UserOneTimeToken {
    id: i64,
    user_id: i64,
    token: String,
    token_type: i64,
    expired_at: i64,
    created_at: i64,
    updated_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<i64>,
}

// //////////////////////////////////////
// master

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct GachaMaster {
    id: i64,
    name: String,
    start_at: i64,
    end_at: i64,
    display_order: i64,
    created_at: i64,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct GachaItemMaster {
    id: i64,
    gacha_id: i64,
    item_type: i64,
    item_id: i64,
    amount: i64,
    weight: i64,
    created_at: i64,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ItemMaster {
    id: i64,
    item_type: i64,
    name: String,
    description: String,
    amount_per_sec: Option<i64>,
    max_level: Option<i64>,
    max_amount_per_sec: Option<i64>,
    base_exp_per_level: Option<i64>,
    gained_exp: Option<i64>,
    shortening_min: Option<i64>,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct LoginBonusMaster {
    id: i64,
    start_at: i64,
    end_at: i64,
    column_count: i64,
    looped: bool,
    created_at: i64,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct LoginBonusRewardMaster {
    id: i64,
    login_bonus_id: i64,
    reward_sequence: i64,
    item_type: i64,
    item_id: i64,
    amount: i64,
    created_at: i64,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct PresentAllMaster {
    id: i64,
    registered_start_at: i64,
    registered_end_at: i64,
    item_type: i64,
    item_id: i64,
    amount: i64,
    present_message: String,
    created_at: i64,
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct VersionMaster {
    id: i64,
    status: i64,
    master_version: String,
}
