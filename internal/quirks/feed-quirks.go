package quirks

import (
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
)

type feed interface {
	Title() string
	URL() string
}

// "MangaDex RSS" is a terrible title for every per-series feed
const mangadexItemRegexp = `^(.+) - [^-]+$`

var mdire = regexp.MustCompile(mangadexItemRegexp)

const mangadexSeriesRegexp = `^https://mangadex\.org/rss/[0-9a-z]+/manga_id/([0-9]+)`

var mdsre = regexp.MustCompile(mangadexSeriesRegexp)

const konachanRegexp = `^https?://konachan\.com/post/piclens\?tags?=(.*)$`

var konare = regexp.MustCompile(konachanRegexp)

// GetFeedTitle overrides the feed title, if necessary
func GetFeedTitle(f feed, gfe *gofeed.Feed) string {
	if gfe.Title == "MangaDex RSS" {
		title := gfe.Title
		if f.Title() != "" && f.Title() != title {
			title = f.Title()
		}

		if gfe.Items == nil || len(gfe.Items) == 0 || !mdsre.MatchString(f.URL()) {
			return title
		}

		groups := mdire.FindStringSubmatch(gfe.Items[0].Title)
		if groups == nil {
			return title
		}

		return groups[1]
	}

	return gfe.Title
}

// GetFeedLink overrides the feed link, if necessary
func GetFeedLink(f feed, gfe *gofeed.Feed) string {
	if gfe.Link == "https://mangadex.org/" {
		groups := mdsre.FindStringSubmatch(f.URL())
		if groups != nil {
			return "https://mangadex.org/title/" + groups[1]
		}
	} else if gfe.Link == "http://konachan.com/" {
		groups := konare.FindStringSubmatch(f.URL())
		if groups != nil {
			return "https://konachan.com/post?tags=" + groups[1]
		}
	}

	if gfe.Link == "https://www.novelupdates.com/favicon.ico" {
		return "https://www.novelupdates.com/reading-list/"
	}

	if strings.HasPrefix(gfe.Link, "https://www.royalroad.com/fiction") {
		return strings.Replace(gfe.Link, "syndication/", "", 1)
	}

	return gfe.Link
}
