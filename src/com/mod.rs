use std::fmt::Debug;
use std::marker::Sized;
use std::time::Duration;

use axum::body::Body;
use axum::http::{HeaderMap, HeaderValue, Response};
use axum::response::IntoResponse;
use color_eyre::Result;
use derive_more::From;
use once_cell::sync::Lazy as SyncLazy;
use once_cell::unsync::Lazy;
use reqwest::header::{CACHE_CONTROL, USER_AGENT};
use reqwest::{Client, StatusCode};
use serde::{Deserialize, Deserializer};
use sqlx::Sqlite;
use sqlx::prelude::FromRow;
use sqlx::query::QueryAs;
use sqlx::sqlite::{SqliteArguments, SqliteRow};
use thiserror::Error;

pub mod category;
mod date;
pub mod feed;
pub mod item;

pub use category::Category;
pub use date::UtcDateTime;
pub use feed::Feed;
pub use item::Item;

const HTTP_TIMEOUT: Duration = Duration::from_secs(30);

pub type RssQueryAs<'a, T> = QueryAs<'a, Sqlite, T, SqliteArguments<'a>>;


#[derive(Debug)]
pub enum Action {
    RerunFailing,
    Rerun(i64),
    // Send ID so that we can be sure there's no chance that a poorly timed Fetcher poll could
    // load an old copy after this.
    // This can be sent on edit or creation.
    FeedChanged(i64),
}

#[derive(Debug)]
pub enum Outcome<T> {
    NoOp(T),
    Update(T),
}

impl<T> Outcome<T> {
    pub fn take(self) -> T {
        match self {
            Self::NoOp(t) | Self::Update(t) => t,
        }
    }

    pub fn if_update(self, f: impl FnOnce(&T)) -> Self {
        match &self {
            Self::NoOp(_) => {}
            Self::Update(t) => f(t),
        }
        self
    }
}

pub type QueryBuilder<'a> = sqlx::QueryBuilder<'a, Sqlite>;
pub type Separated<'a, 'b> = sqlx::query_builder::Separated<'a, 'b, Sqlite, &'static str>;
pub type LazyBuilder<'a> = Lazy<QueryBuilder<'a>>;

pub trait RssStruct: Sized + for<'r> FromRow<'r, SqliteRow> + Send + Unpin {
    fn id(&self) -> i64;

    fn table_name() -> &'static str;
}

pub trait Update<T: RssStruct>: Debug {
    fn validate(&self, s: &T) -> Result<()>;

    fn build_updates<'a>(self, s: &'a T, builder: &mut LazyBuilder<'a>);
}

#[derive(Debug)]
pub enum OnConflict {
    // No special handling
    Error,
    // The constraint, example "(feed_id, key)"
    Ignore(&'static str),
    //     Update {
    //         constraint: &'static str,
    //         // Don't include commit_timestamp
    //         colums: &'static [&'static str],
    //     },
}

pub trait Insert<T: RssStruct>: Debug {
    fn columns() -> &'static [&'static str];

    fn binds_count_hint() -> usize {
        0
    }

    fn on_conflict() -> OnConflict {
        OnConflict::Error
    }

    fn validate(&self) -> Result<()>;

    fn push_values(self, builder: &mut Separated<'_, '_>);
}


pub static CLIENT: SyncLazy<Client> = SyncLazy::new(|| {
    let mut headers = HeaderMap::new();
    // Workaround for dolphinemu.org, but doesn't seem to break any other feeds.
    headers.insert(CACHE_CONTROL, HeaderValue::from_static("no-cache"));

    // Pretend to be wget. Some sites don't like an empty user agent.
    // Reddit in particular will _always_ say to retry in a few seconds,
    // even if you wait hours.
    headers.insert(USER_AGENT, HeaderValue::from_static("Wget/1.19.5 (freebsd11.1)"));


    Client::builder()
        .use_rustls_tls()
        .default_headers(headers)
        .timeout(HTTP_TIMEOUT)
        .brotli(true)
        .gzip(true)
        .deflate(true)
        .build()
        .unwrap()
});

#[derive(From, Error, Debug)]
pub enum HttpError {
    #[error("{0}")]
    Report(color_eyre::Report),
    // #[error("{0}")]
    // Sql(sqlx::Error),
    #[error("Error {0:?}: {1}")]
    Status(StatusCode, &'static str),
}

impl IntoResponse for HttpError {
    fn into_response(self) -> Response<Body> {
        match self {
            Self::Report(e) => {
                error!("{e:?}");

                match e.downcast::<Self>() {
                    Ok(s) => return s.into_response(),
                    Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
                }
            }
            // Self::Sql(e) => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
            Self::Status(e, s) => (e, s.to_string()),
        }
        .into_response()
    }
}

impl HttpError {
    pub const fn bad(err: &'static str) -> Self {
        Self::Status(StatusCode::BAD_REQUEST, err)
    }
}

fn empty_string_is_none<'de, D>(deserializer: D) -> Result<Option<String>, D::Error>
where
    D: Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    if s.is_empty() { Ok(None) } else { Ok(Some(s)) }
}
