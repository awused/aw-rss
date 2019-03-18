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

// GetItemsRequest is a request for items for a category or multiple feeds
type GetItemsRequest struct {
	// ID of the category.
	CategoryID *int64
	// This will never be set at the same time as the CategoryID
	FeedIDs []int64
	// If true fetch all unread items
	Unread bool
	// Fetch all read items after this timestamp (with generally larger IDs)
	ReadAfter *time.Time
	// Fetch ReadBeforeCount items before this timestamp (generally smaller IDs)
	ReadBefore *time.Time
	// Limit how many items are fetched at one time
	ReadBeforeCount *int64
}

// GetItemsResponse is used to fulfill the GetItemsRequest
type GetItemsResponse struct {
	Items []*structs.Item `json:"items"`
}

// GetItems returns the Items needed to fulfill the GetItemsRequest
func (d *Database) GetItems(
	req GetItemsRequest) (*GetItemsResponse, error) {
	glog.V(5).Info("GetBatchItems() started")
	if len(req.FeedIDs) == 0 && req.CategoryID == nil {
		glog.V(2).Info("GetBatchItems() called with empty request")
		return nil, nil
	}

	if len(req.FeedIDs) != 0 && req.CategoryID != nil {
		return nil, errors.New("Can't call BatchItems for both feeds and a category")
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
