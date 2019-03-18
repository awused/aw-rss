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

// Item represents a single entry in an RSS feed
type Item struct {
	id, feedID      int64
	key             string
	title           string
	url             string
	description     string
	timestamp       time.Time
	read            bool
	commitTimestamp time.Time
}

// MarshalJSON is used by the JSON marshaller
// Content is excluded and must be fetched separately to cut down on data
func (it *Item) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64     `json:"id"`
		FeedID          int64     `json:"feedId"`
		Title           string    `json:"title"`
		URL             string    `json:"url"`
		Timestamp       time.Time `json:"timestamp"`
		Read            bool      `json:"read"`
		CommitTimestamp int64     `json:"commitTimestamp"`
	}{
		ID:              it.id,
		FeedID:          it.feedID,
		Title:           it.title,
		URL:             it.url,
		Timestamp:       it.timestamp,
		Read:            it.read,
		CommitTimestamp: it.commitTimestamp.Unix(),
	})
}

// ItemSelectColumns is used by the database when reading items
const ItemSelectColumns string = `
items.id,
items.feedid,
items.key,
items.title,
items.url,
items.timestamp,
items.read,
items.commit_timestamp`

func scanItem(item *Item) []interface{} {
	return []interface{}{
		&item.id,
		&item.feedID,
		&item.key,
		&item.title,
		&item.url,
		&item.timestamp,
		&item.read,
		&item.commitTimestamp}
}

// ScanItem converts one row into an Item
func ScanItem(row *sql.Row) (*Item, error) {
	var item Item
	err := row.Scan(scanItem(&item)...)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &item, nil
}

// ScanItems converts multiple rows into Items
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

// CreateNewItems creates items without proper IDs,
// which may be duplicates of Items already present in the database.
// These should be inserted or ignored into the database then discarded.
func CreateNewItems(f *Feed, gfItems []*gofeed.Item) []*Item {
	glog.V(1).Infof("Creating %d items for [%s]", len(gfItems), f)
	var items []*Item

	for i := range gfItems {
		// Reverse order; if we need to fill in publication timestamps
		// with time.Now() they'll be in the appropriate order
		gfi := gfItems[len(gfItems)-i-1]
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

	item.feedID = f.ID()
	item.key = getKey(gfi)
	item.title = gfi.Title

	if item.feedID == 129 {
		fmt.Println(item.title)
	}

	if gfi.Link != "" {
		item.url = gfi.Link
	} else {
		glog.Infof("No link present in gofeed.Item %+v\n", gfi)
	}
	// So few feeds actually populate their full Content that it is not useful
	item.description = gfi.Description
	item.timestamp = getTimestamp(gfi)

	glog.V(3).Infof("Created [%s]", &item)
	glog.V(7).Infof("Content for new [%s] is %s", &item, item.description)
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
	if gfi.PublishedParsed != nil {
		return gfi.PublishedParsed.UTC()
	}

	// Use the updated timestamp in place of published iff there is no published
	if gfi.UpdatedParsed != nil {
		return gfi.UpdatedParsed.UTC()
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
	return fmt.Sprintf(
		"Item %d (feed %d): %s (%s) time: %s, read: %t",
		it.id, it.feedID, it.url, it.title, it.timestamp, it.read)
}

// ItemInsertColumns are the columns used at insertion time
const ItemInsertColumns string = `feedid,
key,
title,
url,
content,
timestamp,
commit_timestamp`

// ItemInsertPlaceholders are the placeholders used at insertion time
const ItemInsertPlaceholders string = "?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP"

// InsertValues returns the values to be inserted into the database
func (it *Item) InsertValues() []interface{} {
	return []interface{}{
		it.feedID,
		it.key,
		it.title,
		it.url,
		it.description,
		it.timestamp}
}

const itemUpdateSQL string = `
UPDATE
	items
SET
	read = ?,
	commit_timestamp = CURRENT_TIMESTAMP
WHERE
	id = ?;`

func (it *Item) update() EntityUpdate {
	return EntityUpdate{
		it,
		false,
		itemUpdateSQL,
		[]interface{}{
			it.read,
			it.id}}
}

// ID returns the ID
func (it *Item) ID() int64 { return it.id }

// ItemSetRead returns a mutation function that sets an item as read
func ItemSetRead(read bool) func(*Item) EntityUpdate {
	return func(it *Item) EntityUpdate {
		if it.read == read {
			return noopEntityUpdate(it)
		}
		nit := *it
		nit.read = read
		return nit.update()
	}
}
