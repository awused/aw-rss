use axum::http::HeaderValue;
use chrono::format::{Fixed, Item};
use chrono::{Local, Timelike};
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

pub fn init_logging() {
    let filter_layer = EnvFilter::builder()
        .with_default_directive(CONFIG.log_level.parse().unwrap())
        .from_env_lossy()
        // These are unbelievably spammy
        .add_directive("html5ever=warn".parse().unwrap())
        .add_directive("selectors=warn".parse().unwrap());
    let fmt_layer = tracing_subscriber::fmt::layer().with_timer(TimeFormatter {});

    tracing_subscriber::registry()
        .with(filter_layer)
        .with(fmt_layer)
        .with(ErrorLayer::default())
        .init();
    // Lazy::force(&LOCALE);

    // tracing_subscriber::fmt()
    //     // .with_filter()
    //     .with_env_filter(
    //         EnvFilter::builder()
    //             .with_default_directive(LevelFilter::INFO.into())
    //             .from_env_lossy(),
    //     )
    //     .with_timer(TimeFormatter {})
    //     .init();

    // env_logger::Builder::from_default_env().format(|f, record| {
    //     use std::io::Write;
    //     let target = record.target();
    //     let target = target.strip_prefix(PREFIX).unwrap_or(target);
    //     let target = shrink_target(target);
    //     let max_width = max_target_width(target);
    //
    //     let style = f.default_level_style(record.level()).bold();
    //
    //     let now = Local::now().with_nanosecond(0).unwrap().format_with_items(DATE_FORMAT.iter());
    //
    //     let style_render = style.render();
    //     let style_reset = style.render_reset();
    //     let level = record.level();
    //     let args = record.args();
    //
    //     writeln!(f, "{now} {style_render}{level:5}{style_reset} {target:max_width$} > {args}",)
    // });
    // .init();
}

// static MAX_MODULE_WIDTH: AtomicUsize = AtomicUsize::new(0);
// const MAX_WIDTH: usize = 20;
//
// // Strips all but the last two modules.
// fn shrink_target(target: &str) -> &str {
//     if let Some(x) = target.rfind("::") {
//         if let Some(x) = target[0..x].rfind("::") {
//             return &target[x + 2..];
//         }
//     }
//     target
// }
//
// fn max_target_width(target: &str) -> usize {
//     let max_width = MAX_MODULE_WIDTH.load(Ordering::Relaxed);
//     if max_width < target.len() {
//         let newlen = min(target.len(), MAX_WIDTH);
//         MAX_MODULE_WIDTH.store(newlen, Ordering::Relaxed);
//         target.len()
//     } else {
//         max_width
//     }
// }
