use std::boxed::Box;
use std::collections::hash_map::Entry;
use std::collections::HashMap;
use std::convert::Infallible;
use std::future::Future;
use std::rc::Rc;
use std::time::Duration;

use color_eyre::eyre::OptionExt;
use color_eyre::Result;
use event_listener::Event;
use futures_util::StreamExt;
use mapped_futures::mapped_futures::MappedFutures;
use tokio::select;
use tokio::sync::mpsc::UnboundedReceiver;
use tokio::sync::Mutex;
use tokio::time::{sleep_until, Instant};
use url::Url;

use crate::closing;
use crate::com::{Action, Feed, RssStruct};
use crate::database::Database;

mod fetch;

#[derive(Debug)]
enum HostKind {
    Http,
    Executable,
}

#[derive(Debug)]
struct HostData {
    kind: HostKind,
    lock: Mutex<()>,
}

struct Manager<'a> {
    receiver: UnboundedReceiver<Action>,
    host_map: HashMap<String, &'static HostData>,
    active_feeds: HashMap<i64, Rc<Event>>,
    poll_deadline: Instant,

    // Can be borrowed by tasks
    db: &'a Mutex<Database>,
    rerun_failing: &'a Event,
}


#[derive(Debug)]
struct FeedFetcher<'a> {
    feed: Feed,
    db: &'a Mutex<Database>,
    host_data: &'static HostData,
    failing_timeout: Option<Duration>,
    // Based on Feed TTL
    next_fetch: Instant,
    rerun: Rc<Event>,
    rerun_failing: &'a Event,
}

const POLL_DURATION: Duration = Duration::from_secs(60 * 5);

impl<'a> Manager<'a> {
    // This should only ever exit if something is seriously broken
    #[instrument(skip_all)]
    async fn run(mut self) {
        // The code will be MUCH nicer once this is nameable (don't want to box)
        let mut tasks = MappedFutures::new();

        let initial = self.poll_db(&mut tasks).await.expect("Failed to load initial feeds");

        for (id, feed) in initial {
            let fetcher = self.build_fetcher(feed);
            assert!(self.active_feeds.insert(id, fetcher.rerun.clone()).is_none());
            assert!(tasks.insert(id, fetcher.run()));
        }

        info!("Loaded {} initial feeds", self.active_feeds.len());

        loop {
            select! {
                biased;
                _ = closing::closed_fut() => break,
                _ = sleep_until(self.poll_deadline) => {
                    let new = match self.poll_db(&mut tasks).await {
                        Ok(new) => new,
                        Err(e) => {
                            error!("{e:?}");
                            continue;
                        }
                    };

                    if !new.is_empty() {
                        info!("Starting {} new feeds after DB poll from manual edits", new.len());
                    }

                    for (id, feed) in new {
                        let fetcher = self.build_fetcher(feed);
                        assert!(self.active_feeds.insert(id, fetcher.rerun.clone()).is_none());
                        assert!(tasks.insert(id, fetcher.run()));
                    }
                }
                msg = self.receiver.recv() => {
                    let Some(msg) = msg else {
                        if closing::close() {
                            error!("Channel closed unexpectedly");
                        }
                        break;
                    };

                    match self.handle(&mut tasks, msg).await {
                        Ok(None) => {},
                        Ok(Some((id, fetcher))) => {
                            assert!(self.active_feeds.insert(id, fetcher.rerun.clone()).is_none());
                            assert!(tasks.insert(id, fetcher.run()));
                        }
                        Err(e) => error!("{e:?}")
                    }
                }
                _ = tasks.next(), if !tasks.is_empty() => {
                    unreachable!()
                }
            }
        }
    }

