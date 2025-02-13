use std::future::IntoFuture;
use std::sync::Arc;
use std::time::Duration;

use api::api_router;
use axum::Router;
use axum::extract::State;
use axum::http::{Uri, header};
use axum::response::IntoResponse;
use axum::routing::get;
use rust_embed::Embed;
use tokio::net::TcpListener;
use tokio::sync::Mutex;
use tokio::sync::mpsc::UnboundedSender;
use tower::ServiceBuilder;
use tower_http::timeout::TimeoutLayer;
use tower_http::trace::TraceLayer;

use crate::closing;
use crate::com::{Action, HttpError};
use crate::database::Database;
use crate::logger::{RequestSpan, ResponseFormat};


mod api;

#[derive(Debug, Clone)]
pub struct RouterState {
    pub db: Arc<Mutex<Database>>,
    pub fetcher_sender: UnboundedSender<Action>,
}


type AppState = State<RouterState>;

type HttpResult<T> = core::result::Result<T, HttpError>;

#[derive(Embed)]
#[folder = "dist/"]
struct Dist;

async fn try_file(uri: Uri) -> impl IntoResponse {
    let path = uri.path().trim_start_matches('/');

    let (file, path) = match Dist::get(path) {
        Some(file) => (file, path),
        None => (Dist::get("index.html").unwrap(), "index.html"),
    };

    let mime = mime_guess::from_path(path).first_or_octet_stream();
    ([(header::CONTENT_TYPE, mime.as_ref())], file.data).into_response()
}

pub async fn serve(listener: TcpListener, state: RouterState) -> color_eyre::Result<()> {
    let service = ServiceBuilder::new()
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(RequestSpan {})
                .on_response(ResponseFormat {}),
        )
        .layer(TimeoutLayer::new(Duration::from_secs(30)))
        // There is no particular need for a concurrency limit, but this entire application is
        // meant for one user.
        .concurrency_limit(8);

    let app = Router::new()
        .nest("/api", api_router())
        .route("/{*file}", get(try_file))
        .route("/", get(try_file))
        .layer(
            service, // .layer(TimeoutLayer::new(Duration::from_secs(1)))
        )
        .with_state(state);

    axum::serve(listener, app)
        .with_graceful_shutdown(closing::closed_fut())
        .into_future()
        .await?;
    Ok(())
}
