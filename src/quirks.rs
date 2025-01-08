use chrono::{DateTime, Utc};
use url::Url;

use crate::com::Feed;

// Quirks not ported over:
// - fictionpress.net/fanfiction.net republishing items

pub fn site_url(new_link: String, feed: &Feed) -> String {
    if new_link.starts_with("https://konachan.com/") {
        feed.url.replacen("/piclens", "", 1).replacen("/atom", "", 1)
    } else if new_link.starts_with("https://www.royalroad.com/fiction") {
        new_link.replacen("syndication/", "", 1)
    } else if new_link == "https://www.novelupdates.com/favicon.ico" {
        "https://www.novelupdates.com/reading-list/".to_string()
    } else if new_link == "https://forums.spacebattles.com/"
        || new_link == "https://forums.sufficientvelocity.com/"
    {
        feed.url.split("/threadmarks.rss").next().unwrap().to_string()
    } else {
        new_link
    }
}

// Handle guid changing for the same item, this is not a good solution.
pub fn item_key(item_key: String, feed: &Feed, timestamp: DateTime<Utc>) -> String {
    if item_key.starts_with('/')
        && (feed.site_url.starts_with("https://forums.spacebattles.com/")
            || feed.site_url.starts_with("https://forums.sufficientvelocity.com/"))
    {
        // SB/SV use urls as guids, so delegating to item_url is fine as the code is right now.
        return item_url(item_key, feed);
    }

    if feed.site_url.starts_with("https://secure.runescape.com") {
        // Caught them reusing URLs and GUIDs for different items
        return item_key + &timestamp.to_rfc3339();
    }

    item_key
}

// Some feeds on spacebattles produce invalid URLs, but some of this code can be good for invalid
// URLs in general.
pub fn item_url(mut item_url: String, feed: &Feed) -> String {
    if !item_url.starts_with("/") {
        return item_url;
    }

    // Invalid item with a relative URL, grab the host from the site_url
    let Ok(mut url) = Url::parse(&feed.site_url) else {
        return item_url;
    };

    if url.scheme() != "https" && url.scheme() != "http" {
        return item_url;
    }

    let sbsv = url
        .host_str()
        .is_some_and(|h| h == "forums.spacebattles.com" || h == "forums.sufficientvelocity.com");

    if sbsv {
        // /page-1257#post-103747651 -> /post-103747651
        if let Some((a, b)) = item_url.split_once("/page-") {
            if let Some((_, d)) = b.split_once("#post-") {
                item_url = a.to_string() + "/post-" + d;
            }
        }
    } else {
        warn!("Got invalid URL {item_url} for non-SB/SV feed");
    }

    url.set_path("");
    url.set_query(None);
    url.set_fragment(None);


    let Ok(mut url) = url.join(&item_url) else {
        return item_url;
    };

    if sbsv {
        // Saw some /post-103747651?page=1234 urls, for now kill them too
        url.set_query(None)
    }

    url.to_string()
}
