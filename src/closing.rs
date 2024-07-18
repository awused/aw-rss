use std::marker::PhantomData;
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::rc::Rc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;

use async_channel::{bounded, Receiver, Sender};
use once_cell::sync::Lazy;

use crate::spawn_thread;

type CloseSender = Mutex<Option<Sender<()>>>;
type CloseReceiver = Receiver<()>;

static CLOSED: Lazy<Arc<AtomicBool>> = Lazy::new(|| Arc::new(AtomicBool::new(false)));
static CLOSER: Lazy<(CloseSender, CloseReceiver)> = Lazy::new(|| {
    let (s, r) = bounded::<()>(1);
    (Mutex::new(Option::Some(s)), r)
});

#[derive(Default)]
pub struct CloseOnDrop {
    _phantom: PhantomData<Rc<CloseOnDrop>>,
}

// TODO -- https://github.com/rust-lang/rust/issues/68318
// impl !Send for CloseOnDrop {}
// impl !Sync for CloseOnDrop {}

impl Drop for CloseOnDrop {
    fn drop(&mut self) {
        if close() {
            // This means something else panicked and at least one thread did not shut down cleanly.
            error!(
                "CloseOnDrop for {} was dropped without closing::close() being called.",
                thread::current().name().unwrap_or("unnamed")
            );
        }
    }
}

/// Resolves when `close()` is called
pub async fn closed_fut() {
    // We only care that it's closed.
    let _ignored = CLOSER.1.recv().await;
}

/// returns false if we were already closed
pub fn close() -> bool {
    if !CLOSED.swap(true, Ordering::Relaxed) {
        let mut o = CLOSER.0.lock().expect("CLOSER lock poisoned");
        if o.is_some() {
            *o = Option::None;
        } else {
            error!("CLOSER unexpectedly closed before CLOSED");
        }
        drop(o);
        true
    } else {
        false
    }
}

pub fn init() {
    Lazy::force(&CLOSER);

    #[cfg(target_family = "unix")]
    spawn_thread("signals", || {
        use std::os::raw::c_int;

        use signal_hook::consts::TERM_SIGNALS;
        use signal_hook::iterator::exfiltrator::SignalOnly;
        use signal_hook::iterator::SignalsInfo;

        let _cod = CloseOnDrop::default();

        if let Err(e) = catch_unwind(AssertUnwindSafe(|| {
            for sig in TERM_SIGNALS {
                // When terminated by a second term signal, exit with exit code 1.
                signal_hook::flag::register_conditional_shutdown(*sig, 1, CLOSED.clone())
                    .expect("Error registering signal handlers.");
            }

            let mut sigs: Vec<c_int> = Vec::new();
            sigs.extend(TERM_SIGNALS);
            let mut it = match SignalsInfo::<SignalOnly>::new(sigs) {
                Ok(i) => i,
                Err(e) => {
                    error!("Error registering signal handlers: {e:?}");
                    close();
                    return;
                }
            };

            if let Some(s) = it.into_iter().next() {
                info!("Received signal {s}, shutting down");
                close();
                it.handle().close();
            }
        })) {
            error!("Signal thread panicked unexpectedly: {e:?}");
            close();
        };
    });

    #[cfg(windows)]
    spawn_thread("signals", || {
        ctrlc::set_handler(|| {
            if closed() {
                // When terminated by a second term signal, exit with exit code 1.
                std::process::exit(1);
            }

            info!("Received closing signal, shutting down");
            close();
        })
        .expect("Error registering signal handlers.");
    });
}
