use crate::com::Feed;


pub fn site_url(new_link: String, feed: &Feed) -> String {
    if new_link.starts_with("https://konachan.com/") {
        feed.url.replacen("/piclens", "", 1).replacen("/atom", "", 1)
    } else if new_link.starts_with("https://www.royalroad.com/fiction") {
        new_link.replacen("syndication/", "", 1)
    } else {
        new_link
    }
}
