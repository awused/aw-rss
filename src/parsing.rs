use std::io::Cursor;
use std::time::Duration;

use atom_syndication::Link;
use chrono::Utc;
use color_eyre::eyre::eyre;
use color_eyre::{Result, Section};
use rss::Channel;
use sha2::{Digest, Sha256};

use crate::com::feed::ParsedUpdate;
use crate::com::item::ParsedInsert;
use crate::com::{Feed, RssStruct};

#[derive(Debug)]
pub struct ParsedFeed {
    pub update: ParsedUpdate,
    pub items: Vec<ParsedInsert>,
    pub ttl: Option<Duration>,
    pub extension_etag: Option<String>,
}

pub fn parse_feed(feed: &Feed, body: &str) -> Result<ParsedFeed> {
    parse(body, Some(feed))
}

pub fn check_valid_feed(body: &str) -> Result<()> {
    parse(body, None).map(|_| ())
}

#[instrument(skip_all)]
fn parse(body: &str, feed: Option<&Feed>) -> Result<ParsedFeed> {
    let rss_feed = Channel::read_from(Cursor::new(&body));

    if let Ok(mut parsed) = rss_feed {
        trace!("Parsed RSS feed");
        let update = ParsedUpdate {
            title: parsed.title,
            link: Some(parsed.link),
        };

        let mut out = ParsedFeed {
            update,
            items: Vec::new(),
            ttl: None,
            extension_etag: parsed
                .extensions
                .remove("aw-rss")
                .and_then(|mut m| m.remove("etag"))
                .and_then(|mut exts| exts.pop())
                .and_then(|e| e.value),
        };

        let Some(feed) = feed else {
            return Ok(out);
        };

        // Reverse to ensure sorting is consistent when timestamps are equal
        out.items = parsed.items.into_iter().rev().map(|item| (feed, item).into()).collect();
        out.ttl = parsed
            .ttl
            .and_then(|t| t.parse::<u64>().ok())
            .map(|m| Duration::from_secs(60 * m));
        return Ok(out);
    }

    let atom_feed = atom_syndication::Feed::read_from(Cursor::new(&body));

    if let Ok(mut parsed) = atom_feed {
        trace!("Parsed atom feed");
        let update = ParsedUpdate {
            title: parsed.title.value,
            link: extract_atom_url(parsed.links),
        };

        let mut out = ParsedFeed {
            update,
            items: Vec::new(),
            ttl: None,
            extension_etag: parsed
                .extensions
                .remove("aw-rss")
                .and_then(|mut m| m.remove("etag"))
                .and_then(|mut exts| exts.pop())
                .and_then(|e| e.value),
        };

        let Some(feed) = feed else {
            return Ok(out);
        };

        // Reverse to ensure sorting is consistent when timestamps are equal
        out.items = parsed.entries.into_iter().rev().map(|entry| (feed, entry).into()).collect();
        return Ok(out);
    }

    debug!("Failed to decode feed as rss or atom");
    Err(eyre!("Failed to decode feed")
        .section(rss_feed.unwrap_err())
        .section(atom_feed.unwrap_err()))
}

// alternate > self > nothing > whatever else
pub fn extract_atom_url(mut links: Vec<Link>) -> Option<String> {
    links
        .iter()
        .position(|a| a.rel == "alternate")
        .or_else(|| links.iter().position(|a| a.rel == "self"))
        .or_else(|| links.iter().position(|a| a.rel.is_empty()))
        .or_else(|| (!links.is_empty()).then_some(0))
        .map(|i| links.swap_remove(i).href)
}

// Using a hash as a fallback key over the URL has been necessary in the past but might not
// be worth the complexity. Falling back to the URL should be fine in the vast majority of
// cases without needing the hash.

impl From<(&Feed, rss::Item)> for ParsedInsert {
    fn from((feed, item): (&Feed, rss::Item)) -> Self {
        let title = item.title.clone().unwrap_or_default();
        let url = item.link.unwrap_or_default();
        let key = item
            .guid
            .and_then(|g| (!g.value.is_empty()).then_some(g.value))
            .or_else(|| item.title.and_then(|t| item.pub_date.as_ref().map(|p| t + p)))
            .unwrap_or_else(|| {
                let mut hasher = Sha256::new();
                hasher.update(item.description.as_ref().unwrap_or(&url));
                format!("{:X}", hasher.finalize())
            });

        let timestamp = item
            .pub_date
            .as_ref()
            .and_then(|d| dateparser::parse(d).ok())
            .unwrap_or_else(|| {
                warn!("Got item with no timestamp: {url:?}");
                Utc::now()
            });

        Self {
            feed_id: feed.id(),
            key,
            title,
            url,
            timestamp: timestamp.into(),
        }
    }
}

impl From<(&Feed, atom_syndication::Entry)> for ParsedInsert {
    fn from((feed, entry): (&Feed, atom_syndication::Entry)) -> Self {
        let title = entry.title.to_string();
        let url = extract_atom_url(entry.links).unwrap_or_default();
        if url.is_empty() {
            warn!("Got atom item with no url");
        }

        let key = if !entry.id.is_empty() {
            entry.id
        } else {
            warn!("Got atom item with no ID, this shouldn't happen {url:?}");
            entry
                .published
                .as_ref()
                .and_then(|p| {
                    if !entry.title.is_empty() {
                        Some(entry.title.value + &p.to_rfc3339())
                    } else {
                        None
                    }
                })
                .unwrap_or_else(|| {
                    error!("Got atom item with no ID, no title, and no published date: {url}");
                    let mut hasher = Sha256::new();
                    hasher.update(
                        entry.content.as_ref().and_then(|c| c.value.as_ref()).unwrap_or(&url),
                    );
                    format!("{:X}", hasher.finalize())
                })
        };

        let timestamp = entry.published.unwrap_or(entry.updated).to_utc();

        Self {
            feed_id: feed.id(),
            key,
            title,
            url,
            timestamp: timestamp.into(),
        }
    }
}
