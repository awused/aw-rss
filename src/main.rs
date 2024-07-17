// Single threaded, so lock drop tightening isn't a concern when it would make the code uglier.
#![allow(clippy::significant_drop_tightening)]

use std::sync::Arc;
use std::thread::{self, JoinHandle};
use std::time::Duration;

use config::CONFIG;
use database::Database;
use fetcher::run_fetcher;
use router::{route, RouterState};
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

// fn handle_panic(_e: Box<dyn Any + Send>) {
//     closing::fatal(format!(
//         "Unexpected panic in thread {}",
//         thread::current().name().unwrap_or("unnamed")
//     ));
// }

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


// #[get("/")]
// async fn aw_handler() -> impl Responder {
//     let rc = Rc::new(());
//     sleep(Duration::from_millis(1)).await;
//     let db = DB.with(|c| c.get().unwrap().upgrade().unwrap());
//     info!("handling");
//     format!("{}", Rc::strong_count(&rc))
// }
//

// async fn handle_timeout_error(err: BoxError) -> (StatusCode, String) {
//     if err.is::<timeout::error::Elapsed>() {
//         println!("Handle error");
//         (StatusCode::REQUEST_TIMEOUT, "Request took too long".to_string())
//     } else {
//         (StatusCode::INTERNAL_SERVER_ERROR, "".to_string())
//     }
// }
//

#[tokio::main(flavor = "current_thread")]
async fn main() -> color_eyre::Result<()> {
    config::init();
    color_eyre::install()?;
    logger::init_logging();

    closing::init();

    // The database needs to be Send for axum, since it doesn't properly support single-threaded
    // runtimes.
    let db = Arc::new(Mutex::new(Database::new().await?));

    let listener = TcpListener::bind((&*CONFIG.host, CONFIG.port)).await?;

    #[allow(clippy::redundant_pub_crate)]
    {
        let (fetcher_sender, fetcher_receiver) = unbounded_channel();
        let router = route(listener, RouterState { db: db.clone(), fetcher_sender });
        let fetcher = run_fetcher(&db, fetcher_receiver);
        pin!(fetcher, router);

        tokio::select! {
                r = &mut router => {
                    if closing::close() {
                        error!("Axum unexpectedly stopped serving: {r:?}");
                        // It is probably unnecessary to wait for the fetcher
                        timeout(Duration::from_secs(60), &mut fetcher).await??;
                    }
                },
                r = &mut fetcher => {
                    if closing::close() {
                        error!("Fetcher unexpectedly exited: {r:?}");
                        timeout(Duration::from_secs(60), &mut router).await??;
                    }
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

    Ok(())
}
