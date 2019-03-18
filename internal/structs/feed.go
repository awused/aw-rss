package structs

// TODO -- Split this into "Feed" for data the user has entered and "FeedData"
// for automatically created data. Not happy with how this grew. They can both
// be stored in the same table.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/mmcdole/gofeed"
)

// Feed represents a single RSS feed
type Feed struct {
	id        int64
	url       string
	disabled  bool
	title     string
	siteURL   string
	userTitle string

	// The timestamp of the most recent failed fetch
	// helps users see if a breakage is transient
	failingSince    *time.Time
	commitTimestamp time.Time
	createTimestamp time.Time
}

// MarshalJSON is used by the JSON marshaller
func (f *Feed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64      `json:"id"`
		URL             string     `json:"url"`
		Disabled        bool       `json:"disabled"`
		Title           string     `json:"title"`
		SiteURL         string     `json:"siteUrl"`
		UserTitle       string     `json:"userTitle"`
		FailingSince    *time.Time `json:"failingSince,omitempty"`
		CommitTimestamp int64      `json:"commitTimestamp"`
		CreateTimestamp int64      `json:"createTimestamp"`
	}{
		ID:              f.id,
		URL:             f.url,
		Disabled:        f.disabled,
		Title:           f.title,
		SiteURL:         f.siteURL,
		UserTitle:       f.userTitle,
		FailingSince:    f.failingSince,
		CommitTimestamp: f.commitTimestamp.Unix(),
		CreateTimestamp: f.createTimestamp.Unix(),
	})
}

// FeedSelectColumns is used by the database when reading feeds
const FeedSelectColumns string = `
feeds.id,
feeds.url,
feeds.disabled,
feeds.title,
feeds.siteurl,
feeds.usertitle,
feeds.failing_since,
feeds.commit_timestamp,
feeds.create_timestamp`

func scanFeed(feed *Feed) []interface{} {
	return []interface{}{
		&feed.id,
		&feed.url,
		&feed.disabled,
		&feed.title,
		&feed.siteURL,
		&feed.userTitle,
		&feed.failingSince,
		&feed.commitTimestamp,
		&feed.createTimestamp}
}

// ScanFeed converts one row into a feed
func ScanFeed(row *sql.Row) (*Feed, error) {
	var feed Feed
	err := row.Scan(scanFeed(&feed)...)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &feed, nil
}

// ScanFeeds converts multiple rows into feeds
func ScanFeeds(rows *sql.Rows) ([]*Feed, error) {
	feeds := []*Feed{}
	for rows.Next() {
		var feed Feed
		err := rows.Scan(scanFeed(&feed)...)
		if err != nil {
			glog.Error(err)
			return nil, err
		}
		feeds = append(feeds, &feed)
	}
	if err := rows.Err(); err != nil {
		glog.Error(err)
		return nil, err
	}
	return feeds, nil
}

// "MangaDex RSS" is a terrible title for every per-series feed
const mangadexItemRegexp = `^(.+) - [^-]+$`

var mdire = regexp.MustCompile(mangadexItemRegexp)

const mangadexSeriesRegexp = `^https://mangadex\.org/rss/[0-9a-z]+/manga_id/[0-9]+`

var mdsre = regexp.MustCompile(mangadexSeriesRegexp)

func getFeedTitle(f *Feed, gfe *gofeed.Feed) string {
	if gfe.Title != "MangaDex RSS" {
		return gfe.Title
	}

	if gfe.Items == nil || len(gfe.Items) == 0 || !mdsre.MatchString(f.url) {
		glog.Infof("%s", !mdsre.MatchString(f.url))
		return gfe.Title
	}

	groups := mdire.FindStringSubmatch(gfe.Items[0].Title)
	if groups == nil {
		return gfe.Title
	}

	glog.V(2).Infof("Overriding Mangadex RSS title with [%s]", groups[1])
	return groups[1]
}

func (f *Feed) String() string {
	title := f.title
	if f.userTitle != "" {
		title = f.userTitle
	}
	str := fmt.Sprintf("Feed %d: %s (%s)", f.id, f.url, title)
	if f.disabled {
		str += " disabled"
	}
	if f.failingSince != nil {
		str += ", failingSince: " +
			f.failingSince.Local().Format("2006-01-02 15:04:05")
	}
	return str
}

const feedUpdateSQL string = `
UPDATE
	feeds
SET
	disabled = ?,
	usertitle = ?,
	title = ?,
	siteurl = ?,
	failing_since = ?,
	commit_timestamp = CURRENT_TIMESTAMP
WHERE
	id = ?;`

func (f *Feed) update() EntityUpdate {
	return EntityUpdate{
		f,
		false,
		feedUpdateSQL,
		[]interface{}{
			f.disabled,
			f.userTitle,
			f.title,
			f.siteURL,
			f.failingSince,
			f.id}}
}

// ID gets the ID
func (f *Feed) ID() int64 { return f.id }

// URL gets the URL
func (f *Feed) URL() string { return f.url }

// FeedSetFetchFailed is a mutation function that marks a feed as failing
func FeedSetFetchFailed(t time.Time) func(f *Feed) EntityUpdate {
	return func(f *Feed) EntityUpdate {
		if f.failingSince != nil {
			return noopEntityUpdate(f)
		}

		newF := *f
		newF.failingSince = &t
		return newF.update()
	}
}

// FeedSetFetchSuccess return a mutation function that marks a feed as succeeding
func FeedSetFetchSuccess(f *Feed) EntityUpdate {
	if f.failingSince == nil {
		return noopEntityUpdate(f)
	}
	newF := *f
	newF.failingSince = nil
	return newF.update()
}

// FeedMergeGofeed returns a mutation function that merges in updates
// from the latest fetched version of the feed.
func FeedMergeGofeed(gfe *gofeed.Feed) func(*Feed) EntityUpdate {
	return func(f *Feed) EntityUpdate {
		newF := *f
		newF.title = getFeedTitle(f, gfe)

		if gfe.Link != "" {
			newF.siteURL = gfe.Link
		} else {
			if newF.siteURL == "" && !strings.HasPrefix(newF.url, "!") {
				// Default to the feed URL if it's a URL, only log f once
				glog.Warningf("Feed without link [%s]", newF)
				newF.siteURL = newF.url
			}
		}
		if newF.title == f.title && newF.siteURL == f.siteURL {
			return noopEntityUpdate(&newF)
		}
		return newF.update()
	}
}
