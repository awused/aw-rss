use std::convert::Infallible;
use std::string::ToString;
use std::time::Duration;

use chrono::{DateTime, Utc};
use color_eyre::eyre::{OptionExt, eyre};
use color_eyre::{Result, Section, SectionExt};
use futures_util::future::select;
use humantime::format_duration;
use reqwest::header::{ETAG, EXPIRES, IF_MODIFIED_SINCE, IF_NONE_MATCH, LAST_MODIFIED};
use reqwest::{RequestBuilder, StatusCode};
use shlex::Shlex;
use tokio::process::Command;
use tokio::sync::MutexGuard;
use tokio::time::Instant;
use tokio::{pin, time};

use super::{FeedFetcher, HostData};
use crate::com::feed::Failing;
use crate::com::{CLIENT, RssStruct};
use crate::database::Database;
use crate::fetcher::HostKind;
use crate::parsing::{ParsedFeed, parse_feed};

const DEFAULT_POLL_PERIOD: Duration = Duration::from_secs(60 * 30);
// 6 Hours
const MAX_POLL_PERIOD: Duration = Duration::from_secs(60 * 60 * 6);
// 15 Minutes, possibly even too low
const MIN_POLL_PERIOD: Duration = Duration::from_secs(60 * 15);

// A very generous timeout, more to catch stuck processes
const EXECUTABLE_TIMEOUT: Duration = Duration::from_secs(60 * 10);


pub(super) enum Status {
    Success(Headers),
    Failing(Duration),
}

#[derive(Debug)]
enum Body {
    Success(String),
    NotModified,
}

#[derive(Debug, Default)]
pub(super) struct Headers {
    // A duration calculated using the expires header
    expires: Option<Duration>,
    ttl: Option<Duration>,
    last_modified: Option<String>,
    etag: Option<String>,
}

struct Response {
    body: Body,
    headers: Headers,
}


// TODO -- implement Future or IntoFuture and make this nameable once imp Trait works properly
impl FeedFetcher<'_> {
    #[instrument(level = "info", skip_all)]
    async fn fetch_http(&mut self) -> Result<Response> {
        trace!("Fetching HTTP feed");

        let mut headers = self.status.take_headers();

        let resp = headers.apply(CLIENT.get(&self.feed.url)).send().await?;

        headers.merge_from(&resp);

        let body = if resp.status() == StatusCode::NOT_MODIFIED {
            debug!("Not modified");
            Body::NotModified
        } else if resp.status().is_success() {
            Body::Success(resp.text().await?)
        } else {
            return Err(eyre!("Got error code {}", resp.status())
                .section(resp.text().await?.header("Body: ")));
        };


        Ok(Response { body, headers })
    }

    #[instrument(level = "info", skip_all)]
    async fn run_executable(&mut self) -> Result<Response> {
        // If this fails, something unsafe has happened and we should crash
        assert!(self.feed.url.starts_with('!'));
        trace!("Running external executable");

        let mut args = Shlex::new(&self.feed.url[1..]);
        let mut cmd = Command::new(args.next().ok_or_eyre("Invalid command line string")?);
        cmd.args(args).kill_on_drop(true);

        let headers = self.status.take_headers();
        if let Some(etag) = &headers.etag {
            cmd.arg("--etag").arg(etag);
        }

        let output = time::timeout(EXECUTABLE_TIMEOUT, cmd.output()).await??;

        let stdout = String::from_utf8(output.stdout)?;

        if output.status.success() {
            if stdout.len() < 100 && stdout.trim() == "not modified" {
                debug!("Not modified");
                Ok(Response { body: Body::NotModified, headers })
            } else {
                Ok(Response { body: Body::Success(stdout), headers })
            }
        } else {
            let stderr = String::from_utf8_lossy(&output.stderr);
            Err(eyre!("Error running command: {:?}", output.status)
                .section(stdout.header("Stdout:"))
                .section(stderr.trim().to_string().header("Stderr:")))
        }
    }

    #[instrument(level = "error", skip_all, err(Debug))]
    async fn fetch(&mut self) -> Result<()> {
        let Response { body, mut headers } = match self.host.kind {
            HostKind::Http => self.fetch_http().await?,
            HostKind::Executable => self.run_executable().await?,
        };

        if let Body::Success(body) = body {
            let ParsedFeed { update, items, ttl, extension_etag } =
                parse_feed(&self.feed, &body).with_section(|| body.header("Body: "))?;
            headers.ttl = ttl.or(headers.ttl);
            headers.etag = extension_etag.or(headers.etag);

            debug!("Parsed {} items and feed update: {update:?}", items.len());
            // trace!("Items {items:?}");

            // The time spent waiting for the DB lock and writing values is unimportant for
            // calculating the next_fetch time.
            let db = self.db.lock().await;
            Database::handle_parsed(db, &self.feed, update, items).await?;
        }

        trace!("{headers:?}");

        self.next_fetch = headers.next_fetch();
        self.status = Status::Success(headers);

        Ok(())
    }

    async fn wait(&self) {
        let sleep = time::sleep_until(self.next_fetch);
        pin!(sleep);
        select(sleep, self.rerun.listen()).await;
    }

    async fn fail(&mut self, mut guard: MutexGuard<'_, HostData>) {
        let failing = Failing { since: Utc::now().into() };

        let dur = if let Some(dur) = self.status.failing_timeout() {
            dur * 2
        } else {
            // Only increment failing feeds when this starts failing
            guard.failing_feeds += 1;
            Duration::from_secs(guard.failing_feeds.saturating_mul(60))
        };
        drop(guard);

        let db = self.db.lock().await;

        // Update the DB even if we think this was already failing, in case something else edited
        // the DB.
        match Database::single_edit(db, self.feed.id(), failing).await {
            Ok(o) => self.feed = o.take(),
            Err(e) => error!("{:?}", e.wrap_err("Failed to mark feed as failing")),
        }

        self.status = Status::Failing(dur.min(MAX_POLL_PERIOD));

        warn!("Retrying in {}", format_duration(dur));
        let sleep = time::sleep(dur);
        pin!(sleep);
        select(sleep, select(self.rerun.listen(), self.rerun_failing.listen())).await;
    }

    #[instrument(level = "error", skip(self), fields(feed = %self.feed))]
    pub(super) async fn run(mut self) -> Infallible {
        loop {
            // We only want one in-flight request per host
            let mut guard = self.host.lock.lock().await;
            match self.fetch().await {
                Ok(_) => {
                    // Every time any feed succeeds, decrement, so small numbers of failing feeds
                    // cannot lock down a host.
                    guard.failing_feeds = guard.failing_feeds.saturating_sub(1);
                    drop(guard);
                    self.wait().await;
                }
                Err(_) => self.fail(guard).await,
            }
        }
    }
}

