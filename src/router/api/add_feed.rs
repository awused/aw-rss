use std::io::Cursor;
use std::time::{Duration, Instant};

use axum::extract::State;
use axum::Json;
use color_eyre::eyre::{bail, Context};
use color_eyre::Result;
use once_cell::sync::Lazy;
use reqwest::header::{HeaderMap, HeaderValue};
use reqwest::{Client, StatusCode, Url};
use rss::Channel;
use scraper::{Html, Selector};
use serde::{Deserialize, Serialize};
use url::Host;

use crate::com::feed::ParsedUpdate;
use crate::com::item::ParsedInsert;
use crate::com::{extract_atom_url, Feed};
use crate::router::{AppError, AppResult, AppState, RouterState};


#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct Request {
    url: String,
    #[serde(default, rename = "title")]
    user_title: String,

    #[serde(default)]
    force: bool,
}

#[derive(Serialize, Debug)]
#[serde(rename_all = "camelCase")]
#[serde(tag = "status")]
pub enum Response {
    Success { feed: Feed },
    // It'd be cool to actually implement this
    //Candidates { candidates: Vec<String> },
    Invalid,
}

// pub(super) async fn handle(
//     State(state): AppState,
//     Json(req): Json<Request>,
// ) -> AppResult<Json<ItemsResponse>> {
pub(super) async fn handle(
    State(state): AppState,
    Json(req): Json<Request>,
) -> AppResult<Json<Response>> {
    add(state, req).await?;

    Ok(Json(Response::Invalid))
}

static CLIENT: Lazy<Client> = Lazy::new(|| {
    let mut headers = HeaderMap::new();
    // Workaround for dolphinemu.org, but doesn't seem to break any other feeds.
    headers.insert("Cache-Control", HeaderValue::from_static("no-cache"));

    // Pretend to be wget. Some sites don't like an empty user agent.
    // Reddit in particular will _always_ say to retry in a few seconds,
    // even if you wait hours.
    headers.insert("User-Agent", HeaderValue::from_static("Wget/1.19.5 (freebsd11.1)"));


    Client::builder()
        .default_headers(headers)
        .timeout(Duration::from_secs(30))
        .brotli(true)
        .gzip(true)
        .deflate(true)
        .build()
        .unwrap()
});

#[rustfmt::skip]
static SELECTOR: Lazy<Selector> = Lazy::new(|| {
    Selector::parse("\
        body > link[type='application/rss+xml'],\
        body > link[type='application/atom+xml'],\
        head > link[type='application/rss+xml'],\
        head > link[type='application/atom+xml']",
    )
    .unwrap()
});

#[instrument(skip(state))]
pub(super) async fn add(state: RouterState, req: Request) -> Result<Response> {
    let url =
        Url::parse(&req.url).wrap_err(AppError::Status(StatusCode::BAD_REQUEST, "Invalid URL"))?;

    if url.scheme() != "http" && url.scheme() != "https" {
        bail!(AppError::Status(StatusCode::BAD_REQUEST, "URL scheme must be http or https"));
    }

    let url = unconditional_url_rewrites(url);

    info!("Attempting to load feed at {url}");

    let body = CLIENT.get(url).send().await?.text().await?;
    let parsed = match parse_feed(&body, None) {
        Ok(parsed) => parsed,
        Err(e) => {
            info!("Failed to parse feed, attempting to parse as HTML");
            warn!("Errors were: {e}");

            // Html is !Send
            let next_link = {
                let doc = Html::parse_document(&body);

                let mut links = doc.select(&SELECTOR).filter_map(|link| link.attr("href"));

                let Some(first) = links.next() else {
                    bail!(AppError::Status(
                        StatusCode::BAD_REQUEST,
                        "Could not parse feed or find feed in HTML"
                    ));
                };

                info!("Found URL ({first})");
                if let Some(second) = links.next() {
                    warn!("Found second URL, aborting ({second})");
                    bail!(AppError::Status(
                        StatusCode::BAD_REQUEST,
                        "Found multiple feeds in HTML"
                    ));
                }
                first.to_string()
            };

            let url = Url::parse(&next_link)
                .wrap_err(AppError::Status(StatusCode::BAD_REQUEST, "Invalid URL"))?;

            if url.scheme() != "http" && url.scheme() != "https" {
                bail!(AppError::Status(
                    StatusCode::BAD_REQUEST,
                    "URL scheme must be http or https"
                ));
            }

            info!("Attempting to load feed at {next_link}");
            let body = CLIENT.get(url).send().await?.text().await?;
            parse_feed(&body, None)?
        }
    };

    info!("oh hey");
    // let text =
    bail!("{parsed:?}")
}


#[instrument(skip(body))]
fn parse_feed(body: &str, feed_id: Option<i64>) -> Result<(ParsedUpdate, Vec<ParsedInsert>)> {
    let start = Instant::now();
    let rss_feed = Channel::read_from(Cursor::new(&body));
    println!("rss parsing {:?}", start.elapsed());

    if let Ok(feed) = rss_feed {
        debug!("Parsed RSS feed");
        let update = ParsedUpdate { title: feed.title, link: Some(feed.link) };

        let Some(feed_id) = feed_id else {
            return Ok((update, Vec::new()));
        };

        let items = feed.items.into_iter().map(|item| (feed_id, item).into()).collect();
        return Ok((update, items));
    }

    let start = Instant::now();
    let atom_feed = atom_syndication::Feed::read_from(Cursor::new(&body));
    println!("atom parsing {:?}", start.elapsed());

    if let Ok(feed) = atom_feed {
        let update = ParsedUpdate {
            title: feed.title.value,
            link: extract_atom_url(feed.links),
        };

        let Some(feed_id) = feed_id else {
            return Ok((update, Vec::new()));
        };

        let items = feed.entries.into_iter().map(|entry| (feed_id, entry).into()).collect();
        return Ok((update, items));
    }

    error!(
        "Failed to decode feed rss: ({}) atom: ({})",
        rss_feed.unwrap_err(),
        atom_feed.unwrap_err()
    );

    bail!(AppError::Status(StatusCode::BAD_REQUEST, "Failed to decode feed"));
}

fn unconditional_url_rewrites(mut url: Url) -> Url {
    // This one would be unnecessary if Candidates were implemented
    if url.path() == "/post"
        && (url.host() == Some(Host::Domain("yande.re"))
            || url.host() == Some(Host::Domain("konachan.com")))
    {
        url.set_path("/post/atom");
    }
    url
}
