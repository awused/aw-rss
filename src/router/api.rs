use std::collections::HashMap;
use std::ops::Add;

use axum::extract::{Path, State};
use axum::routing::{get, post};
use axum::{Json, Router};
use chrono::{DateTime, TimeDelta, Utc};
use color_eyre::Result;
use derive_more::From;
use serde::{Deserialize, Serialize};
use tokio::sync::MutexGuard;

use super::{AppResult, AppState};
use crate::com::feed::UserEdit as FeedEdit;
use crate::com::item::UserEdit as ItemEdit;
use crate::com::{Category, Feed, FetcherAction, Item, RssStruct};
use crate::database::Database;
use crate::RouterState;

mod add_feed;
mod get_items;


#[derive(Serialize, Debug, From)]
#[serde(rename_all = "camelCase")]
struct ItemsResponse {
    items: Vec<Item>,
}

async fn feed(State(state): AppState, Path(id): Path<i64>) -> AppResult<Json<Feed>> {
    let mut db = state.db.lock().await;
    Ok(Json(db.get_feed(id).await?))
}

async fn item(State(state): AppState, Path(id): Path<i64>) -> AppResult<Json<Item>> {
    let mut db = state.db.lock().await;
    Ok(Json(db.get_item(id).await?))
}
async fn category(State(state): AppState, Path(id): Path<i64>) -> AppResult<Json<Category>> {
    let mut db = state.db.lock().await;
    Ok(Json(db.get_category(id).await?))
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct CurrentState {
    timestamp: i64,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    items: Vec<Item>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    feeds: Vec<Feed>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    categories: Vec<Category>,
    #[serde(skip_serializing_if = "HashMap::is_empty")]
    newest_timestamps: HashMap<i64, DateTime<Utc>>,
}

async fn current(State(state): AppState) -> AppResult<Json<CurrentState>> {
    let mut db = state.db.lock().await;
    let mut tx = db.transaction().await?;

    let state = CurrentState {
        timestamp: tx.timestamp().await?,
        items: tx.current_items().await?,
        feeds: tx.current_feeds().await?,
        categories: tx.current_categories().await?,
        newest_timestamps: tx.newest_timestamps().await?,
    };

    tx.commit().await?;

    Ok(state.into())
}


#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct Updates {
    timestamp: i64,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    items: Vec<Item>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    feeds: Vec<Feed>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    categories: Vec<Category>,
    must_refresh: bool,
}

const MAX_STALENESS: TimeDelta = TimeDelta::weeks(1);

async fn updates(State(state): AppState, Path(ts): Path<i64>) -> AppResult<Json<Updates>> {
    let mut db = state.db.lock().await;
    let mut tx = db.transaction().await?;

    let ts = DateTime::from_timestamp(ts, 0).unwrap();
    let timestamp = tx.timestamp().await?;

    if ts.add(MAX_STALENESS).timestamp() < timestamp {
        tx.commit().await?;

        return Ok(Updates {
            timestamp,
            items: Vec::new(),
            feeds: Vec::new(),
            categories: Vec::new(),
            must_refresh: true,
        }
        .into());
    }

    let ts = ts.into();
    let updates = Updates {
        timestamp,
        items: tx.updated(ts).await?,
        feeds: tx.updated(ts).await?,
        categories: tx.updated(ts).await?,
        must_refresh: false,
    };

    tx.commit().await?;

    Ok(updates.into())
}

async fn item_read(State(state): AppState, Path(id): Path<i64>) -> AppResult<Json<Item>> {
    let edit = ItemEdit { read: true };
    let db = state.db.lock().await;
    Ok(Json(Database::single_edit(db, id, edit).await?.take()))
}

async fn item_unread(State(state): AppState, Path(id): Path<i64>) -> AppResult<Json<Item>> {
    let edit = ItemEdit { read: false };
    let db = state.db.lock().await;
    Ok(Json(Database::single_edit(db, id, edit).await?.take()))
}

async fn disabled_feeds(State(state): AppState) -> AppResult<Json<Vec<Feed>>> {
    let mut db = state.db.lock().await;
    Ok(Json(db.disabled_feeds().await?))
}

#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
struct ReadFeedRequest {
    max_item_id: i64,
}

impl ReadFeedRequest {
    async fn apply(self, mut db: MutexGuard<'_, Database>, feed_id: i64) -> Result<Vec<Item>> {
        let query = sqlx::query_as(
            "
UPDATE items
SET read = 1
WHERE feed_id = ? AND read = 0 AND id <= ?
RETURNING *",
        )
        .bind(feed_id)
        .bind(self.max_item_id);

        let mut unsorted = db.fetch_all(query).await?;

        unsorted.sort_by_key(|i: &Item| i.id());

        Ok(unsorted)
    }
}

async fn feed_read(
    State(state): AppState,
    Path(id): Path<i64>,
    Json(req): Json<ReadFeedRequest>,
) -> AppResult<Json<ItemsResponse>> {
    let db = state.db.lock().await;
    Ok(Json(req.apply(db, id).await?.into()))
}

#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
struct EditFeedRequest {
    edit: FeedEdit,
}

async fn feed_edit(
    State(state): AppState,
    Path(id): Path<i64>,
    Json(req): Json<EditFeedRequest>,
) -> AppResult<Json<Feed>> {
    let db = state.db.lock().await;
    Ok(Json(
        Database::single_edit(db, id, req.edit)
            .await?
            // Sending won't fail unless we're closing, at which point we don't care.
            .if_update(|_| state.fetcher_sender.send(FetcherAction::FeedChanged(id)).unwrap())
            .take(),
    ))
}

async fn rerun_failing(State(state): AppState) {
    state.fetcher_sender.send(FetcherAction::RerunFailing).unwrap()
}

async fn feed_rerun(State(state): AppState, Path(id): Path<i64>) {
    state.fetcher_sender.send(FetcherAction::Rerun(id)).unwrap()
}

pub(super) fn api_router() -> Router<RouterState> {
    Router::new()
        // Items
        .route("/items", post(get_items::handle))
        .route("/items/:id/read", post(item_read))
        .route("/items/:id/unread", post(item_unread))

        // Feeds
        .route("/feeds/disabled", get(disabled_feeds))
        .route("/feeds/add", post(add_feed::handle))
        .route("/feeds/rerun-failing", post(rerun_failing))
        .route("/feeds/:id/edit", post(feed_edit))
        .route("/feeds/:id/read", post(feed_read))
        .route("/feeds/:id/rerun", post(feed_rerun))

        // Categories
        // .route("/categories/add", post(category_add))
        // .route("/categories/reorder", post(category_reorder))
        // .route("/categories/:id/edit", post(category_edit))

        .route("/current", get(current))
        .route("/updates/:timestamp", get(updates))


        // Temporary for debugging/development
        .route("/feed/:id", get(feed))
        .route("/item/:id", get(item))
        .route("/category/:id", get(category))
}
