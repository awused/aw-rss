package database

import (
	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// MutateFeed applies `fn` to one feed in the DB and returns it
func (d *Database) MutateFeed(
	id int64,
	fn func(*structs.Feed) structs.EntityUpdate) (*structs.Feed, error) {
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

	f, err := getFeed(tx, id)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	update := fn(f)
	if update.Noop() {
		err = tx.Commit()
		return f, err
	}

	err = updateEntity(tx, update)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	newF, err := getFeed(tx, id)
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

	return newF, nil
}

func getFeed(dot dbOrTx, id int64) (*structs.Feed, error) {
	sql := entityGetSQL("feeds", structs.FeedSelectColumns)

	return structs.ScanFeed(dot.QueryRow(sql, id))
}

// GetFeed gets a single (disabled) feed when requested by the frontend
// func (d *Database) GetFeed

// GetDisabledFeeds returns all disabled feeds from the database for the admin
// page. There's no support for pagination as it's assumed the number of feeds
// will never be that excessively large.
// func (d *Database) GetDisabledFeeds
