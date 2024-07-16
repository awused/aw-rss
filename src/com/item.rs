use std::fmt::Debug;

use chrono::Utc;
use serde::Serialize;
use sha2::{Digest, Sha256};
use sqlx::prelude::FromRow;

use super::{LazyBuilder, RssStruct, Update, UtcDateTime};
use crate::com::extract_atom_url;

#[derive(Serialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Item {
    id: i64,
    feed_id: i64,
    // Unfortunately not guaranteed to be utf-8 in practice
    #[serde(skip)]
    key: Vec<u8>,
    pub title: String,
    pub url: String,
    pub timestamp: UtcDateTime,
    pub read: bool,
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

// Represents all the things a user is allowed to do to an item,
// which is currently only setting read/unread.
#[derive(Debug)]
pub struct UserEdit {
    pub read: bool,
}

impl Update<Item> for UserEdit {
    fn build_updates(self, item: &Item, builder: &mut LazyBuilder<'_>) {
        if item.read != self.read {
            builder.push(", read = ").push_bind(self.read);
        }
    }
}

#[derive(Debug)]
pub struct ParsedInsert {
    feed_id: i64,
    key: Vec<u8>,
    title: String,
    url: String,
    timestamp: UtcDateTime,
}

impl From<(i64, rss::Item)> for ParsedInsert {
    fn from((feed_id, item): (i64, rss::Item)) -> Self {
        let title = item.title.clone().unwrap_or_default();
        let key = item
            .guid
            .and_then(|g| (!g.value.is_empty()).then_some(g.value))
            .or_else(|| item.title.and_then(|t| item.pub_date.as_ref().map(|p| t + p)))
            .unwrap_or_else(|| {
                let mut hasher = Sha256::new();
                hasher.update(item.description.unwrap_or_default());
                format!("{:X}", hasher.finalize())
            });

        let timestamp = item
            .pub_date
            .as_ref()
            .and_then(|d| dateparser::parse(d).ok())
            .unwrap_or_else(|| {
                warn!("Got item with no timestamp in {feed_id}: {:?}", item.link);
                Utc::now()
            });

        Self {
            feed_id,
            key: key.into(),
            title,
            url: item.link.unwrap_or_default(),
            timestamp: timestamp.into(),
        }
    }
}

impl From<(i64, atom_syndication::Entry)> for ParsedInsert {
    fn from((feed_id, entry): (i64, atom_syndication::Entry)) -> Self {
        let title = entry.title.to_string();
        let url = extract_atom_url(entry.links).unwrap_or_default();

        let key = if !entry.id.is_empty() {
            entry.id
        } else {
            warn!("Got atom item with no ID, this shouldn't happen {feed_id} ({url})");
            todo!();
        };

        let timestamp = entry.published.unwrap_or(entry.updated).to_utc();

        Self {
            feed_id,
            key: key.into(),
            title,
            url,
            timestamp: timestamp.into(),
        }
    }
}

impl Debug for Item {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "[Item {} - feed {}: {} ({}), {}",
            self.id, self.feed_id, self.url, self.title, self.timestamp,
        )?;

        if self.read {
            f.write_str(" read")?;
        }

        f.write_str("]")
    }
}
