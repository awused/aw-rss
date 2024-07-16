use std::collections::HashMap;
use std::path::Path;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::Result;
use derive_more::From;
use sqlx::sqlite::SqliteConnectOptions;
use sqlx::{migrate, Connection, Sqlite, SqliteConnection};
use tokio::sync::MutexGuard;

use crate::com::{Category, EditResult, Feed, Item, RssQueryAs, RssStruct, Update, UtcDateTime};
use crate::config::CONFIG;

#[derive(Debug, From)]
pub struct Transaction<'a>(sqlx::Transaction<'a, Sqlite>);


#[derive(Debug)]
pub struct Database {
    connection: Option<SqliteConnection>,
}

impl Drop for Database {
    fn drop(&mut self) {
        if let Some(_con) = self.connection.take() {
            error!("Database dropped without being closed");
        }
    }
}

impl Database {
    #[instrument]
    pub async fn new() -> Result<Self> {
        if CONFIG.database == Path::new(":memory:") {
            warn!("Using in-memory database, state will not persist between runs");
        } else {
            info!("Using database {:?}", CONFIG.database);
        }

        let options = SqliteConnectOptions::new()
            .filename(&CONFIG.database)
            .create_if_missing(true)
            .busy_timeout(Duration::from_secs(30))
            // This is the default, but be explicit
            .foreign_keys(true)
            .thread_name(|id| format!("sqlite-worker-{id}"));

        let mut con = SqliteConnection::connect_with(&options).await?;

        // Check if the sqlx migrations table exists, if not, run a vacuum after the initial
        // migrations.
        let run_vacuum = sqlx::query_scalar::<_, String>(
            "SELECT name FROM sqlite_master WHERE type = 'table' AND name = '_sqlx_migrations'",
        )
        .fetch_optional(&mut con)
        .await?
        .is_none();

        info!("Running Database migrations");
        let migrations = migrate!("./src/database/migrations");
        migrations.run(&mut con).await?;

        if run_vacuum {
            info!("Running VACUUM on initial migration, this may take a while.");
            sqlx::query("VACUUM").execute(&mut con).await?;
        }

        Ok(Self { connection: Some(con) })
    }

    fn con(&mut self) -> Result<&mut SqliteConnection> {
        self.connection.as_mut().ok_or(sqlx::Error::PoolClosed.into())
    }

    #[instrument(skip_all)]
    pub async fn close(&mut self) -> Result<()> {
        let con = self.connection.take().ok_or(sqlx::Error::PoolClosed)?;

        con.close().await.map_err(Into::into)
    }

    // TODO -- remove these
    #[instrument(skip(self))]
    pub async fn get_feed(&mut self, id: i64) -> Result<Feed> {
        let con = self.con()?;

        let feed = sqlx::query_as("SELECT * FROM feeds WHERE id = ?")
            .bind(id)
            .fetch_one(con)
            .await?;
        info!("getting feed {feed:?}");
        Ok(feed)
    }

    #[instrument(skip(self))]
    pub async fn get_item(&mut self, id: i64) -> Result<Item> {
        let con = self.con()?;

        let item = sqlx::query_as("SELECT * FROM items WHERE id = ?")
            .bind(id)
            .fetch_one(con)
            .await?;
        info!("getting feed {item:?}");
        Ok(item)
    }

    #[instrument(skip(self))]
    pub async fn get_category(&mut self, id: i64) -> Result<Category> {
        let cat = sqlx::query_as("SELECT * FROM categories WHERE id = ?")
            .bind(id)
            .fetch_one(self.con()?)
            .await?;
        info!("getting feed {cat:?}");
        Ok(cat)
    }

    #[instrument(skip(guard))]
    pub async fn single_edit<T: RssStruct, E: Update<T>>(
        mut guard: MutexGuard<'_, Self>,
        id: i64,
        edit: E,
    ) -> Result<EditResult<T>> {
        let mut tx = guard.transaction().await?;
        let edited = T::update(id, &mut tx, edit).await?;
        match edited {
            EditResult::NoOp(_) => {
                tx.rollback().await?;
                debug!("No-op update");
            }
            EditResult::Update(_) => {
                tx.commit().await?;
                info!("Update applied");
            }
        }

        Ok(edited)
    }

