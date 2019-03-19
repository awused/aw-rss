package database

import (
	"errors"
	"strings"
	"time"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// MutateItem applies `fn` to one item from the DB and returns it
func (d *Database) MutateItem(
	id int64,
	fn func(*structs.Item) structs.EntityUpdate) (*structs.Item, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	tx, err := d.db.Begin()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	it, err := getItem(tx, id)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	update := fn(it)
	if update.Noop() {
		err = tx.Commit()
		return it, err
	}

	err = updateEntity(tx, update)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	newIt, err := getItem(tx, id)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	return newIt, nil
}

func getItem(dot dbOrTx, id int64) (*structs.Item, error) {
	sql := entityGetSQL("items", structs.ItemSelectColumns)

	return structs.ScanItem(dot.QueryRow(sql, id))
}

// InsertItems inserts new items into the database if they aren't present
func (d *Database) InsertItems(items []*structs.Item) error {
	glog.V(5).Info("InsertItems() started")
	if len(items) == 0 {
		glog.V(2).Info("InsertItems() called with empty list")
		return nil
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return err
	}

	sql := insertSQL("items", structs.ItemInsertColumns, structs.ItemInsertPlaceholders)
	binds := []interface{}{}

	for _, i := range items {
		// TODO -- ON CONFLICT UPDATE if we want to handle updates
		glog.V(4).Infof("Attempting to insert [%s]", i)
		binds = append(binds, i.InsertValues()...)
	}

	glog.V(2).Infof("Inserting %d potentially new items", len(items))
	glog.V(6).Info(sql)
	glog.V(7).Info(binds)

	tx, err := d.db.Begin()
	if err != nil {
		glog.Error(err)
		return err
	}

	_, err = tx.Exec(strings.Repeat(sql, len(items)), binds...)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return err
	}

	glog.V(5).Info("InsertItem() completed")
	return nil
}

// GetItemsRequest is a request for items matching some constraints
// Items from disabled feeds will not be fetched unless they're specified in
// FeedIDs
// If both FeedIDs and CategoryID are absent, items from every enabled feed
// will be fetched
type GetItemsRequest struct {
	// ID of the category, if any
	CategoryID *int64
	// IDs of the Feed, if any
	FeedIDs []*int64
	// Include feeds specified by FeedIDs in the response
	IncludeFeed bool

	// If true fetch all unread items
	Unread bool
	// Fetch all read items after this timestamp (exclusive)
	ReadAfter *time.Time
	// Fetch ReadBeforeCount items before this timestamp (inclusive)
	ReadBefore *time.Time
}

// GetItemsResponse is used to fulfill the GetItemsRequest
type GetItemsResponse struct {
	Items []*structs.Item `json:"items"`
	Feeds []*structs.Feed `json:"feeds,omitempty"`
}

// GetItems returns the Items needed to fulfill the GetItemsRequest
func (d *Database) GetItems(
	req GetItemsRequest) (*GetItemsResponse, error) {
	glog.V(5).Info("GetItems() started")
	if !req.Unread && req.ReadAfter == nil && req.ReadBefore == nil {
		glog.V(2).Info("GetItems() called with empty request")
		return nil, nil
	}

	if len(req.FeedIDs) != 0 && req.CategoryID != nil {
		return nil, errors.New("Can't call GetItems for both feeds and a category")
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	// TODO
	return nil, nil
}
