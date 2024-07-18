// Single threaded, so lock drop tightening isn't a concern when it would make the code uglier to
// no actual benefit.
// Plus it's often completely wrong
#![allow(clippy::significant_drop_tightening)]

use std::sync::Arc;
use std::thread::{self, JoinHandle};
use std::time::Duration;

use config::CONFIG;
use database::Database;
use router::RouterState;
use tokio::net::TcpListener;
use tokio::pin;
use tokio::sync::mpsc::unbounded_channel;
use tokio::sync::Mutex;
use tokio::time::{sleep, timeout};

#[macro_use]
extern crate tracing;

mod closing;
mod com;
mod config;
mod database;
mod fetcher;
mod logger;
mod parsing;
mod quirks;
mod router;

fn spawn_thread<F, T>(name: &str, f: F) -> JoinHandle<T>
where
    F: FnOnce() -> T + Send + 'static,
    T: Send + 'static,
{
    thread::Builder::new()
        .name(name.to_string())
        .spawn(f)
        .unwrap_or_else(|e| panic!("Error spawning thread {name}: {e}"))
}

#[tokio::main(flavor = "current_thread")]
async fn main() -> color_eyre::Result<()> {
    config::init();
    color_eyre::install()?;
    logger::init_logging()?;

    closing::init();

    // The database needs to be Send for axum, since it doesn't properly support single-threaded
    // runtimes.
    let db = Arc::new(Mutex::new(Database::new().await?));

    let listener = TcpListener::bind((&*CONFIG.host, CONFIG.port)).await?;

    #[allow(clippy::redundant_pub_crate)]
    {
        let (fetcher_sender, fetcher_receiver) = unbounded_channel();
        let router = router::serve(listener, RouterState { db: db.clone(), fetcher_sender });
        let fetcher = fetcher::run(&db, fetcher_receiver);
        pin!(fetcher, router);

        tokio::select! {
            r = &mut router => {
                if closing::close() {
                    error!("Axum unexpectedly stopped serving: {r:?}");
                }
                // It is probably unnecessary to wait for the fetcher
                timeout(Duration::from_secs(60), fetcher).await?;
            },
            r = &mut fetcher => {
                if closing::close() {
                    error!("Fetcher unexpectedly exited: {r:?}");
                }
                timeout(Duration::from_secs(60), router).await??;
            }
            _ = async {
                closing::closed_fut().await;
                sleep(Duration::from_secs(60)).await;
            } => {
                error!("Application closed but axum/fetcher did not exit in a timely manner")
            }
        }
    }

    info!("Attempting to gracefully close the database");
    db.try_lock()?.close().await?;

    info!("Exited cleanly");
    Ok(())
}
