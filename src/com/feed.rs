use std::fmt::{Debug, Display};

use color_eyre::Result;
use once_cell::unsync::Lazy;
use serde::{Deserialize, Serialize};
use sqlx::prelude::FromRow;

use super::{Insert, LazyBuilder, RssStruct, Update, UtcDateTime};
use crate::quirks;

#[derive(Serialize, FromRow, Debug)]
#[serde(rename_all = "camelCase")]
pub struct Feed {
    id: i64,
    pub url: String,
    pub disabled: bool,
    title: String,
    pub site_url: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    user_title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    category_id: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    failing_since: Option<UtcDateTime>,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    commit_timestamp: UtcDateTime,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    create_timestamp: UtcDateTime,
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
    fn validate(&self, _feed: &Feed) -> Result<()> {
        Ok(())
    }

    fn build_updates(self, feed: &Feed, builder: &mut LazyBuilder<'_>) {
        let mut sep = Lazy::new(|| { builder }.separated(", "));

        if let Some(cat) = self.category_id {
            if feed.category_id != Some(cat) {
                sep.push(" category_id = ").push_bind_unseparated(cat);
            }
        } else if self.clear_category && feed.category_id.is_some() {
            sep.push(" category_id = NULL ");
        }

        if let Some(disable) = self.disabled {
            if disable != feed.disabled {
                sep.push(" disabled = ").push_bind_unseparated(disable);
            }
        }

        if let Some(ut) = self.user_title {
            if ut != feed.user_title {
                sep.push(" user_title = ").push_bind_unseparated(ut);
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
    fn validate(&self, _feed: &Feed) -> Result<()> {
        Ok(())
    }

    fn build_updates<'a>(self, feed: &'a Feed, builder: &mut LazyBuilder<'a>) {
        let mut sep = Lazy::new(|| { builder }.separated(", "));

        if self.title != feed.title {
            sep.push(" title = ").push_bind_unseparated(self.title);
        }

        if let Some(link) = self.link {
            let link = quirks::site_url(link, feed);
            if link != feed.site_url {
                sep.push(" site_url = ").push_bind_unseparated(link);
            }
        } else if feed.site_url.is_empty() && !feed.url.starts_with('!') {
            warn!("Feed has no site link even after update");
            sep.push(" site_url = ").push_bind_unseparated(&feed.url);
        }

        if feed.failing_since.is_some() {
            sep.push(" failing_since = NULL ");
        }
    }
}

#[derive(Debug)]
pub struct Failing {
    pub since: UtcDateTime,
}

impl Update<Feed> for Failing {
    fn validate(&self, _feed: &Feed) -> Result<()> {
        Ok(())
    }

    fn build_updates<'a>(self, s: &'a Feed, builder: &mut LazyBuilder<'a>) {
        if s.failing_since.is_none() {
            builder.push(" failing_since = ").push_bind(self.since);
        }
    }
}

#[derive(Debug)]
pub struct UserInsert {
    pub url: String,
    pub user_title: String,
    pub category_id: Option<i64>,
}

impl Insert<Feed> for UserInsert {
    fn columns() -> &'static [&'static str] {
        &["url", "user_title", "category_id"]
    }

    fn validate(&self) -> Result<()> {
        Ok(())
    }

    fn push_values(self, builder: &mut super::Separated<'_, '_>) {
        builder
            .push_bind(self.url)
            .push_bind(self.user_title)
            .push_bind(self.category_id);
    }
}


impl Display for Feed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        // Very compact since this gets logged a lot in Context
        let title = if self.user_title.is_empty() { &self.title } else { &self.user_title };
        let title = title
            .char_indices()
            .take(20)
            .last()
            .map_or(&**title, |(n, c)| &title[0..n + c.len_utf8()]);

        write!(f, "[Feed {}: {} ({title})", self.id, self.url,)?;

        if self.disabled {
            f.write_str(" disabled")?;
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
