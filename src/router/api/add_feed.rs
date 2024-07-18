use axum::extract::State;
use axum::Json;
use color_eyre::eyre::{bail, Context};
use color_eyre::{Result, Section};
use once_cell::sync::Lazy;
use reqwest::Url;
use scraper::{Html, Selector};
use serde::{Deserialize, Serialize};
use url::Host;

use crate::com::feed::UserInsert;
use crate::com::{Action, Feed, RssStruct, CLIENT};
use crate::database::Database;
use crate::parsing::check_valid_feed;
use crate::router::{AppState, HttpError, HttpResult, RouterState};


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
}

pub(super) async fn handle(
    State(state): AppState,
    Json(req): Json<Request>,
) -> HttpResult<Json<Response>> {
    Ok(Json(add(state, req).await?))
}


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

pub(super) async fn add(state: RouterState, req: Request) -> Result<Response> {
    let url = if req.force {
        // It still needs to be a valid http/http URL
        parse_url(&req.url)?
    } else {
        get_valid_feed_url(&req.url).await?
    };

    let insert = UserInsert {
        url: url.to_string(),
        user_title: req.user_title,
    };

    let db = state.db.lock().await;
    let feed = Database::single_insert(db, insert).await?;
    state.fetcher_sender.send(Action::FeedChanged(feed.id()))?;
    Ok(Response::Success { feed })
}

#[instrument]
async fn get_valid_feed_url(url: &str) -> Result<Url> {
    let url = parse_url(url)?;

    info!("Attempting to load feed");

    let body = CLIENT.get(url.clone()).send().await?.text().await?;
    let Err(e) = check_valid_feed(&body) else {
        return Ok(url);
    };

    info!("Failed to parse feed, attempting to parse as HTML");
    // This is logged again if the if it fails again, so it's low priority
    trace!("Errors were: {e}");

    let mut e = Some(e);
    let mut wrapped = move || e.take().unwrap();
    // Html is !Send
    let first_link = {
        let doc = Html::parse_document(&body);

        let mut links = doc.select(&SELECTOR).filter_map(|link| link.attr("href"));

        let first = links
            .next()
            .ok_or_else(|| HttpError::bad("Could not parse feed or find feed in HTML"))
            .with_section(&mut wrapped)?;

        info!("Found URL ({first})");
        if let Some(second) = links.next() {
            warn!("Found second URL, aborting ({second})");
            bail!(HttpError::bad("Found multiple feeds in HTML, TODO Candidates"));
        }
        first.to_string()
    };

    let url = parse_url(&first_link).with_section(&mut wrapped)?;

    info!("Attempting to load feed at {first_link}");
    let body = CLIENT.get(url.clone()).send().await?.text().await?;
    check_valid_feed(&body).with_section(&mut wrapped)?;

    Ok(url)
}

fn parse_url(url: &str) -> Result<Url> {
    let url = Url::parse(url).wrap_err(HttpError::bad("Invalid URL"))?;

    if url.scheme() != "http" && url.scheme() != "https" {
        bail!(HttpError::bad("URL scheme must be http or https"));
    }

    Ok(unconditional_url_rewrites(url))
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
