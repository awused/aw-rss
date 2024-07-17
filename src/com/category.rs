use std::fmt::Debug;

use color_eyre::eyre::bail;
use color_eyre::Result;
use serde::{Deserialize, Serialize};
use sqlx::prelude::FromRow;

use super::{Insert, RssStruct, Update, UtcDateTime};
use crate::com::HttpError;

#[derive(Serialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Category {
    id: i64,
    /// Disabled categories are effectively deleted, but hang around so
    /// that frontends are not inconvenienced. Feeds will destroy their
    /// relationships with this category.
    /// Any new categories will completely overwrite a disabled category.
    pub disabled: bool,
    /// A short name for the category
    /// Consists of only lowercase letters and hyphens
    /// The frontend will do its best to redirect /:name to /category/:name
    pub name: String,
    pub title: String,
    /// Hidden in the nav bar unless open or ?all=true is specified
    pub hidden_nav: bool,
    /// Doesn't contribute to unread counts and doesn't show up in the default list.
    /// Implied by hidden_nav.
    pub hidden_main: bool,
    #[serde(serialize_with = "UtcDateTime::ts_serialize")]
    pub commit_timestamp: UtcDateTime,
    /// Categories without sort positions are sorted by their IDs, after any
    /// categories with sort positions
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sort_position: Option<i64>,
}

impl RssStruct for Category {
    fn id(&self) -> i64 {
        self.id
    }

    fn table_name() -> &'static str {
        "categories"
    }
}


#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UserInsert {
    name: String,
    title: String,
    hidden_nav: bool,
    hidden_main: bool,
}

fn validate_name(name: &str) -> Result<()> {
    if name.is_empty() || !name.chars().next().unwrap().is_ascii_lowercase() {
        bail!(HttpError::bad("Category name must start with a lowercase ASCII character"));
    }

    if name.chars().any(|c| c != '-' && !c.is_ascii_lowercase() && !c.is_ascii_digit()) {
        bail!(HttpError::bad(
            "Category names must only contain lowercase alphanumeric characters and hyphens"
        ));
    }

    Ok(())
}

impl Insert<Category> for UserInsert {
    fn columns() -> &'static [&'static str] {
        &["name", "title", "hidden_nav", "hidden_main"]
    }

    fn validate(&self) -> Result<()> {
        if self.title.is_empty() {
            bail!(HttpError::bad("Category title cannot be empty"));
        }

        validate_name(&self.name)
    }

    fn push_values(self, builder: &mut super::Separated<'_, '_>) {
        builder
            .push_bind(self.name)
            .push_bind(self.title)
            .push_bind(self.hidden_nav)
            .push_bind(self.hidden_main);
    }
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UserEdit {
    #[serde(default, deserialize_with = "super::empty_string_is_none")]
    name: Option<String>,
    #[serde(default, deserialize_with = "super::empty_string_is_none")]
    title: Option<String>,
    hidden_nav: Option<bool>,
    hidden_main: Option<bool>,
    #[serde(default)]
    disabled: bool,
}

impl Update<Category> for UserEdit {
    fn validate(&self, s: &Category) -> Result<()> {
        if let Some(name) = &self.name {
            validate_name(name)?;
        }

        if self.title.as_ref().is_some_and(String::is_empty) {
            bail!(HttpError::bad("Category title cannot be empty"));
        }

        if !self.disabled && s.disabled {
            bail!(HttpError::bad("Categories cannot be re-enabled"));
        }

        Ok(())
    }

    fn build_updates<'a>(self, s: &'a Category, builder: &mut super::LazyBuilder<'a>) {
        if self.disabled {
            // Disabling categories is a weird case because they can't be re-enabled yet.
            // But it should be possible in the future, so feeds.category_id isn't nulled out.
            // The name is set to a unique but invalid name so it won't conflict in the future.
            //
            // Manual restoration is possible.
            builder.push(", disabled = 1, name = ").push_bind(s.id.to_string());
            return;
        }

        if let Some(name) = self.name {
            if s.name != name {
                builder.push(", name = ").push_bind(name);
            }
        }

        if let Some(title) = self.title {
            if s.title != title {
                builder.push(", title = ").push_bind(title);
            }
        }

        if let Some(hnav) = self.hidden_nav {
            if s.hidden_nav != hnav {
                builder.push(", hidden_nav = ").push_bind(hnav);
            }
        }

        if let Some(hmain) = self.hidden_main {
            if s.hidden_main != hmain {
                builder.push(", hidden_main = ").push_bind(hmain);
            }
        }
    }
}

impl Debug for Category {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "[Category {}: {} ({})", self.id, self.name, self.title,)?;

        if self.disabled {
            f.write_str(", disabled")?;
        } else if self.hidden_nav {
            write!(f, ", hidden_nav")?;
        } else if self.hidden_main {
            write!(f, ", hidden_main")?;
        }


        f.write_str("]")
    }
}
