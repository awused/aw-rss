use std::convert::Infallible;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::eyre::eyre;
use color_eyre::{Result, Section, SectionExt};
use futures_util::future::select;
use tokio::process::Command;
use tokio::time::Instant;
use tokio::{pin, time};

use super::FeedFetcher;
use crate::com::feed::Failing;
use crate::com::{RssStruct, CLIENT};
use crate::database::Database;
use crate::fetcher::HostKind;
use crate::parsing::{parse_feed, ParsedFeed};

const DEFAULT_POLL_PERIOD: Duration = Duration::from_secs(60 * 30);
// 6 Hours
const MAX_POLL_PERIOD: Duration = Duration::from_secs(60 * 60 * 6);
// 15 Minutes, possibly even too low
const MIN_POLL_PERIOD: Duration = Duration::from_secs(60 * 15);

// A very generous timeout, more to catch stuck processes
const EXECUTABLE_TIMEOUT: Duration = Duration::from_secs(60 * 10);

// TODO -- implement Future or IntoFuture and make this nameable once imp Trait works properly
impl<'a> FeedFetcher<'a> {
    #[instrument(skip_all)]
    async fn fetch_http(&mut self) -> Result<(String, Option<DateTime<Utc>>)> {
        debug!("Fetching");
        let resp = CLIENT.get(&self.feed.url).send().await?;
        let expires = resp
            .headers()
            .get("Expires")
            .and_then(|h| h.to_str().ok())
            .and_then(|e| DateTime::parse_from_rfc2822(e).ok())
            .map(|d| d.to_utc());
        trace!("Got expires header {expires:?}");
        Ok((resp.text().await?, expires))
    }

    #[instrument(skip_all)]
    async fn run_executable(&mut self) -> Result<(String, Option<DateTime<Utc>>)> {
        // If this fails, something unsafe has happened.
        assert!(self.feed.url.starts_with('!'));

        let cmd = Command::new("sh")
            .arg("-c")
            .arg(&self.feed.url[1..])
            .kill_on_drop(true)
            .output();

        let output = time::timeout(EXECUTABLE_TIMEOUT, cmd).await??;

        let stdout = String::from_utf8(output.stdout)?;

        if output.status.success() {
            Ok((stdout, None))
        } else {
            let stderr = String::from_utf8_lossy(&output.stderr);
            Err(eyre!("Error running command: {:?}", output.status)
                .section(stdout.header("Stdout:"))
                .section(stderr.trim().to_string().header("Stderr:")))
        }
    }

    async fn fetch(&mut self) -> Result<()> {
        // We only want one in-flight request per host
        let _guard = self.host_data.lock.lock().await;

        let (body, expires) = match self.host_data.kind {
            HostKind::Http => self.fetch_http().await?,
            HostKind::Executable => self.run_executable().await?,
        };

        let ParsedFeed { update, items, ttl } =
            parse_feed(&self.feed, &body).with_section(|| body.header("Body: "))?;

        trace!(
            "Parsed {} items\nfeed: {update:?}\nttl: {ttl:?}, expires: {expires:?}",
            items.len()
        );
        // trace!("Items {items:?}");

        // rss ttl > expired header > default -> clamp(min/max)
        let dur = ttl
            .or_else(|| {
                expires.and_then(|d| d.signed_duration_since(Utc::now()).abs().to_std().ok())
            })
            .unwrap_or(DEFAULT_POLL_PERIOD)
            .clamp(MIN_POLL_PERIOD, MAX_POLL_PERIOD);
        trace!("Calculated sleep duration {dur:?}");

        self.next_fetch = Instant::now() + dur;

        let db = self.db.lock().await;
        Database::handle_parsed(db, &self.feed, update, items).await?;

        self.failing_timeout = None;
        Ok(())
    }

    async fn wait(&mut self) {
        let sleep = time::sleep_until(self.next_fetch);
        pin!(sleep);
        select(sleep, self.rerun.listen()).await;
    }

    async fn fail(&mut self) {
        let failing = Failing { since: Utc::now().into() };

        let db = self.db.lock().await;

        #[allow(clippy::significant_drop_in_scrutinee)]
        match Database::single_edit(db, self.feed.id(), failing).await {
            Ok(o) => self.feed = o.take(),
            Err(e) => {
                error!("{:?}", e.wrap_err("Failed to mark feed as failing"));
            }
        }

        let dur = self
            .failing_timeout
            .map_or(Duration::from_secs(60), |d| d * 2)
            .min(MAX_POLL_PERIOD);
        self.failing_timeout = Some(dur);

        info!("Retrying in {dur:?}");
        let sleep = time::sleep(dur);
        pin!(sleep);
        select(sleep, select(self.rerun.listen(), self.rerun_failing.listen())).await;
    }

    #[instrument(skip(self), fields(feed = %self.feed), level = "error")]
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
