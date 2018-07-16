package structs

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/golang/glog"
	"github.com/mmcdole/gofeed"
)

type Item struct {
	id, feedId               int64
	key, title, url, content string
	timestamp                time.Time
	Read                     bool
}

const ItemSelectColumns string = "items.id, items.feedid, items.key, items.title, items.url, items.content, items.timestamp, items.read"

// Content is excluded and must be fetched separately to cut down on data
func (this *Item) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        int64     `json:"id"`
		FeedId    int64     `json:"feedId"`
		Title     string    `json:"title"`
		URL       string    `json:"url"`
		Timestamp time.Time `json:"timestamp"`
		Read      bool      `json:"read"`
	}{
		ID:        this.id,
		FeedId:    this.feedId,
		Title:     this.title,
		URL:       this.url,
		Timestamp: this.timestamp,
		Read:      this.Read,
	})
}

func ScanItem(row *sql.Row) (*Item, error) {
	var item Item
	err := row.Scan(&item.id, &item.feedId, &item.key, &item.title, &item.url, &item.content, &item.timestamp, &item.Read)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &item, nil
}

func ScanItems(rows *sql.Rows) ([]*Item, error) {
	items := []*Item{}
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.id, &item.feedId, &item.key, &item.title, &item.url, &item.content, &item.timestamp, &item.Read)
		if err != nil {
			glog.Error(err)
			return nil, err
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		glog.Error(err)
		return nil, err
	}
	return items, nil
}

// Create new Items without proper IDs, which may be duplicates of Items already present in the database.
// These should be inserted or ignored into the database then discarded.
func CreateNewItems(f *Feed, gfItems []*gofeed.Item) []*Item {
	glog.V(1).Infof("Creating %d items for [%s]", len(gfItems), f)
	var items []*Item

	for _, gfi := range gfItems {
		items = append(items, createNewItem(gfi, f))
	}

	handleFeedQuirks(items, gfItems, f)

	glog.V(2).Infof("Created %d items for [Feed: %d]", len(gfItems), f.Id())
	return items
}

func createNewItem(gfi *gofeed.Item, f *Feed) *Item {
	defer func() {
		if r := recover(); r != nil {
			glog.Error(r.(error))
			glog.Errorf("%+v\n", gfi)
		}
	}()
	var item Item

	item.feedId = f.Id()
	item.key = getKey(gfi)
	item.title = gfi.Title
	if gfi.Link != "" {
		item.url = gfi.Link
	} else {
		glog.Infof("No link present in gofeed.Item %+v\n", gfi)
	}
	item.content = gfi.Description
	if gfi.Content != "" {
		item.content = gfi.Content
	}
	item.timestamp = getTimestamp(gfi)

	glog.V(3).Infof("Created [%s]", &item)
	glog.V(7).Infof("Content for new [%s] is %s", &item, item.content)
	return &item
}

// Copied from go-pkg-rss which was public domain
// https://github.com/mmcdole/gofeed/issues/95
func getKey(gfi *gofeed.Item) string {
	if gfi.GUID != "" {
		return gfi.GUID
	}

	if gfi.Title != "" && gfi.Published != "" {
		return gfi.Title + gfi.Published
	}

	h := sha256.New()
	io.WriteString(h, gfi.Description)
	return string(h.Sum(nil))
}

func getTimestamp(gfi *gofeed.Item) time.Time {
	if gfi.UpdatedParsed != nil {
		return gfi.UpdatedParsed.UTC()
	}

	if gfi.PublishedParsed != nil {
		return gfi.PublishedParsed.UTC()
	}

	glog.V(2).Infof("No parseable date for item \"%s\"", gfi.Link)
	return time.Now().UTC()
}

const fictionRegexp = `^(https?://)?(www\.)(fictionpress\.com|fanfiction\.net)/`

// Handle quirky behaviour from poorly built feed generators
func handleFeedQuirks(items []*Item, gfItems []*gofeed.Item, f *Feed) {
	/*
	 Fictionpress and Fanfiction.net use the same feed generator.
	 Instead of publishing new items, they republish the same item
	 with the same id with a new publication date.
	 Make this unambiguous by appending the timestamp.
	*/
	fre := regexp.MustCompile(fictionRegexp)
	if fre.MatchString(f.Url()) {
		for _, item := range items {
			item.key = item.key + item.timestamp.String()
		}
	}
}

func (this *Item) String() string {
	return fmt.Sprintf("Item %d (feed %d): %s (%s) time: %s, read: %t", this.id, this.feedId, this.url, this.title, this.timestamp, this.Read)
}

const ItemInsertColumns string = "feedid, key, title, url, content, timestamp"
const ItemInsertValues string = "?, ?, ?, ?, ?, ?"

func (this *Item) InsertValues() []interface{} {
	return []interface{}{this.feedId, this.key, this.title, this.url, this.content, this.timestamp}
}

const ItemUpdateColumns string = "read = ?"

func (this *Item) UpdateValues() []interface{} {
	return []interface{}{this.Read}
}

func (this *Item) Id() int64            { return this.id }
func (this *Item) FeedId() int64        { return this.feedId }
func (this *Item) Key() string          { return this.key }
func (this *Item) Title() string        { return this.title }
func (this *Item) Url() string          { return this.url }
func (this *Item) Content() string      { return this.content }
func (this *Item) Timestamp() time.Time { return this.timestamp }
