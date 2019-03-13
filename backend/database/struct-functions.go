package database

import (
	"database/sql"
	"time"

	. "github.com/awused/rss-aggregator/backend/structs"
	"github.com/golang/glog"
)

/**
 * Feeds
 */
func (this *Database) GetFeeds(includeDisabled bool) ([]*Feed, error) {
	glog.V(5).Info("GetFeeds() started")
	this.lock.RLock()
	defer this.lock.RUnlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	sql := "SELECT " + FeedSelectColumns + " FROM feeds"
	if !includeDisabled {
		sql = sql + " WHERE disabled = 0"
	}

	rows, err := this.db.Query(sql)
	if err != nil {
		return nil, err
	}
	feeds, err := ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("GetFeeds() retrieved %d feeds", len(feeds))
		glog.V(5).Info("GetFeeds() completed")
		return feeds, nil
	}
}

func (this *Database) UserUpdateFeed(f *Feed) error {
	glog.V(5).Info("UserUpdateFeed() started")
	this.lock.Lock()
	defer this.lock.Unlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return err
	}

	sql := "UPDATE feeds SET " + FeedUserUpdateColumns + " WHERE id = ?"
	binds := append(f.UserUpdateValues(), f.Id())

	glog.V(4).Infof("Writing user updated feed [%s]", f)
	_, err := this.db.Exec(sql, binds...)
	if err != nil {
		glog.Error(err)
		return err
	}

	glog.V(5).Info("UserUpdateFeed() completed")
	return nil
}

func (this *Database) NonUserUpdateFeed(f *Feed) error {
	glog.V(5).Info("NonUserUpdateFeed() started")
	this.lock.Lock()
	defer this.lock.Unlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return err
	}

	sql := "UPDATE feeds SET " + FeedNonUserUpdateColumns + " WHERE id = ?"
	binds := append(f.NonUserUpdateValues(), f.Id())

	glog.V(4).Infof("Writing non-user updated feed [%s]", f)
	_, err := this.db.Exec(sql, binds...)
	if err != nil {
		glog.Error(err)
		return err
	}

	glog.V(5).Info("NonUserUpdateFeed() completed")
	return nil
}

/**
 * Items
 */
func (this *Database) GetItem(id int64) (*Item, error) {
	glog.V(5).Info("GetItem() started")
	this.lock.RLock()
	defer this.lock.RUnlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	sql := "SELECT " + ItemSelectColumns + " FROM items WHERE items.id = ?"

	rows, err := this.db.Query(sql, id)
	if err != nil {
		return nil, err
	}
	items, err := ScanItems(rows)
	if err != nil {
		return nil, err
	}

	if len(items) != 1 {
		glog.Warningf("Tried to fetch item %d from the database but it did not exist", id)
		return nil, nil
	}

	glog.V(3).Infof("GetItem() retrieved item [%s]", items[0])
	return items[0], nil
}

func (this *Database) GetItems(includeRead bool) ([]*Item, error) {
	glog.V(5).Info("GetItems() started")
	this.lock.RLock()
	defer this.lock.RUnlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	sql := "SELECT " + ItemSelectColumns + " FROM items INNER JOIN feeds ON items.feedid = feeds.id WHERE feeds.disabled = 0"
	if !includeRead {
		sql = sql + " AND items.read = 0"
	}
	sql = sql + " ORDER BY timestamp DESC"

	rows, err := this.db.Query(sql)
	if err != nil {
		return nil, err
	}
	items, err := ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("GetItems() retrieved %d items", len(items))
		glog.V(5).Info("GetItems() completed")
		return items, nil
	}
}

func (this *Database) InsertItems(items []*Item) error {
	glog.V(5).Info("InsertItems() started")
	this.lock.Lock()
	defer this.lock.Unlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return err
	}

	sql := ""
	binds := []interface{}{}

	for _, i := range items {
		// TODO -- ON CONFLICT UPDATE ......
		glog.V(4).Infof("Attempting to insert [%s]", i)
		sql = sql + "INSERT OR IGNORE INTO items(" + ItemInsertColumns + ") VALUES (" + ItemInsertValues + ");\n"
		binds = append(binds, i.InsertValues()...)
	}

	glog.V(3).Infof("Inserting %d potentially new items", len(items))
	glog.V(6).Info(sql)
	glog.V(7).Info(binds)

	tx, err := this.db.Begin()
	if err != nil {
		glog.Error(err)
		return err
	}

	_, err = tx.Exec(sql, binds...)
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

