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
	commitTimestamp          time.Time
}

// Content is excluded and must be fetched separately to cut down on data
func (it *Item) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64     `json:"id"`
		FeedId          int64     `json:"feedId"`
		Title           string    `json:"title"`
		URL             string    `json:"url"`
		Timestamp       time.Time `json:"timestamp"`
		Read            bool      `json:"read"`
		CommitTimestamp int64     `json:"commitTimestamp"`
	}{
		ID:              it.id,
		FeedId:          it.feedId,
		Title:           it.title,
		URL:             it.url,
		Timestamp:       it.timestamp,
		Read:            it.Read,
		CommitTimestamp: it.commitTimestamp.Unix(),
	})
}

const ItemSelectColumns string = `
items.id,
items.feedid,
items.key,
items.title,
items.url,
items.content,
items.timestamp,
items.read,
items.commit_timestamp`

func scanItem(item *Item) []interface{} {
	return []interface{}{
		&item.id,
		&item.feedId,
		&item.key,
		&item.title,
		&item.url,
		&item.content,
		&item.timestamp,
		&item.Read,
		&item.commitTimestamp}
}

func ScanItem(row *sql.Row) (*Item, error) {
	var item Item
	err := row.Scan(scanItem(&item)...)
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
		err := rows.Scan(scanItem(&item)...)
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

	handleItemQuirks(items, gfItems, f)

	glog.V(2).Infof("Created %d items for [Feed: %d]", len(gfItems), f.ID())
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

	item.feedId = f.ID()
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

/*
 Fictionpress and Fanfiction.net use the same feed generator.
 Instead of publishing new items, they republish the same item
 with the same id with a new publication date.
 Make this unambiguous by appending the timestamp.
*/
var fre = regexp.MustCompile(fictionRegexp)

// Handle quirky behaviour from poorly built feed generators
func handleItemQuirks(items []*Item, gfItems []*gofeed.Item, f *Feed) {
	if fre.MatchString(f.URL()) {
		for _, item := range items {
			item.key = item.key + item.timestamp.String()
		}
	}
}

func (it *Item) String() string {
	return fmt.Sprintf("Item %d (feed %d): %s (%s) time: %s, read: %t", it.id, it.feedId, it.url, it.title, it.timestamp, it.Read)
}

const ItemInsertColumns string = "feedid, key, title, url, content, timestamp, commit_timestamp"
const ItemInsertValues string = "?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP"

func (it *Item) InsertValues() []interface{} {
	return []interface{}{it.feedId, it.key, it.title, it.url, it.content, it.timestamp}
}

const ItemUpdateColumns string = "read = ?"

func (it *Item) UpdateValues() []interface{} {
	return []interface{}{it.Read}
}

func (it *Item) Id() int64            { return it.id }
func (it *Item) FeedId() int64        { return it.feedId }
func (it *Item) Key() string          { return it.key }
func (it *Item) Title() string        { return it.title }
func (it *Item) Url() string          { return it.url }
func (it *Item) Content() string      { return it.content }
func (it *Item) Timestamp() time.Time { return it.timestamp }
