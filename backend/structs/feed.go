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
	id             int64
	url            string
	Disabled       bool
	Title, SiteURL string
	UserTitle      string

	LastFetchFailed bool // Whether or not the last attempt to fetch this feed failed
	// The timestamp of the last successful fetch, helps users see if a breakage is transient
	LastSuccessTime time.Time
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
		Disabled:        f.Disabled,
		Title:           f.Title,
		SiteURL:         f.SiteURL,
		LastFetchFailed: f.LastFetchFailed,
		UserTitle:       f.UserTitle,
		LastSuccessTime: f.LastSuccessTime,
		CommitTimestamp: f.commitTimestamp.Unix(),
		CreateTimestamp: f.createTimestamp.Unix(),
	})
}

// FeedSelectColumns is used by the database
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
		&feed.Disabled,
		&feed.Title,
		&feed.SiteURL,
		&feed.LastFetchFailed,
		&feed.UserTitle,
		&feed.LastSuccessTime,
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

// HandleUpdate merges in updates from the latest fetched version of the feed
func (f *Feed) HandleUpdate(feed *gofeed.Feed) {
	f.Title = getFeedTitle(f, feed)

	if feed.Link != "" {
		f.SiteURL = feed.Link
	} else {
		if f.SiteURL == "" && !strings.HasPrefix(f.url, "!") {
			// Default to the feed URL if it's a URL, only log f once
			glog.Warningf("Feed without link [%s]", f)
			f.SiteURL = f.url
		}
	}
}

// "MangaDex RSS" is a terrible title for every per-series feed
const mangadexItemRegexp = `^(.+) - [^-]+$`

var mdire = regexp.MustCompile(mangadexItemRegexp)

const mangadexSeriesRegexp = `^https://mangadex\.org/rss/[0-9a-z]+/manga_id/[0-9]+`

var mdsre = regexp.MustCompile(mangadexSeriesRegexp)

func getFeedTitle(f *Feed, feed *gofeed.Feed) string {
	if feed.Title != "MangaDex RSS" {
		return feed.Title
	}

	if feed.Items == nil || len(feed.Items) == 0 || !mdsre.MatchString(f.url) {
		glog.Infof("%s", !mdsre.MatchString(f.url))
		return feed.Title
	}

	groups := mdire.FindStringSubmatch(feed.Items[0].Title)
	if groups == nil {
		return feed.Title
	}

	glog.V(2).Infof("Overriding Mangadex RSS title with [%s]", groups[1])
	return groups[1]
}

func (f *Feed) String() string {
	title := f.Title
	if f.UserTitle != "" {
		title = f.UserTitle
	}
	return fmt.Sprintf("Feed %d: %s (%s) disabled: %t, lastFetchFailed: %t, lastSuccessTime %s",
		f.id, f.url, title, f.Disabled, f.LastFetchFailed, f.LastSuccessTime)
}

// FeedUserUpdateColumns defines the set of columns that users are able to update
const FeedUserUpdateColumns string = `
disabled = ?,
usertitle = ?,
commit_timestamp = CURRENT_TIMESTAMP`

// UserUpdateValues fills in the values used above
func (f *Feed) UserUpdateValues() []interface{} {
	return []interface{}{f.Disabled, f.UserTitle}
}

// FeedNonUserUpdateColumns defines the set of columns updated automatically.
// Must not overlap with user set columns to avoid clobbering user data.
const FeedNonUserUpdateColumns string = `
title = ?,
siteurl = ?,
lastfetchfailed = ?,
lastsuccesstime = ?,
commit_timestamp = CURRENT_TIMESTAMP`

// NonUserUpdateValues fills in the values used above
func (f *Feed) NonUserUpdateValues() []interface{} {
	return []interface{}{f.Title, f.SiteURL, f.LastFetchFailed, f.LastSuccessTime}
}

// ID gets the ID
func (f *Feed) ID() int64 { return f.id }

// URL gets the URL
func (f *Feed) URL() string { return f.url }
