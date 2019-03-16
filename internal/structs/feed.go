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

	lastFetchFailed bool // Whether or not the last attempt to fetch this feed failed
	// The timestamp of the last successful fetch, helps users see if a breakage is transient
	lastSuccessTime time.Time
	commitTimestamp time.Time
	createTimestamp time.Time
}

// MarshalJSON is used by the JSON marshaller
func (f *Feed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64     `json:"id"`
		URL             string    `json:"url"`
		Disabled        bool      `json:"disabled"`
		Title           string    `json:"title"`
		SiteURL         string    `json:"siteUrl"`
		LastFetchFailed bool      `json:"lastFetchFailed"`
		UserTitle       string    `json:"userTitle"`
		LastSuccessTime time.Time `json:"lastSuccessTime"`
		CommitTimestamp int64     `json:"commitTimestamp"`
		CreateTimestamp int64     `json:"createTimestamp"`
	}{
		ID:              f.id,
		URL:             f.url,
		Disabled:        f.disabled,
		Title:           f.title,
		SiteURL:         f.siteURL,
		LastFetchFailed: f.lastFetchFailed,
		UserTitle:       f.userTitle,
		LastSuccessTime: f.lastSuccessTime,
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
feeds.lastfetchfailed,
feeds.usertitle,
feeds.lastsuccesstime,
feeds.commit_timestamp,
feeds.create_timestamp`

func scanFeed(feed *Feed) []interface{} {
	return []interface{}{
		&feed.id,
		&feed.url,
		&feed.disabled,
		&feed.title,
		&feed.siteURL,
		&feed.lastFetchFailed,
		&feed.userTitle,
		&feed.lastSuccessTime,
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
	return fmt.Sprintf("Feed %d: %s (%s) disabled: %t, lastFetchFailed: %t, lastSuccessTime %s",
		f.id, f.url, title, f.disabled, f.lastFetchFailed, f.lastSuccessTime)
}

const feedUpdateSQL string = `
UPDATE
	feeds
SET
	disabled = ?,
	usertitle = ?,
	title = ?,
	siteurl = ?,
	lastfetchfailed = ?,
	lastsuccesstime = ?,
	commit_timestamp = CURRENT_TIMESTAMP
WHERE
	id = ?;`

// UpdateSQL transforms the feed into a SQL update statement
func (f *Feed) UpdateSQL() EntityUpdate {
	return EntityUpdate{
		feedUpdateSQL,
		[]interface{}{
			f.disabled,
			f.userTitle,
			f.title,
			f.siteURL,
			f.lastFetchFailed,
			f.lastSuccessTime,
			f.id}}
}

// ID gets the ID
func (f *Feed) ID() int64 { return f.id }

// URL gets the URL
func (f *Feed) URL() string { return f.url }

// FeedSetLastFetchFailed is a mutation function that marks a feed as failing
func FeedSetLastFetchFailed(f *Feed) *Feed {
	newF := *f
	newF.lastFetchFailed = true
	return &newF
}

// FeedSetFetchSuccess return a mutation function that marks a feed as succeeding
func FeedSetFetchSuccess(t time.Time) func(f *Feed) *Feed {
	return func(f *Feed) *Feed {
		newF := *f
		newF.lastFetchFailed = false
		newF.lastSuccessTime = t
		return &newF
	}
}

// FeedMergeGofeedOnSuccess returns a mutation function that merges in updates
// from the latest fetched version of the feed.
func FeedMergeGofeedOnSuccess(gfe *gofeed.Feed) func(*Feed) *Feed {
	return func(f *Feed) *Feed {
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
		return &newF
	}
}
