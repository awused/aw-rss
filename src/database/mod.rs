use std::collections::HashMap;
use std::path::Path;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::Result;
use derive_more::derive::From;
use once_cell::unsync::Lazy;
use sqlx::sqlite::SqliteConnectOptions;
use sqlx::{migrate, Connection, Sqlite, SqliteConnection};
use tokio::sync::MutexGuard;

use crate::com::feed::ParsedUpdate;
use crate::com::item::ParsedInsert;
use crate::com::{
    Category, Feed, Insert, Item, OnConflict, Outcome, QueryBuilder, RssQueryAs, RssStruct, Update,
    UtcDateTime,
};
use crate::config::CONFIG;

#[derive(Debug, From)]
pub struct Transaction<'a>(Option<sqlx::Transaction<'a, Sqlite>>);

impl Drop for Transaction<'_> {
    fn drop(&mut self) {
        if self.0.is_some() {
            warn!("Transaction dropped without being explicitly committed or rolled back");
        }
    }
}

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

    #[instrument(skip(guard))]
    pub async fn get<T: RssStruct>(mut guard: MutexGuard<'_, Self>, id: i64) -> Result<T> {
        sqlx::query_as(&format!("SELECT * FROM {} WHERE id = ?", T::table_name()))
            .bind(id)
            .fetch_one(guard.con()?)
            .await
            .map_err(Into::into)
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
    pub async fn single_insert<T: RssStruct + std::fmt::Debug, I: Insert<T>>(
        mut guard: MutexGuard<'_, Self>,
        insert: I,
    ) -> Result<T> {
        let mut tx = guard.transaction().await?;
        let inserted = tx.insert(insert).await?;
        info!("Inserted into {}", T::table_name());
        trace!("Value inserted was {inserted:?}");
        tx.commit().await?;
        Ok(inserted)
    }

    #[instrument(skip_all)]
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
                trace!("No changes after applying updates");
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

    #[instrument(skip(guard))]
    pub async fn active_feeds(mut guard: MutexGuard<'_, Self>) -> Result<Vec<Feed>> {
        sqlx::query_as("SELECT * FROM feeds WHERE disabled = 0")
            .fetch_all(guard.con()?)
            .await
            .map_err(Into::into)
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
        Ok(Transaction(self.con()?.begin().await?.into()))
    }
}


impl Transaction<'_> {
    pub fn con(&mut self) -> &mut SqliteConnection {
        self.0.as_mut().unwrap()
    }

    #[instrument(skip_all)]
    pub async fn timestamp(&mut self) -> Result<i64> {
        let dt: DateTime<Utc> =
            sqlx::query_scalar("SELECT datetime('now')").fetch_one(self.con()).await?;
        Ok(dt.timestamp() - 1)
    }

    #[instrument(skip_all)]
    pub async fn commit(mut self) -> Result<()> {
        self.0.take().unwrap().commit().await.map_err(Into::into)
    }

    #[instrument(skip_all)]
    pub async fn rollback(mut self) -> Result<()> {
        self.0.take().unwrap().rollback().await.map_err(Into::into)
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

        let mut builder: Lazy<QueryBuilder<'_>> =
            Lazy::new(|| QueryBuilder::new(format!("UPDATE {} SET ", T::table_name())));

        edit.build_updates(&s, &mut builder);

        let Some(edit) = Lazy::get_mut(&mut builder) else {
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
            OnConflict::Ignore(constraint) => {
                builder.push(" ON CONFLICT").push(constraint).push(" DO NOTHING ");
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

    #[instrument(skip_all, fields(count = inserts.len()))]
    async fn bulk_insert<T: RssStruct, I: Insert<T>>(&mut self, inserts: Vec<I>) -> Result<u64> {
        if inserts.is_empty() {
            return Ok(0);
        }

        inserts.iter().try_for_each(I::validate)?;

        let mut builder = Self::start_insert::<T, I>();

        // Default sqlite limit for versions 3.32.0+
        let hint = I::binds_count_hint();
        if inserts.len() * hint > 32766 {
            error!(
                "Trying to insert too much: {} inserts would need {} binds, which is over the \
                 sqlite maximum. Skipping older items.",
                inserts.len(),
                inserts.len() * hint
            );

            builder.push_values(inserts.into_iter().take(32766 / hint), |mut s, ins| {
                ins.push_values(&mut s)
            });
        } else {
            builder.push_values(inserts, |mut s, ins| ins.push_values(&mut s));
        }

        Self::insert_conflict::<T, I>(&mut builder);

        Ok(builder.build().execute(self.con()).await.map(|r| r.rows_affected())?)
    }
}
