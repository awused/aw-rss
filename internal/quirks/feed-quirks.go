package quirks

import (
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
)

// "MangaDex RSS" is a terrible title for every per-series feed
const mangadexItemRegexp = `^(.+) - [^-]+$`

var mdire = regexp.MustCompile(mangadexItemRegexp)

const mangadexSeriesRegexp = `^https://mangadex\.org/rss/[0-9a-z]+/manga_id/([0-9]+)`

var mdsre = regexp.MustCompile(mangadexSeriesRegexp)

type feed interface {
	URL() string
}

// GetFeedTitle overrides the feed title, if necessary
func GetFeedTitle(f feed, gfe *gofeed.Feed) string {
	if gfe.Title != "MangaDex RSS" {
		return gfe.Title
	}

	if gfe.Items == nil || len(gfe.Items) == 0 || !mdsre.MatchString(f.URL()) {
		return gfe.Title
	}

	groups := mdire.FindStringSubmatch(gfe.Items[0].Title)
	if groups == nil {
		return gfe.Title
	}

	return groups[1]
}

// GetFeedLink overrides the feed link, if necessary
func GetFeedLink(f feed, gfe *gofeed.Feed) string {
	if gfe.Link == "https://mangadex.org/" {
		groups := mdsre.FindStringSubmatch(f.URL())
		if groups != nil {
			return "https://mangadex.org/title/" + groups[1]
		}
	} else if strings.HasPrefix(gfe.Link, "https://www.royalroad.com/fiction") {
		return strings.Replace(gfe.Link, "syndication/", "", 1)
	}

	return gfe.Link
}
