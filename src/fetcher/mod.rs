use std::boxed::Box;
use std::collections::HashMap;
use std::collections::hash_map::Entry;
use std::convert::Infallible;
use std::future::Future;
use std::rc::Rc;
use std::time::Duration;

use color_eyre::Result;
use color_eyre::eyre::OptionExt;
use event_listener::Event;
use fetch::{Headers, Status};
use futures_util::StreamExt;
use mapped_futures::MappedFutures;
use tokio::select;
use tokio::sync::mpsc::UnboundedReceiver;
use tokio::sync::{Mutex, MutexGuard};
use tokio::time::{Instant, sleep_until};
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

#[derive(Debug, Default)]
struct HostData {
    // Increases by 1 each time a feed starts failing
    // Drops by 1 every time a feed succeeds
    // Used to increase starting timeouts when many feeds are failing for one host.
    failing_feeds: u64,
}

#[derive(Debug)]
struct Host {
    kind: HostKind,
    lock: Mutex<HostData>,
}

struct Manager<'a> {
    receiver: UnboundedReceiver<Action>,
    host_map: HashMap<String, &'static Host>,
    active_feeds: HashMap<i64, Rc<Event>>,
    poll_deadline: Instant,

    // Can be borrowed by tasks
    db: &'a Mutex<Database>,
    rerun_failing: &'a Event,
}


struct FeedFetcher<'a> {
    feed: Feed,
    db: &'a Mutex<Database>,
    host: &'static Host,
    status: Status,
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

        let db = self.db.lock().await;
        let initial = self.poll_db(db, &mut tasks).await.expect("Failed to load initial feeds");

        for (id, feed) in initial {
            let fetcher = self.build_fetcher(feed);
            assert!(self.active_feeds.insert(id, fetcher.rerun.clone()).is_none());
            assert!(tasks.insert(id, fetcher.run()));
        }

        info!("Loaded {} initial feeds", self.active_feeds.len());

        let mut pending_msg = None;

        loop {
            select! {
                biased;
                _ = closing::closed_fut() => break,
                // This is two sections to guarantee that we don't lose a message in a cancelled
                // future.
                msg = self.receiver.recv(), if pending_msg.is_none() => {
                    if msg.is_none() {
                        if closing::close() {
                            error!("Channel closed unexpectedly");
                        }
                        break;
                    };
                    pending_msg = msg;
                }
                guard = self.db.lock(), if pending_msg.is_some() => {
                    match self.handle(guard, &mut tasks, pending_msg.take().unwrap()).await {
                        Ok(None) => {},
                        Ok(Some((id, fetcher))) => {
                            assert!(self.active_feeds.insert(id, fetcher.rerun.clone()).is_none());
                            assert!(tasks.insert(id, fetcher.run()));
                        }
                        Err(e) => error!("{e:?}")
                    }
                }
                guard = async {
                    sleep_until(self.poll_deadline).await;
                    self.db.lock().await
                } => {
                    let new = match self.poll_db(guard, &mut tasks).await {
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
                _ = tasks.next(), if !tasks.is_empty() => {
                    unreachable!()
                }
            }
        }
    }

    #[instrument(skip_all)]
    async fn poll_db(
        &mut self,
        db: MutexGuard<'a, Database>,
        tasks: &mut MappedFutures<i64, impl Future<Output = Infallible>>,
    ) -> Result<HashMap<i64, Feed>> {
        self.poll_deadline += POLL_DURATION;

        // We cannot trust the commit timestamp if wall time rolls back, so do a full scan.
        // With this scan in place, the worst case is that a user might need to refresh their
        // client, but the server will always pick up all Feed changes.
        let feeds = Database::active_feeds(db).await?;

        debug!("Loaded {} active feeds", feeds.len());

        let mut alive: HashMap<_, _> = feeds.into_iter().map(|f| (f.id(), f)).collect();

        self.active_feeds.retain(|k, _v| {
            if alive.remove(k).is_some() {
                true
            } else {
                info!("Cancelling task for disabled feed: {k}");
                assert!(tasks.cancel(k));
                false
            }
        });

        Ok(alive)
    }

    #[instrument(skip(self, db, tasks))]
    async fn handle(
        &mut self,
        db: MutexGuard<'a, Database>,
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
                let feed: Feed = Database::get(db, id).await?;

                if feed.disabled && self.active_feeds.remove(&feed.id()).is_some() {
                    info!("Cancelling task for disabled feed: {feed}");
                    assert!(tasks.cancel(&feed.id()));
                } else if !feed.disabled && !self.active_feeds.contains_key(&feed.id()) {
                    info!("Creating task for newly enabled feed: {feed}");
                    return Ok(Some((feed.id(), self.build_fetcher(feed))));
                }
            }
        }

        Ok(None)
    }

    #[instrument(skip(self), fields(%feed))]
    fn build_fetcher(&mut self, feed: Feed) -> FeedFetcher<'a> {
        let host = self.insert_host(&feed);
        let rerun = Rc::new(Event::new());

        debug!("Starting task");
        FeedFetcher {
            feed,
            db: self.db,
            host,
            status: Status::Success(Headers::default()),
            next_fetch: Instant::now(),
            rerun,
            rerun_failing: self.rerun_failing,
        }
    }

    fn insert_host(&mut self, feed: &Feed) -> &'static Host {
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
        } else {
            Url::parse(&feed.url)
                .ok()
                .and_then(|url| {
                    url.host_str().map(|host| host.trim_start_matches("www.").to_string())
                })
                .unwrap_or_else(|| {
                    error!("Got unparseable Feed URL");
                    String::new()
                })
        };

        match self.host_map.entry(host) {
            Entry::Occupied(o) => o.into_mut(),
            Entry::Vacant(v) => {
                let data = Box::leak(Host { kind, lock: Mutex::default() }.into());
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
