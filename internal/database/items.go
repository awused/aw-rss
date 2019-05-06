package database

import (
	"errors"
	"strings"
	"time"

	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

// MutateItem applies `fn` to one item from the DB and returns it
func (d *Database) MutateItem(
	id int64,
	fn func(*structs.Item) structs.EntityUpdate) (*structs.Item, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer tx.Rollback()

	it, err := getItem(tx, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	update := fn(it)
	if update.Noop() {
		err = tx.Commit()
		return it, err
	}

	err = updateEntity(tx, update)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	newIt, err := getItem(tx, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return newIt, nil
}

func getItem(dot dbOrTx, id int64) (*structs.Item, error) {
	sql := entityGetSQL("items", structs.ItemSelectColumns)

	return structs.ScanItem(dot.QueryRow(sql, id))
}

func getItems(dot dbOrTx, ids []int64) ([]*structs.Item, error) {
	sql := entityBatchGetSQL("items", structs.ItemSelectColumns, len(ids))
	// Ugly
	binds := make([]interface{}, len(ids), len(ids))
	for i, v := range ids {
		binds[i] = v
	}

	rows, err := dot.Query(sql, binds...)
	if err != nil {
		return nil, err
	}
	return structs.ScanItems(rows)
}

// InsertItems inserts new items into the database if they aren't present.
// The order of the items matters, as item IDs are used to break timestamp ties
// when sorting on the frontend.
func (d *Database) InsertItems(items []*structs.Item) error {
	if len(items) == 0 {
		log.Trace("InsertItems() called with empty list")
		return nil
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return err
	}

	insertColumns := structs.ItemInsertColumns
	insertPlaceholders := structs.ItemInsertPlaceholders

	if d.conf.Dedupe {
		// Allow duplicates within the same feed
		// The unhandled corner case is where multiple feeds have 'legitimate'
		// duplicates, but I don't think it's worth the effort to handle.
		insertColumns += ", read"
		insertPlaceholders += `,
				(SELECT EXISTS (SELECT url FROM items WHERE url = ? AND feedid != ?))`
	}

	sql := insertSQL("items", insertColumns, insertPlaceholders)
	binds := []interface{}{}

	for _, i := range items {
		// TODO -- ON CONFLICT UPDATE if we want to handle updates
		log.Tracef("Attempting to insert [%s]", i)
		insertValues := i.InsertValues()
		if d.conf.Dedupe {
			insertValues = append(insertValues, i.URL(), i.FeedID())
		}

		binds = append(binds, insertValues...)
	}

	log.Debugf("Inserting %d potentially new items", len(items))

	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(strings.Repeat(sql, len(items)), binds...)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// GetItemsRequest is a request for items matching some constraints
// Items from disabled feeds will not be fetched unless they're specified in
// FeedIDs
// If both FeedIDs and CategoryID are absent, items from every enabled feed
// will be fetched
type GetItemsRequest struct {
	// ID of the category, if any
	CategoryID *int64 `json:"categoryId"`
	// IDs of the Feeds, if any
	FeedIDs []int64 `json:"feedIds"`
	// Include feeds specified by FeedIDs in the response
	// Does not work for categories
	IncludeFeeds bool `json:"includeFeeds"`

	// If true fetch all unread items
	Unread bool `json:"unread"`
	// Note that they will be filtered by timestamp, but ordered by ID.
	ReadBeforeCount int `json:"readBeforeCount"`
	// Fetch _at least_ ReadBeforeCount items before this timestamp (exclusive)
	// Guaranteed that all existing read items between ReadBefore and the minimum
	// timestamp in the response are fetched.
	ReadBefore *time.Time `json:"readBefore"`

	// Fetch _all_ read items after this timestamp (inclusive)
	// This is used when backfilling on the frontend, but only in the rare
	// case where a category is open and a feed is added to it or re-enabled.
	ReadAfter *time.Time `json:"readAfter"`
}

// GetItemsResponse is used to fulfill the GetItemsRequest
type GetItemsResponse struct {
	Items []*structs.Item `json:"items"`
	Feeds []*structs.Feed `json:"feeds,omitempty"`
	// Don't include the categories, if any
	// The frontend either has it or will on refresh
}

// GetItems returns the Items needed to fulfill the GetItemsRequest
func (d *Database) GetItems(
	req GetItemsRequest) (*GetItemsResponse, error) {
	var err error
	resp := GetItemsResponse{
		Items: []*structs.Item{}}

	if !req.Unread && req.ReadAfter == nil && req.ReadBefore == nil {
		log.Info("GetItems() called with empty request")
		return &resp, nil
	}

	if len(req.FeedIDs) != 0 && req.CategoryID != nil {
		return nil, errors.New("Can't call GetItems for both feeds and a category")
	}

	if req.Unread && len(req.FeedIDs) == 0 {
		return nil, errors.New("Can't request unread except by feeds")
	}

	if (req.ReadAfter != nil) && (req.ReadBefore != nil) {
		return nil, errors.New("Must not specify both ReadBefore and ReadAfter")
	}

	if req.ReadAfter != nil {
		*req.ReadAfter = req.ReadAfter.UTC().Truncate(time.Second)
	}

	if req.ReadBefore != nil {
		*req.ReadBefore = req.ReadBefore.UTC().Truncate(time.Second)
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	if !req.IncludeFeeds {
		resp.Items, err = getItemsFor(d.db, req)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		return &resp, nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer tx.Rollback()

	resp.Items, err = getItemsFor(tx, req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if req.IncludeFeeds && req.FeedIDs != nil {
		resp.Feeds, err = getFeeds(tx, req.FeedIDs)
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &resp, nil
}

func getItemsFor(dot dbOrTx, req GetItemsRequest) ([]*structs.Item, error) {
	selectSQL := `SELECT ` + structs.ItemSelectColumns
	sql := `
			FROM
					feeds CROSS JOIN items on items.feedid = feeds.id
			WHERE `
	binds := []interface{}{}

	if req.CategoryID != nil {
		sql += ` feeds.categoryid = ? `
		binds = append(binds, req.CategoryID)
	} else if len(req.FeedIDs) != 0 {
		placeholders := make([]string, len(req.FeedIDs), len(req.FeedIDs))
		for i, v := range req.FeedIDs {
			placeholders[i] = "?"
			binds = append(binds, v)
		}

		sql += `
				feeds.id IN (` + strings.Join(placeholders, ", ") + `) `
	} else {
		sql += ` feeds.disabled = 0 `
	}

	if req.ReadAfter != nil && req.ReadBefore != nil {
		// Pretty sure this is unnecessary
		return nil, errors.New("unimplemented")
	} else if req.ReadAfter != nil {
		where := "items.read = 1 AND items.timestamp >= ?"
		binds = append(binds, req.ReadAfter)

		if req.Unread {
			where = "((" + where + ") OR items.read = 0)"
		}
		sql += ` AND ` + where + ` `
	} else if req.ReadBefore != nil {
		count := req.ReadBeforeCount
		if count <= 0 {
			count = 100
		}

		// Timestamps are truncacted to the second for consistency, so it's
		// entirely possible to have many at the same timestamp, especially
		// from feeds that do not provide timestamps.
		// Ensure there are no gaps by fetching 'count' items, getting the
		// timestamp, then using that.
		// This _can_ be done in one sql statement, but it's unwieldly
		timestampSQL := `
				SELECT MIN(timestamp)
				FROM (SELECT items.timestamp ` + sql + `
						AND items.read = 1 AND items.timestamp < ?
				ORDER BY items.timestamp DESC
				LIMIT ?)`
		timestampBinds := append(binds, req.ReadBefore, count)

		row := dot.QueryRow(timestampSQL, timestampBinds...)
		var b []uint8
		err := row.Scan(&b)
		if err != nil {
			log.Error(err)
			// Couldn't get a minimum timestamp -> there are no read items
			// TODO -- If we need to handle ReadBefore and Unread, change this
			return []*structs.Item{}, nil
		}
		minTimestamp := string(b)

		where := "items.read = 1 AND items.timestamp < ? AND items.timestamp >= ?"
		binds = append(binds, req.ReadBefore, minTimestamp)

		if req.Unread {
			// This shouldn't be necessary
			return nil, errors.New("unimplemented")
		}

		sql += ` AND ` + where
	} else {
		sql += ` AND items.read = 0 `
	}

	sql += ` ORDER BY items.id ASC;`

	rows, err := dot.Query(selectSQL+sql, binds...)
	if err != nil {
		return nil, err
	}

	return structs.ScanItems(rows)
}

// MarkItemsReadByFeed marks all items up to maxID as read for the feed
func (d *Database) MarkItemsReadByFeed(feedID int64, maxID int64) (
	[]*structs.Item, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer tx.Rollback()

	// It's not really optimal to read the full items here, but it's safer
	// to use the existing item mutation functions
	selectSQL := `
			SELECT ` + structs.ItemSelectColumns + `
			FROM items
			WHERE feedid = ? AND read = 0 AND id <= ?;`
	rows, err := tx.Query(selectSQL, feedID, maxID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	items, err := structs.ScanItems(rows)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	updateSQL := []string{}
	updateBinds := []interface{}{}
	ids := []int64{}

	for _, it := range items {
		ids = append(ids, it.ID())

		update := structs.ItemSetRead(true)(it)
		sql, binds := update.Get()
		updateSQL = append(updateSQL, sql)
		updateBinds = append(updateBinds, binds...)
	}

	_, err = tx.Exec(strings.Join(updateSQL, "\n"), updateBinds...)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	updatedItems, err := getItems(tx, ids)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return updatedItems, nil
}
