use std::borrow::Cow;
use std::fmt::{Debug, Display};

use chrono::format::StrftimeItems;
use chrono::{DateTime, Timelike, Utc};
use derive_more::derive::From;
use serde::{ser, Deserialize, Serialize};
use sqlx::encode::IsNull;
use sqlx::error::BoxDynError;
use sqlx::sqlite::SqliteArgumentValue;
use sqlx::{Decode, Encode, Sqlite, Type};

// Sqlite uses textual comparison and stores a dumb version of RFC 3339.
// It'll parse fine without this, but for comparisons we need to be exact.
// As long as everything is UTC, this is enough.
const TIMESTAMP_FORMAT: StrftimeItems<'_> = StrftimeItems::new("%F %T");

#[derive(Serialize, Deserialize, Debug, Clone, Copy, From)]
#[serde(transparent)]
pub struct UtcDateTime(DateTime<Utc>);

impl<'q> Encode<'q, Sqlite> for UtcDateTime {
    fn encode_by_ref(
        &self,
        buf: &mut <Sqlite as sqlx::Database>::ArgumentBuffer<'q>,
    ) -> Result<IsNull, BoxDynError> {
        buf.push(SqliteArgumentValue::Text(Cow::Owned(format!(
            "{}",
            self.0.with_nanosecond(0).unwrap().format_with_items(TIMESTAMP_FORMAT)
        ))));
        Ok(IsNull::No)
    }
}

impl<'r> Decode<'r, Sqlite> for UtcDateTime {
    fn decode(value: <Sqlite as sqlx::Database>::ValueRef<'r>) -> Result<Self, BoxDynError> {
        Ok(Self(DateTime::<Utc>::decode(value)?))
    }
}

impl Type<Sqlite> for UtcDateTime {
    fn type_info() -> <sqlx::Sqlite as sqlx::Database>::TypeInfo {
        DateTime::<Utc>::type_info()
    }

    fn compatible(ty: &<sqlx::Sqlite as sqlx::Database>::TypeInfo) -> bool {
        DateTime::<Utc>::compatible(ty)
    }
}

impl Display for UtcDateTime {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        Display::fmt(&self.0, f)
    }
}


impl UtcDateTime {
    pub fn ts_serialize<S>(dt: &Self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: ser::Serializer,
    {
        serializer.serialize_i64(dt.0.timestamp())
    }
}
