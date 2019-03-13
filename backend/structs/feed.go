package structs

// TODO -- Split this into "Feed" for data the user has entered and "FeedData"
// for automatically created data. Not happy with how this grew. They can both
// be stored in the same table.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/golang/glog"
	"github.com/mmcdole/gofeed"
)

type Feed struct {
	id             int64
	url            string
	Disabled       bool
	Title, SiteUrl string
	UserTitle      string

	LastFetchFailed bool // Whether or not the last attempt to fetch this feed failed
	// The timestamp of the last successful fetch, helps users see if a breakage is transient
	LastSuccessTime time.Time
	commitTimestamp time.Time
}

func (this *Feed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64     `json:"id"`
		URL             string    `json:"url"`
		Disabled        bool      `json:"disabled"`
		Title           string    `json:"title"`
		SiteUrl         string    `json:"siteUrl"`
		LastFetchFailed bool      `json:"lastFetchFailed"`
		UserTitle       string    `json:"userTitle"`
		LastSuccessTime time.Time `json:"lastSuccessTime"`
		CommitTimestamp int64     `json:"commitTimestamp"`
	}{
		ID:              this.id,
		URL:             this.url,
		Disabled:        this.Disabled,
		Title:           this.Title,
		SiteUrl:         this.SiteUrl,
		LastFetchFailed: this.LastFetchFailed,
		UserTitle:       this.UserTitle,
		LastSuccessTime: this.LastSuccessTime,
		CommitTimestamp: this.commitTimestamp.Unix(),
	})
}

const FeedSelectColumns string = `
feeds.id,
feeds.url,
feeds.disabled,
feeds.title,
feeds.siteurl,
feeds.lastfetchfailed,
feeds.usertitle,
feeds.lastsuccesstime,
feeds.commit_timestamp`

func scanFeed(feed *Feed) []interface{} {
	return []interface{}{
		&feed.id,
		&feed.url,
		&feed.Disabled,
		&feed.Title,
		&feed.SiteUrl,
		&feed.LastFetchFailed,
		&feed.UserTitle,
		&feed.LastSuccessTime,
		&feed.commitTimestamp}
}

func ScanFeed(row *sql.Row) (*Feed, error) {
	var feed Feed
	err := row.Scan(scanFeed(&feed)...)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &feed, nil
}

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

func (this *Feed) HandleUpdate(feed *gofeed.Feed) {
	this.Title = getFeedTitle(this, feed)

	if feed.Link != "" {
		this.SiteUrl = feed.Link
	} else {
		if this.SiteUrl == "" {
			// Default to the feed URL, only log this once
			glog.Warningf("Feed without link [%s]", this)
			this.SiteUrl = this.url
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

func (this *Feed) String() string {
	title := this.Title
	if this.UserTitle != "" {
		title = this.UserTitle
	}
	return fmt.Sprintf("Feed %d: %s (%s) disabled: %t, lastFetchFailed: %t, lastSuccessTime %s",
		this.id, this.url, title, this.Disabled, this.LastFetchFailed, this.LastSuccessTime)
}

// Columns set in response to a user's action.
const FeedUserUpdateColumns string = `
disabled = ?,
usertitle = ?,
commit_timestamp = CURRENT_TIMESTAMP`

func (this *Feed) UserUpdateValues() []interface{} {
	return []interface{}{this.Disabled, this.UserTitle}
}

// Columns set automatically by the program. Should not overlap with user set columns to avoid clobbering user data.
const FeedNonUserUpdateColumns string = `
title = ?,
siteurl = ?,
lastfetchfailed = ?,
lastsuccesstime = ?,
commit_timestamp = CURRENT_TIMESTAMP`

func (this *Feed) NonUserUpdateValues() []interface{} {
	return []interface{}{this.Title, this.SiteUrl, this.LastFetchFailed, this.LastSuccessTime}
}

func (this *Feed) Id() int64                  { return this.id }
func (this *Feed) Url() string                { return this.url }
func (this *Feed) CommitTimestamp() time.Time { return this.commitTimestamp }