func (this *Database) UpdateItem(it *Item) error {
	glog.V(5).Info("UpdateItem() started")
	this.lock.Lock()
	defer this.lock.Unlock()

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return err
	}

	sql := "UPDATE items SET " + ItemUpdateColumns + " WHERE id = ?"
	binds := append(it.UpdateValues(), it.Id())

	glog.V(2).Infof("Writing updated item [%s]", it)
	_, err := this.db.Exec(sql, binds...)
	if err != nil {
		glog.Error(err)
		return err
	}

	glog.V(5).Info("UpdateItem() completed")
	return nil
}

/**
 * Updates
 */
type Updates struct {
	Timestamp  int64   `json:"timestamp,omitempty"`
	Items      []*Item `json:"items,omitempty"`
	Feeds      []*Feed `json:"feeds,omitempty"`
	Incomplete bool    `json:"incomplete,omitempty"`
}

// Above 1000 updates to any one type we give up
// The frontend will have to resync from scratch
// TODO -- move to config file
const limit = 1000

func minTime(t time.Time, o time.Time) time.Time {
	if t.Before(o) {
		return o
	}
	return t
}

func (this *Updates) finish() {
	var t time.Time
	if len(this.Feeds) > 0 {
		t = this.Feeds[len(this.Feeds)-1].CommitTimestamp()
	}
	if len(this.Items) > 0 {
		t = minTime(t, this.Items[len(this.Items)-1].CommitTimestamp())
	}

	if !t.Equal(time.Time{}) {
		// Ensure we never miss an update
		// Updates are idempotent on the frontend
		this.Timestamp = t.Unix() - 1
	}

	if len(this.Feeds) == limit || len(this.Items) == limit {
		this.Incomplete = true
	}
}

func (this *Database) GetUpdates(t time.Time) (*Updates, error) {
	// Lock the database only once to ensure we have a consistent view of the DB
	// and never miss updates
	this.lock.RLock()
	defer this.lock.RUnlock()

	// Match sqlite's format
	tstr := t.Format("2006-01-02 15:04:05")
	up := &Updates{}

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return up, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := this.db.Begin()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return up, err
	}

	up.Feeds, err = this.getUpdatedFeeds(tx, tstr)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return up, err
	}

	up.Items, err = this.getUpdatedItems(tx, tstr)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return up, err
	}

	err = tx.Commit()
	up.finish()
	return up, err
}

func (this *Database) getUpdatedFeeds(tx *sql.Tx, tstr string) ([]*Feed, error) {
	glog.V(5).Info("getUpdatedFeeds() started")

	sql := "SELECT " + FeedSelectColumns + ` FROM feeds
		WHERE commit_timestamp > ?
		ORDER BY commit_timestamp ASC LIMIT ?;`

	rows, err := tx.Query(sql, tstr, limit)
	if err != nil {
		return nil, err
	}
	feeds, err := ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("getUpdatedFeeds() retrieved %d feeds", len(feeds))
		glog.V(5).Info("getUpdatedFeeds() completed")
		return feeds, nil
	}

}

func (this *Database) getUpdatedItems(tx *sql.Tx, tstr string) ([]*Item, error) {
	glog.V(5).Info("getUpdatedItems() started")

	sql := "SELECT " + ItemSelectColumns + ` FROM items
		WHERE commit_timestamp > ?
		ORDER BY commit_timestamp ASC LIMIT ?;`

	rows, err := tx.Query(sql, tstr, limit)
	if err != nil {
		return nil, err
	}
	items, err := ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("getUpdatedItems() retrieved %d feeds", len(items))
		glog.V(5).Info("getUpdatedItems() completed")
		return items, nil
	}

}
