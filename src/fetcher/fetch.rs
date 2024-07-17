use std::convert::Infallible;
use std::future::pending;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::Result;
use futures_util::future::{select, select_all};
use tokio::pin;
use tokio::time::sleep_until;

use super::FeedFetcher;
use crate::com::feed::Failing;
use crate::com::RssStruct;
use crate::database::Database;

// TODO -- implement Future or IntoFuture and make this nameable once imp Trait works properly
impl<'a> FeedFetcher<'a> {
    const DEFAULT_POLL_PERIOD: Duration = Duration::from_secs(60 * 30);
    // 6 Hours
    const MAX_POLL_PERIOD: Duration = Duration::from_secs(60 * 60 * 6);
    // 15 Minutes, possibly even too low
    const MIN_POLL_PERIOD: Duration = Duration::from_secs(60 * 15);

    // SET TTL
    // rss ttl > expired header > default -> max/min

    async fn fetch(&mut self) -> Result<()> {
        let _guard = self.host_data.lock.lock().await;
        debug!("Fetching");

        Ok(())
    }

    async fn wait(&mut self) {
        let sleep = sleep_until(self.next_fetch);
        pin!(sleep);
        select(sleep, self.rerun.listen()).await;
    }

    async fn fail(&mut self) {
        let failing = Failing { since: Utc::now().into() };

        let db = self.db.lock().await;

        match Database::single_edit(db, self.feed.id(), failing).await {
            Ok(o) => self.feed = o.take(),
            Err(e) => error!("{:?}", e),
        }
    }

    #[instrument(skip(self), fields(feed = ?self.feed))]
    pub(super) async fn run(mut self) -> Infallible {
        loop {
            match self.fetch().await {
                Ok(_) => self.wait().await,
                Err(e) => {
                    error!("{e:?}");
                    self.fail().await;
                }
            }
        }
    }
}