    #[instrument(skip_all)]
    async fn poll_db(
        &mut self,
        tasks: &mut MappedFutures<i64, impl Future<Output = Infallible>>,
    ) -> Result<HashMap<i64, Feed>> {
        self.poll_deadline += POLL_DURATION;

        // We cannot trust the commit timestamp in the presence of user edits, so this is a full
        // scan.
        let feeds = Database::current_feeds(self.db.lock().await).await?;

        debug!("Loaded {} active feeds", feeds.len());

        let mut alive: HashMap<_, _> = feeds.into_iter().map(|f| (f.id(), f)).collect();

        self.active_feeds.retain(|k, _v| {
            if alive.remove(k).is_some() {
                true
            } else {
                debug!("Cancelling task for disabled feed: {k}");
                assert!(tasks.cancel(k));
                false
            }
        });

        Ok(alive)
    }

    #[instrument(skip(self, tasks))]
    async fn handle(
        &mut self,
        tasks: &mut MappedFutures<i64, impl Future<Output = Infallible>>,
        action: Action,
    ) -> Result<Option<(i64, FeedFetcher<'a>)>> {
        trace!("Handling action");

        match action {
            Action::RerunFailing => {
                let n = self.rerun_failing.notify_relaxed(usize::MAX);
                info!("Notified approximately {n} failing tasks");
            }
            Action::Rerun(id) => {
                let event =
                    self.active_feeds.get(&id).ok_or_eyre("Got rerun command for absent feed")?;

                let n = event.notify_relaxed(1);
                info!("Notified {n} feeds to rerun");
            }
            Action::FeedChanged(id) => {
                let db = self.db.lock().await;
                let feed: Feed = Database::get(db, id).await?;

                if feed.disabled && self.active_feeds.remove(&feed.id()).is_some() {
                    info!("Cancelling task for disabled feed: {feed:?}");
                    assert!(tasks.cancel(&feed.id()));
                } else if !feed.disabled && !self.active_feeds.contains_key(&feed.id()) {
                    info!("Cancelling task for newly enabled feed: {feed:?}");
                    return Ok(Some((feed.id(), self.build_fetcher(feed))));
                }
            }
        }

        Ok(None)
    }

    #[instrument(skip(self))]
    fn build_fetcher(&mut self, feed: Feed) -> FeedFetcher<'a> {
        let host_data = self.insert_host(&feed);
        let rerun = Rc::new(Event::new());

        debug!("Starting task");
        FeedFetcher {
            feed,
            db: self.db,
            host_data,
            failing_timeout: None,
            next_fetch: Instant::now(),
            rerun,
            rerun_failing: self.rerun_failing,
        }
    }

    fn insert_host(&mut self, feed: &Feed) -> &'static HostData {
        let mut kind = HostKind::Http;

        let host = if feed.url.starts_with('!') {
            kind = HostKind::Executable;

            let split: Vec<_> = feed.url.splitn(3, ' ').collect();
            if split[0] == "!rss-scrapers" && split.len() > 2 {
                // As a special case for https://github.com/awused/rss-scrapers
                split[0].to_string() + " " + split[1]
            } else {
                split[0].to_string()
            }
        } else if let Ok(url) = Url::parse(&feed.url) {
            if let Some(host) = url.host_str() {
                host.trim_start_matches("www.").to_string()
            } else {
                error!("Got unparseable Feed URL for {feed:?}");
                "".to_string()
            }
        } else {
            error!("Got unparseable Feed URL for {feed:?}");
            "".to_string()
        };

        match self.host_map.entry(host) {
            Entry::Occupied(o) => o.into_mut(),
            Entry::Vacant(v) => {
                let data = Box::leak(HostData { kind, lock: Mutex::new(()) }.into());
                v.insert(data)
            }
        }
    }
}


pub async fn run(db: &Mutex<Database>, receiver: UnboundedReceiver<Action>) {
    Manager {
        receiver,
        active_feeds: HashMap::new(),
        host_map: HashMap::new(),
        poll_deadline: Instant::now(),
        db,
        rerun_failing: &Event::new(),
    }
    .run()
    .await
}
