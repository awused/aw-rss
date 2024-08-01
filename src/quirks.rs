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

// Some feeds on spacebattles produce invalid URLs, but some of this code can be good for invalid
// URLs in general.
pub fn item_url(item_url: String, feed: &Feed) -> String {
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

    url.set_path("");
    url.set_query(None);
    url.set_fragment(None);

    let Ok(mut url) = url.join(&item_url) else {
        return item_url;
    };

    if url
        .host_str()
        .is_some_and(|h| h == "forums.spacebattles.com" || h == "forums.sufficientvelocity.com")
    {
        // SB/SV have started producing complete and utter garbage
        // The fragment contains query params which is just broken.
        url.set_query(None);
        if let Some(frag) = url.fragment() {
            let clean_fragment = frag.split('?').next().unwrap().to_string();

            url.set_fragment(Some(&clean_fragment));
        }
    } else {
        warn!("Got invalid URL {item_url} for non-SB/SV feed");
    }

    url.to_string()
}
