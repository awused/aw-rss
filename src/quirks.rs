use crate::com::Feed;

// Quirks not ported over:
// - fictionpress.net/fanfiction.net republishing items
// - mangadex host delays (TODO)

pub fn site_url(new_link: String, feed: &Feed) -> String {
    if new_link.starts_with("https://konachan.com/") {
        feed.url.replacen("/piclens", "", 1).replacen("/atom", "", 1)
    } else if new_link.starts_with("https://www.royalroad.com/fiction") {
        new_link.replacen("syndication/", "", 1)
    } else if new_link == "https://www.novelupdates.com/favicon.ico" {
        "https://www.novelupdates.com/reading-list/".to_string()
    } else {
        new_link
    }
}
