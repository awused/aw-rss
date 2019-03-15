package database

import (
	"database/sql"
	"strconv"
	"time"

	. "github.com/awused/aw-rss/backend/structs"
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
	defer rows.Close()
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
	binds := append(f.UserUpdateValues(), f.ID())

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
	binds := append(f.NonUserUpdateValues(), f.ID())

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
	defer rows.Close()
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
	defer rows.Close()
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

// TODO -- All these methods are sloppy with how much work they do in the
// critical sections

/**
 * Initial data or on full refresh.
 */
type CurrentState struct {
	Timestamp int64   `json:"timestamp"`
	Items     []*Item `json:"items,omitempty"`
	Feeds     []*Feed `json:"feeds,omitempty"`
	// Categoriess []*Category `json:"categories,omitempty"`
	// For now, at least, this is always all of the data at once
	// If pagination support is added it will only be for items
}

func (this *Database) getTransactionTimestamp(tx *sql.Tx) (int64, error) {
	var b []uint8
	// https://github.com/mattn/go-sqlite3/issues/316
	err := tx.QueryRow("SELECT strftime('%s','now')").Scan(&b)
	if err != nil {
		return 0, err
	}
	t, err := strconv.ParseInt(string(b), 10, 64)
	// Ensure we never miss an update
	// Updates are idempotent on the frontend
	return t - 1, err
}

func (this *Database) GetCurrentState() (*CurrentState, error) {
	// Lock the database only once to ensure we have a consistent view of the DB
	// and never miss updates
	this.lock.RLock()
	defer this.lock.RUnlock()

	// Match sqlite's format
	cs := &CurrentState{}

	if err := this.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := this.db.Begin()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Timestamp, err = this.getTransactionTimestamp(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Feeds, err = this.getCurrentFeeds(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Items, err = this.getCurrentItems(tx)
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
	return cs, err
}

func (this *Database) getCurrentFeeds(tx *sql.Tx) ([]*Feed, error) {
	glog.V(5).Info("getCurrentFeeds() started")

	sql := "SELECT " + FeedSelectColumns + `
	    FROM
					feeds
			WHERE
					feeds.disabled = 0
			ORDER BY feeds.id ASC`

	rows, err := tx.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds, err := ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("getCurrentFeeds() retrieved %d feeds", len(feeds))
		glog.V(5).Info("getCurrentFeeds() completed")
		return feeds, nil
	}
}
func (this *Database) getCurrentItems(tx *sql.Tx) ([]*Item, error) {
	glog.V(5).Info("getCurrentItems() started")

	sql := "SELECT " + ItemSelectColumns + `
	    FROM
					feeds CROSS JOIN items ON items.feedid = feeds.id
			WHERE
					feeds.disabled = 0 AND items.read = 0
			ORDER BY items.id ASC`

	rows, err := tx.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("getCurrentItems() retrieved %d items", len(items))
		glog.V(5).Info("getCurrentItems() completed")
		return items, nil
	}
}

/**
 * Updates
 */
type Updates struct {
	Timestamp int64   `json:"timestamp,omitempty"`
	Items     []*Item `json:"items,omitempty"`
	Feeds     []*Feed `json:"feeds,omitempty"`
	// Categoriess []*Category `json:"categories,omitempty"`
	// When the client's state is too old and they should refresh instead
	MustRefresh bool `json:"mustRefresh,omitempty"`
}

var maxClientStaleness = time.Duration(time.Hour * 24 * 7)

// The frontend will have to resync from scratch
// TODO -- Make it possible to catch up from Incomplete Updates
// sort by commitTimestamp ASC first, then sort again by ID

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
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := this.db.Begin()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Timestamp, err = this.getTransactionTimestamp(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	newT := time.Unix(up.Timestamp, 0).UTC()
	if t.Add(maxClientStaleness).Before(newT) {
		up.MustRefresh = true
		err = tx.Commit()
		if err != nil {
			glog.Error(err)
			tx.Rollback()
			return nil, err
		}
		return up, nil
	}

	up.Feeds, err = this.getUpdatedFeeds(tx, tstr)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Items, err = this.getUpdatedItems(tx, tstr)
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
	return up, err
}

func (this *Database) getUpdatedFeeds(tx *sql.Tx, tstr string) ([]*Feed, error) {
	glog.V(5).Info("getUpdatedFeeds() started")

	sql := "SELECT " + FeedSelectColumns + `
		FROM feeds INDEXED BY feeds_commit_index
		WHERE commit_timestamp > ?
		ORDER BY feeds.id ASC;`

	rows, err := tx.Query(sql, tstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

	sql := "SELECT " + ItemSelectColumns + `
		FROM items INDEXED BY items_commit_index
		WHERE commit_timestamp > ?
		ORDER BY items.id ASC;`

	rows, err := tx.Query(sql, tstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	} else {
		glog.V(3).Infof("getUpdatedItems() retrieved %d items", len(items))
		glog.V(5).Info("getUpdatedItems() completed")
		return items, nil
	}

}
