use std::fmt::Debug;

use serde::Serialize;
use sqlx::prelude::FromRow;

use super::{RssStruct, UtcDateTime};

#[derive(Serialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Category {
    pub id: i64,
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
    /// Doesn't contribute to unread counts and doesn't show up in the default view.
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
