use std::cell::RefCell;
use std::collections::HashMap;
use std::sync::Arc;

use color_eyre::Result;
use tokio::sync::mpsc::UnboundedReceiver;
use tokio::sync::Mutex;

use crate::closing;
use crate::com::FetcherAction;
use crate::database::Database;


struct Fetcher {
    db: Arc<Mutex<Database>>,
    receiver: UnboundedReceiver<()>,
    host_map: RefCell<HashMap<String, ()>>,
}

pub async fn run_fetcher(
    db: &Mutex<Database>,
    mut receiver: UnboundedReceiver<FetcherAction>,
) -> Result<()> {
    while let Some(next) = receiver.recv().await {
        error!("Got {next:?}");
    }
    closing::closed_fut().await;

    Ok(())
}
