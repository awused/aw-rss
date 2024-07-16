use std::fmt::Debug;

use serde::{Deserialize, Serialize};
use sqlx::prelude::FromRow;

use super::{LazyBuilder, RssStruct, Update, UtcDateTime};
use crate::quirks;

#[derive(Serialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Feed {
    id: i64,
    pub url: String,
    pub disabled: bool,
    pub title: String,
    pub site_url: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub user_title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub category_id: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub failing_since: Option<UtcDateTime>,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    pub commit_timestamp: UtcDateTime,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    pub create_timestamp: UtcDateTime,
}

impl RssStruct for Feed {
    fn id(&self) -> i64 {
        self.id
    }

    fn table_name() -> &'static str {
        "feeds"
    }
}


// Represents all the things a user is allowed to do to an item,
// which is currently only setting read/unread.
#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UserEdit {
    pub category_id: Option<i64>,
    #[serde(default)]
    pub clear_category: bool,
    pub disabled: Option<bool>,
    pub user_title: Option<String>,
}

impl Update<Feed> for UserEdit {
    fn build_updates(self, feed: &Feed, builder: &mut LazyBuilder<'_>) {
        if let Some(cat) = self.category_id {
            if feed.category_id != Some(cat) {
                builder.push(", category_id = ").push_bind(cat);
            }
        } else if self.clear_category && feed.category_id.is_some() {
            builder.push(", category_id = NULL ");
        }

        if let Some(disable) = self.disabled {
            if disable != feed.disabled {
                builder.push(", disabled = ").push_bind(disable);
            }
        }

        if let Some(ut) = self.user_title {
            if ut != feed.user_title {
                builder.push(", user_title = ").push_bind(ut);
            }
        }
    }
}

#[derive(Debug)]
pub struct ParsedUpdate {
    pub title: String,
    pub link: Option<String>,
}

impl Update<Feed> for ParsedUpdate {
    fn build_updates<'a>(self, feed: &'a Feed, builder: &mut LazyBuilder<'a>) {
        if self.title != feed.title {
            builder.push(", title = ").push_bind(self.title);
        }

        if let Some(link) = self.link {
            let link = quirks::site_url(link, feed);
            if link != feed.site_url {
                builder.push(", site_url = ").push_bind(link);
            }
        } else if feed.site_url.is_empty() && !feed.site_url.starts_with('!') {
            warn!("Feed without site link even after update {feed:?}");
            builder.push(", site_url = ").push_bind(&feed.url);
        }

        if feed.failing_since.is_some() {
            builder.push(", failing_since = NULL ");
        }
    }
}

#[derive(Debug)]
pub struct Failing {
    pub since: UtcDateTime,
}

impl Update<Feed> for Failing {
    fn build_updates<'a>(self, s: &'a Feed, builder: &mut LazyBuilder<'a>) {
        if s.failing_since.is_none() {
            builder.push(", failing_since = ").push_bind(self.since);
        }
    }
}


impl Debug for Feed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        // Very compact since this gets logged a lot
        write!(
            f,
            "[Feed {}: {}",
            self.id,
            self.url,
            // if self.user_title.is_empty() { &self.title } else { &self.user_title },
        )?;

        if self.disabled {
            f.write_str(" disabled")?;
        }

        if let Some(s) = self.failing_since {
            write!(f, ", failing: {s}")?;
        }

        f.write_str("]")
    }
}

impl Debug for UserEdit {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let mut d = f.debug_struct("UserEdit");

        if let Some(category_id) = self.category_id {
            d.field("category_id", &category_id);
        }

        if self.clear_category {
            d.field("clear_category", &self.clear_category);
        }

        if let Some(disabled) = self.disabled {
            d.field("disabled", &disabled);
        }

        if let Some(user_title) = &self.user_title {
            d.field("user_title", user_title);
        }

        d.finish()
    }
}
