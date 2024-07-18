use std::fs::OpenOptions;

use axum::http::HeaderValue;
use chrono::format::{Fixed, Item};
use chrono::{Local, Timelike};
use color_eyre::eyre::bail;
use tower_http::trace::{MakeSpan, OnResponse};
use tracing::{event, Level};
use tracing_error::ErrorLayer;
use tracing_subscriber::fmt::format::Writer;
use tracing_subscriber::fmt::time::FormatTime;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::util::SubscriberInitExt;
use tracing_subscriber::EnvFilter;

use crate::config::CONFIG;

static DATE_FORMAT: &[Item; 1] = &[Item::Fixed(Fixed::RFC3339)];

struct TimeFormatter {}

impl FormatTime for TimeFormatter {
    fn format_time(&self, w: &mut Writer<'_>) -> std::fmt::Result {
        // OffsetDateTime::now_utc().to_offset(*LOCALE).form.format_into(w, &Rfc3339).map_err(|_|
        // std::fmt::Error).map(|_|) Ok(())
        // Force 0ns for consistent seconds formatting
        let now = Local::now().with_nanosecond(0).unwrap().format_with_items(DATE_FORMAT.iter());
        write!(w, "{now}")
    }
}

#[derive(Debug, Clone, Copy)]
pub struct RequestSpan {}

impl<B> MakeSpan<B> for RequestSpan {
    fn make_span(&mut self, request: &axum::http::Request<B>) -> tracing::Span {
        tracing::span!(Level::ERROR, "", "{}:{}", request.method(), request.uri(),)
    }
}

#[derive(Debug, Clone, Copy)]
pub struct ResponseFormat {}

impl<B> OnResponse<B> for ResponseFormat {
    fn on_response(
        self,
        response: &axum::http::Response<B>,
        latency: std::time::Duration,
        _span: &tracing::Span,
    ) {
        let status = response.status();
        let status = status.canonical_reason().unwrap_or_else(|| status.as_str());

        if let Some(size) = response.headers().get("content-length").and_then(|b: &HeaderValue| {
            let bytes: usize = b.to_str().ok()?.parse().ok()?;
            Some(humansize::SizeFormatter::new(bytes, humansize::WINDOWS))
        }) {
            event!(Level::INFO, status, "finished {:?} {}", latency, size);
        } else {
            event!(Level::INFO, status, "finished {:?} no body", latency,);
        }
    }
}

pub fn init_logging() -> color_eyre::Result<()> {
    let filter_layer = EnvFilter::builder()
        .with_default_directive(CONFIG.log_level.parse().unwrap())
        .from_env_lossy()
        // These are unbelievably spammy
        .add_directive("html5ever=warn".parse().unwrap())
        .add_directive("selectors=warn".parse().unwrap());

    let fmt_layer = tracing_subscriber::fmt::layer().with_timer(TimeFormatter {});
    let registry = tracing_subscriber::registry().with(filter_layer);

    if let Some(file) = CONFIG.log_file.as_ref() {
        if !file.is_absolute() {
            bail!("Configured log_file {file:?} is not an absolute path");
        }

        let file = OpenOptions::new().truncate(true).create(true).write(true).open(file)?;
        registry.with(fmt_layer.with_writer(file)).with(ErrorLayer::default()).init();
    } else {
        registry.with(fmt_layer).with(ErrorLayer::default()).init();
    }

    Ok(())
}
