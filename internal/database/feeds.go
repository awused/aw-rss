package database

import (
	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// MutateFeed applies `fn` to one feed in the DB and returns it
func (d *Database) MutateFeed(
	id int64,
	fn func(*structs.Feed) *structs.Feed) (*structs.Feed, error) {
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

	updated := fn(f)
	err = updateEntity(tx, updated)
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
