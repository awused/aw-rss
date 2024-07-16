use std::fmt::Debug;
use std::marker::Sized;

use atom_syndication::Link;
use color_eyre::Result;
use once_cell::unsync::Lazy;
use sqlx::prelude::FromRow;
use sqlx::query::QueryAs;
use sqlx::sqlite::{SqliteArguments, SqliteRow};
use sqlx::Sqlite;

use crate::database::Transaction;

pub mod category;
mod date;
pub mod feed;
pub mod item;

pub use category::Category;
pub use date::UtcDateTime;
pub use feed::Feed;
pub use item::Item;

pub type RssQueryAs<'a, T> = QueryAs<'a, Sqlite, T, SqliteArguments<'a>>;

#[derive(Debug)]
pub enum FetcherAction {
    RerunFailing,
    Rerun(i64),
    // Send ID so that we can be sure there's no chance that a poorly timed Fetcher poll could
    // load an old copy after this.
    // This can be sent on edit or creation.
    FeedChanged(i64),
}

#[derive(Debug)]
pub enum EditResult<T> {
    NoOp(T),
    Update(T),
}

impl<T> EditResult<T> {
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

type QueryBuilder<'a> = sqlx::QueryBuilder<'a, Sqlite>;
type LazyBuilder<'a> = Lazy<QueryBuilder<'a>>;

pub trait Update<T: RssStruct>: Debug {
    fn build_updates<'a>(self, s: &'a T, builder: &mut LazyBuilder<'a>);
}

pub trait RssStruct: Sized + for<'r> FromRow<'r, SqliteRow> + Send + Unpin + Debug {
    fn id(&self) -> i64;

    fn table_name() -> &'static str;

    async fn update(
        id: i64,
        tx: &mut Transaction<'_>,
        edit: impl Update<Self>,
    ) -> Result<EditResult<Self>> {
        let s: Self = tx.get(id).await?;

        let mut builder: LazyBuilder<'_> = Lazy::new(|| {
            QueryBuilder::new(format!(
                "UPDATE {} SET commit_timestamp = CURRENT_TIMESTAMP ",
                Self::table_name()
            ))
        });

        edit.build_updates(&s, &mut builder);

        let Ok(mut edit) = Lazy::into_value(builder) else {
            return Ok(EditResult::NoOp(s));
        };

        Ok(EditResult::Update(
            edit.push(" WHERE id = ")
                .push_bind(id)
                .push(" RETURNING *")
                .build_query_as()
                .fetch_one(tx.con())
                .await?,
        ))
    }
}

// alternate > self > nothing > whatever else
pub fn extract_atom_url(mut links: Vec<Link>) -> Option<String> {
    links
        .iter()
        .position(|a| a.rel == "alternate")
        .or_else(|| links.iter().position(|a| a.rel == "self"))
        .or_else(|| links.iter().position(|a| a.rel == ""))
        .or_else(|| (!links.is_empty()).then_some(0))
        .map(|i| links.swap_remove(i).href)
}