    #[instrument(skip_all)]
    pub async fn fetch_all<'a, T: RssStruct>(
        &mut self,
        query: RssQueryAs<'a, T>,
    ) -> Result<Vec<T>> {
        let con = self.con()?;
        query.fetch_all(con).await.map_err(Into::into)
    }

    #[instrument(skip(self))]
    pub async fn disabled_feeds(&mut self) -> Result<Vec<Feed>> {
        sqlx::query_as("SELECT * FROM feeds WHERE disabled = 1 ORDER BY id ASC")
            .fetch_all(self.con()?)
            .await
            .map_err(Into::into)
    }

    pub async fn transaction(&mut self) -> Result<Transaction> {
        Ok(self.con()?.begin().await?.into())
    }
}


impl Transaction<'_> {
    pub fn con(&mut self) -> &mut SqliteConnection {
        &mut self.0
    }

    #[instrument(skip_all)]
    pub async fn timestamp(&mut self) -> Result<i64> {
        let dt: DateTime<Utc> =
            sqlx::query_scalar("SELECT datetime('now')").fetch_one(self.con()).await?;
        Ok(dt.timestamp() - 1)
    }

    #[instrument(skip_all)]
    pub async fn commit(self) -> Result<()> {
        self.0.commit().await.map_err(Into::into)
    }

    #[instrument(skip_all)]
    pub async fn rollback(self) -> Result<()> {
        self.0.rollback().await.map_err(Into::into)
    }

    #[instrument(skip_all)]
    pub async fn newest_timestamps(&mut self) -> Result<HashMap<i64, DateTime<Utc>>> {
        let newest: Vec<(i64, DateTime<Utc>)> = sqlx::query_as(
            "
SELECT
    feed_id,
    MAX(items.timestamp)
FROM items
INNER JOIN feeds
    ON feeds.id = items.feed_id
WHERE feeds.disabled = 0
GROUP BY feed_id",
        )
        .fetch_all(self.con())
        .await?;

        Ok(newest.into_iter().collect())
    }

    pub async fn current_categories(&mut self) -> Result<Vec<Category>> {
        sqlx::query_as("SELECT * FROM categories WHERE disabled = 0 ORDER BY id ASC")
            .fetch_all(self.con())
            .await
            .map_err(Into::into)
    }

    pub async fn current_feeds(&mut self) -> Result<Vec<Feed>> {
        sqlx::query_as("SELECT * FROM feeds WHERE disabled = 0 ORDER BY id ASC")
            .fetch_all(self.con())
            .await
            .map_err(Into::into)
    }

    pub async fn current_items(&mut self) -> Result<Vec<Item>> {
        sqlx::query_as(
            "
SELECT items.* FROM
    feeds CROSS JOIN items ON items.feed_id = feeds.id
WHERE
    feeds.disabled = 0 AND items.read = 0
ORDER BY items.id ASC",
        )
        .fetch_all(self.con())
        .await
        .map_err(Into::into)
    }

    #[instrument(skip(self))]
    pub async fn updated<T: RssStruct>(&mut self, timestamp: UtcDateTime) -> Result<Vec<T>> {
        let table = T::table_name();
        sqlx::query_as(&format!(
            "
SELECT * FROM
    {table} INDEXED BY {table}_commit_index
WHERE
    commit_timestamp > ?
ORDER BY id ASC"
        ))
        .bind(timestamp)
        .fetch_all(self.con())
        .await
        .map_err(Into::into)
    }

    #[instrument(skip(self))]
    pub async fn get<T: RssStruct>(&mut self, id: i64) -> Result<T> {
        let table = T::table_name();
        sqlx::query_as(&format!(
            "
SELECT * FROM
    {table}
WHERE
    id = ?
ORDER BY id ASC"
        ))
        .bind(id)
        .fetch_one(self.con())
        .await
        .map_err(Into::into)
    }
}
