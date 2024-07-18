use std::fmt::Debug;

use color_eyre::eyre::Result;
use once_cell::unsync::Lazy;
use serde::Serialize;
use sqlx::prelude::FromRow;

use super::{Insert, LazyBuilder, OnConflict, RssStruct, Separated, Update, UtcDateTime};
use crate::config::CONFIG;

#[derive(Serialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Item {
    id: i64,
    feed_id: i64,
    // Unfortunately not guaranteed to be utf-8 in practice,
    // but we don't really need it in code.
    // #[serde(skip)]
    // #[allow(dead_code)]
    // key: Vec<u8>,
    title: String,
    url: String,
    timestamp: UtcDateTime,
    read: bool,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    commit_timestamp: UtcDateTime,
}

impl RssStruct for Item {
    fn id(&self) -> i64 {
        self.id
    }

    fn table_name() -> &'static str {
        "items"
    }
}

#[derive(Debug)]
pub struct UserEdit {
    pub read: bool,
}

impl Update<Item> for UserEdit {
    fn validate(&self, _item: &Item) -> Result<()> {
        Ok(())
    }

    fn build_updates(self, item: &Item, builder: &mut LazyBuilder<'_>) {
        if item.read != self.read {
            builder.push(", read = ").push_bind(self.read);
        }
    }
}

#[derive(Debug)]
pub struct ParsedInsert {
    pub feed_id: i64,
    // Force strings on insert, even though we might have some invalid utf-8 in the DB
    pub key: String,
    pub title: String,
    pub url: String,
    pub timestamp: UtcDateTime,
}

thread_local! {
    static DEDUPE: Lazy<bool> = Lazy::new(|| CONFIG.dedupe);
}

const DEDUPE_COLUMNS: [&str; 6] = ["feed_id", "key", "title", "url", "timestamp", "read"];

impl Insert<Item> for ParsedInsert {
    fn columns() -> &'static [&'static str] {
        if DEDUPE.with(|b| **b) { &DEDUPE_COLUMNS } else { &DEDUPE_COLUMNS[0..5] }
    }

    fn binds_count_hint() -> usize {
        7
    }

    fn on_conflict() -> OnConflict {
        OnConflict::Ignore("(feed_id, key)")
    }

    fn validate(&self) -> Result<()> {
        Ok(())
    }

    fn push_values(self, builder: &mut Separated<'_, '_>) {
        builder
            .push_bind(self.feed_id)
            .push_bind(self.key)
            .push_bind(self.title)
            // A bit of a wasteful clone, oh well
            .push_bind(self.url.clone())
            .push_bind(self.timestamp);

        if DEDUPE.with(|b| **b) {
            builder
                .push(" (SELECT EXISTS (SELECT url FROM items WHERE url = ")
                .push_bind_unseparated(self.url)
                .push_unseparated(" AND feed_id != ")
                .push_bind_unseparated(self.feed_id)
                .push_unseparated(")) ");
        }
    }
}


// impl Display for Item {
//     fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
//         write!(
//             f,
//             "[Item {} - feed {}: {} ({}), {}",
//             self.id, self.feed_id, self.url, self.title, self.timestamp,
//         )?;
//
//         if self.read {
//             f.write_str(" read")?;
//         }
//
//         f.write_str("]")
//     }
// }
