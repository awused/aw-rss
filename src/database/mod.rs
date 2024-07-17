use std::collections::HashMap;
use std::path::Path;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::Result;
use derive_more::From;
use once_cell::unsync::Lazy;
use sqlx::sqlite::SqliteConnectOptions;
use sqlx::{migrate, Connection, Sqlite, SqliteConnection};
use tokio::sync::MutexGuard;

use crate::com::feed::ParsedUpdate;
use crate::com::item::ParsedInsert;
use crate::com::{
    Category, Feed, Insert, Item, LazyBuilder, OnConflict, Outcome, QueryBuilder, RssQueryAs,
    RssStruct, Update, UtcDateTime,
};
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
    ) -> Result<Outcome<T>> {
        let mut tx = guard.transaction().await?;
        let edited = tx.update(id, edit).await?;
        match edited {
            Outcome::NoOp(_) => {
                tx.rollback().await?;
                debug!("No-op update");
            }
            Outcome::Update(_) => {
                tx.commit().await?;
                info!("Update applied");
            }
        }

        Ok(edited)
    }

    #[instrument(skip(guard))]
    pub async fn single_insert<T: RssStruct, I: Insert<T>>(
        mut guard: MutexGuard<'_, Self>,
        insert: I,
    ) -> Result<T> {
        let mut tx = guard.transaction().await?;
        let inserted = tx.insert(insert).await?;
        info!("Inserted into {}: {inserted:?}", T::table_name());
        tx.commit().await?;
        Ok(inserted)
    }

    #[instrument(skip(guard, feed))]
    pub async fn handle_parsed(
        mut guard: MutexGuard<'_, Self>,
        feed: &Feed,
        feed_update: ParsedUpdate,
        item_inserts: Vec<ParsedInsert>,
    ) -> Result<Outcome<(Feed, u64)>> {
        let mut tx = guard.transaction().await?;
        let updated = tx.update(feed.id(), feed_update).await?;
        let num_inserted = tx.bulk_insert(item_inserts).await?;

        match (updated, num_inserted) {
            (Outcome::NoOp(feed), 0) => {
                debug!("No changes after applying updates");
                tx.rollback().await?;
                Ok(Outcome::NoOp((feed, 0)))
            }
            (Outcome::Update(feed) | Outcome::NoOp(feed), n) => {
                debug!("Inserted {n} new items");
                tx.commit().await?;
                Ok(Outcome::Update((feed, n)))
            }
        }
    }

    #[instrument(skip_all)]
    pub async fn fetch_all<'a, T: RssStruct>(
        &mut self,
        query: RssQueryAs<'a, T>,
    ) -> Result<Vec<T>> {
        query.fetch_all(self.con()?).await.map_err(Into::into)
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

    #[instrument(skip(self))]
    async fn update<T: RssStruct>(&mut self, id: i64, edit: impl Update<T>) -> Result<Outcome<T>> {
        let s: T = self.get(id).await?;
        edit.validate(&s)?;

        let mut builder: LazyBuilder<'_> = Lazy::new(|| {
            QueryBuilder::new(format!(
                "UPDATE {} SET commit_timestamp = CURRENT_TIMESTAMP ",
                T::table_name()
            ))
        });

        edit.build_updates(&s, &mut builder);

        let Ok(mut edit) = Lazy::into_value(builder) else {
            return Ok(Outcome::NoOp(s));
        };

        Ok(Outcome::Update(
            edit.push(" WHERE id = ")
                .push_bind(id)
                .push(" RETURNING *")
                .build_query_as()
                .fetch_one(self.con())
                .await?,
        ))
    }

    fn start_insert<'a, T: RssStruct, I: Insert<T>>() -> QueryBuilder<'a> {
        let mut builder = QueryBuilder::new(format!("INSERT INTO {}(", T::table_name()));

        let cols = I::columns();
        let mut sep = builder.separated(", ");
        for col in cols {
            sep.push(col);
        }

        builder.push(") ");
        builder
    }

    fn insert_conflict<T: RssStruct, I: Insert<T>>(builder: &mut QueryBuilder<'_>) {
        match I::on_conflict() {
            OnConflict::Error => {}
            OnConflict::Ignore => {
                builder.push(" ON CONFLICT IGNORE ");
            }
        }
    }

    #[instrument(skip(self))]
    async fn insert<T: RssStruct, I: Insert<T>>(&mut self, insert: I) -> Result<T> {
        insert.validate()?;

        let mut builder = Self::start_insert::<T, I>();

        builder.push_values([insert; 1], |mut s, ins| ins.push_values(&mut s));

        Self::insert_conflict::<T, I>(&mut builder);

        Ok(builder.push(" RETURNING *").build_query_as().fetch_one(self.con()).await?)
    }

    #[instrument(skip(self))]
    async fn bulk_insert<T: RssStruct, I: Insert<T>>(&mut self, inserts: Vec<I>) -> Result<u64> {
        inserts.iter().try_for_each(I::validate)?;

        let mut builder = Self::start_insert::<T, I>();

        builder.push_values(inserts, |mut s, ins| ins.push_values(&mut s));

        Self::insert_conflict::<T, I>(&mut builder);

        Ok(builder.build().execute(self.con()).await.map(|r| r.rows_affected())?)
    }
}
