package database

import (
	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// GetCurrentFeeds returns the set of enabled feeds
// It's currently only used by the backend on initialization
// and as a guard against out of band edits.
func (d *Database) GetCurrentFeeds() ([]*structs.Feed, error) {
	glog.V(5).Info("GetFeeds() started")
	d.lock.RLock()
	defer d.lock.RUnlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	return getCurrentFeeds(d.db)
}

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

// GetFeed gets a single (always disabled) feed when requested by the frontend
// func (d *Database) GetFeed

// GetDisabledFeeds returns all disabled feeds from the database for the admin
// page. There's no support for pagination or filtering as it's assumed the
// number of feeds will never be prohibitively large.
// func (d *Database) GetDisabledFeeds

func getCurrentFeeds(dot dbOrTx) ([]*structs.Feed, error) {
	glog.V(5).Info("getCurrentFeeds() started")

	sql := "SELECT " + structs.FeedSelectColumns + `
	    FROM
					feeds
			WHERE
					feeds.disabled = 0
			ORDER BY feeds.id ASC`

	rows, err := dot.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds, err := structs.ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("getCurrentFeeds() retrieved %d feeds", len(feeds))
	glog.V(5).Info("getCurrentFeeds() completed")
	return feeds, nil
}

// InsertNewFeed creates and inserts a new feed from the url and user title
func (d *Database) InsertNewFeed(url string, userTite string) (
	*structs.Feed, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	glog.Info("Adding new feed [%s]", url)

	sql := `INSERT INTO feeds(url, usertitle) VALUES (?, ?);`
	res, err := d.db.Exec(sql, url, userTite)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return getFeed(d.db, id)
}