impl Status {
    const fn failing_timeout(&self) -> Option<Duration> {
        match self {
            Self::Success(_) => None,
            Self::Failing(dur) => Some(*dur),
        }
    }

    fn take_headers(&mut self) -> Headers {
        match self {
            Self::Success(h) => std::mem::take(h),
            Self::Failing(_) => Headers::default(),
        }
    }
}

impl Headers {
    fn merge_from(&mut self, resp: &reqwest::Response) {
        self.expires = resp
            .headers()
            .get(EXPIRES)
            .and_then(|h| h.to_str().ok())
            .and_then(|e| DateTime::parse_from_rfc2822(e).ok())
            .and_then(|d| d.signed_duration_since(Utc::now()).abs().to_std().ok())
            .or(self.expires);

        if !resp.status().is_success() {
            // Only take new etag/last-modified headers from real fetches, do not trust them from
            // failures or 304s. At least some feed generators (SB/SV) are broken and will set
            // last-modified to the fetch time and then keep the "real" time in the past despite
            // recent updates.
            return;
        }


        let last_modified = resp
            .headers()
            .get(LAST_MODIFIED)
            .and_then(|h| h.to_str().ok())
            .filter(|e| DateTime::parse_from_rfc2822(e).is_ok());
        if self.last_modified.as_deref() != last_modified {
            self.last_modified = last_modified.map(ToString::to_string);
        }

        let etag = resp.headers().get(ETAG).and_then(|h| h.to_str().ok()).filter(|e| !e.is_empty());
        if self.etag.as_deref() != etag {
            self.etag = etag.map(ToString::to_string);
        }
    }

    fn apply(&self, mut req: RequestBuilder) -> RequestBuilder {
        // etag _should_ be more reliable than last-modified, and doesn't suffer from resolution
        // limits, so trust etag alone even when both are present.
        // I've found some sites can change etag without properly updating last-modified.
        if let Some(etag) = &self.etag {
            req = req.header(IF_NONE_MATCH, etag);
        } else if let Some(last_modified) = &self.last_modified {
            req = req.header(IF_MODIFIED_SINCE, last_modified);
        }

        req
    }

    fn next_fetch(&self) -> Instant {
        // rss ttl > expired header > default -> clamp(min/max)
        let poll_duration = self
            .ttl
            .or(self.expires)
            .unwrap_or(DEFAULT_POLL_PERIOD)
            .clamp(MIN_POLL_PERIOD, MAX_POLL_PERIOD);
        trace!("Calculated sleep duration {poll_duration:?}");

        Instant::now() + poll_duration
    }
}
