use std::num::NonZeroU32;

use axum::extract::State;
use axum::Json;
use color_eyre::Result;
use serde::Deserialize;
use sqlx::{QueryBuilder, Sqlite};
use tokio::sync::MutexGuard;

use super::ItemsResponse;
use crate::com::{Item, UtcDateTime};
use crate::database::Database;
use crate::router::{AppState, HttpError, HttpResult};


// This could be cleaned up a bit with enums, but it's not worth it due to the risk of silently
// doing the wrong thing.
#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct Request {
    category_id: Option<i64>,
    #[serde(default)]
    feed_ids: Vec<i64>,
    #[serde(default)]
    unread: bool,

    #[serde(flatten)]
    before: Option<ReadBefore>,

    // Fetch _all_ read items after this timestamp (inclusive)
    // This is used when backfilling on the frontend, but only in the rare
    // case where a category is open and a feed is added to it or re-enabled.
    read_after: Option<UtcDateTime>,
    // This only really makes sense in the context of fetching items for non-specific disabled
    // feeds.
    // Which I'll probably never implement.
    // #[serde(default)]
    // pub include_feeds: bool,
}

#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct ReadBefore {
    // Fetch _at least_ readBeforeCount items before this timestamp (exclusive)
    // Guaranteed that all existing read items between ReadBefore and the minimum
    // timestamp in the response are fetched.
    #[serde(rename = "readBefore")]
    date: UtcDateTime,
    #[serde(rename = "readBeforeCount")]
    count: NonZeroU32,
}


pub(super) async fn handle(
    State(state): AppState,
    Json(req): Json<Request>,
) -> HttpResult<Json<ItemsResponse>> {
    req.validate().map_err(HttpError::bad)?;

    let db = state.db.lock().await;
    Ok(Json(req.query(db).await?.into()))
}

impl Request {
    fn validate(&self) -> Result<(), &'static str> {
        if !self.unread && self.read_after.is_none() && self.before.is_none() {
            return Err("Empty or invalid GetItems request");
        }

        if self.category_id.is_some() && !self.feed_ids.is_empty() {
            return Err("Can't get by both categoryId and feedIds");
        }

        if self.unread && self.feed_ids.is_empty() {
            return Err("Can only request unread items by feedIds");
        }

        if self.read_after.is_some() && self.before.is_some() {
            return Err("Can't get both readAfter and readBefore");
        }

        if self.unread && self.before.is_some() {
            return Err("Can't get both unread and readBefore");
        }

        Ok(())
    }

    fn target_clause(&self, builder: &mut QueryBuilder<'_, Sqlite>) {
        if let Some(cat) = self.category_id {
            builder.push(" feeds.category_id = ").push_bind(cat);
        } else if !self.feed_ids.is_empty() {
            builder.push(" feeds.id IN (");

            let mut sep = builder.separated(",");
            for f in &self.feed_ids {
                // These are just integers
                sep.push_bind(*f);
            }

            builder.push(") ");
        } else {
            builder.push(" feeds.disabled = 0 ");
        }
    }

    async fn query(&self, mut db: MutexGuard<'_, Database>) -> Result<Vec<Item>> {
        let mut builder = QueryBuilder::new(
            "
SELECT items.*
FROM feeds CROSS JOIN items ON items.feed_id = feeds.id
WHERE
",
        );

        self.target_clause(&mut builder);

        if let Some(after) = self.read_after {
            // There was a branch for self.unread here, but it was never taken
            builder.push(" AND items.read = 1 AND items.timestamp >= ").push_bind(after);
        } else if let Some(before) = &self.before {
            // This grabs at least count items, but ensures that we get all items with
            // the same timestamp as the count'th oldest read item in the query.

            builder
                .push(" AND items.read = 1 AND items.timestamp < ")
                .push_bind(before.date)
                // All lines after this are just for getting the minimum timestamp
                .push(
                    "
AND items.timestamp >= (
    SELECT MIN(timestamp)
    FROM (
        SELECT items.timestamp
        FROM feeds CROSS JOIN items ON items.feed_id = feeds.id
        WHERE ",
                );

            self.target_clause(&mut builder);

            builder
                .push(" AND items.read = 1 AND items.timestamp < ")
                .push_bind(before.date)
                .push(" ORDER BY items.timestamp DESC LIMIT ")
                .push_bind(before.count.get())
                .push(" )) ");
        } else {
            builder.push(" AND items.read = 0 ");
        }

        builder.push(" ORDER BY items.id ASC");

        db.fetch_all(builder.build_query_as()).await
    }
}
