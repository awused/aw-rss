use axum::extract::State;
use axum::Json;
use color_eyre::Result;
use derive_more::derive::From;
use serde::{Deserialize, Serialize};
use sqlx::QueryBuilder;
use tokio::sync::MutexGuard;

use crate::com::category::Category;
use crate::com::RssStruct;
use crate::database::Database;
use crate::router::{AppState, HttpResult};


#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct Request {
    category_ids: Vec<i64>,
}

#[derive(Serialize, Debug, From)]
#[serde(rename_all = "camelCase")]
pub struct Response {
    categories: Vec<Category>,
}

pub(super) async fn handle(
    State(state): AppState,
    Json(req): Json<Request>,
) -> HttpResult<Json<Response>> {
    let db = state.db.lock().await;
    Ok(Json(req.execute(db).await?.into()))
}

impl Request {
    #[instrument(skip_all)]
    async fn execute(&self, mut db: MutexGuard<'_, Database>) -> Result<Vec<Category>> {
        if self.category_ids.is_empty() {
            return Ok(Vec::new());
        }

        let mut builder = QueryBuilder::new(
            "
WITH updates(id, sort_position) AS (
    VALUES (",
        );

        let mut sep = builder.separated("),(");
        for (pos, id) in self.category_ids.iter().enumerate() {
            sep.push(id.to_string()).push_unseparated(",").push_unseparated(pos.to_string());
        }

        builder.push(
            "))
UPDATE categories
SET
    sort_position = updates.sort_position
FROM
    updates
WHERE
    categories.id = updates.id
RETURNING *",
        );

        let mut cats: Vec<Category> = db.fetch_all(builder.build_query_as()).await?;
        cats.sort_by_key(RssStruct::id);
        Ok(cats)
    }
}
