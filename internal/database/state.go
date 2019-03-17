package database

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// CurrentState contains the initial data sent to the client
type CurrentState struct {
	Timestamp int64           `json:"timestamp"`
	Items     []*structs.Item `json:"items,omitempty"`
	Feeds     []*structs.Feed `json:"feeds,omitempty"`
	// Categoriess []*Category `json:"categories,omitempty"`
	// Timestamps of the newest items in each feed
	NewestTimestamps map[int64]time.Time `json:"newestTimestamps,omitempty"`
	// For now, at least, this is always all of the data at once
	// If pagination support is added it will only be for items
}

func (d *Database) getTransactionTimestamp(tx *sql.Tx) (int64, error) {
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

// GetCurrentState returns the current state of all "valid" entities in the DB
func (d *Database) GetCurrentState() (*CurrentState, error) {
	// Lock the database only once to ensure we have a consistent view of the DB
	// and never miss updates
	d.lock.RLock()
	defer d.lock.RUnlock()

	// Match sqlite's format
	cs := &CurrentState{}

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := d.db.Begin()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Timestamp, err = d.getTransactionTimestamp(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Feeds, err = d.getCurrentFeeds(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Items, err = d.getCurrentItems(tx)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.NewestTimestamps, err = d.getNewestTimestamps(tx)
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

func (d *Database) getCurrentFeeds(tx *sql.Tx) ([]*structs.Feed, error) {
	glog.V(5).Info("getCurrentFeeds() started")

	sql := "SELECT " + structs.FeedSelectColumns + `
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
	feeds, err := structs.ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("getCurrentFeeds() retrieved %d feeds", len(feeds))
	glog.V(5).Info("getCurrentFeeds() completed")
	return feeds, nil
}

func (d *Database) getCurrentItems(tx *sql.Tx) ([]*structs.Item, error) {
	glog.V(5).Info("getCurrentItems() started")

	sql := "SELECT " + structs.ItemSelectColumns + `
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
	items, err := structs.ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("getCurrentItems() retrieved %d items", len(items))
	glog.V(5).Info("getCurrentItems() completed")
	return items, nil
}

func (d *Database) getNewestTimestamps(tx *sql.Tx) (
	map[int64]time.Time, error) {
	sql := `
SELECT
	A.feedid, items.timestamp
FROM (
	SELECT feedid, MAX(items.id) AS id
	FROM feeds
	CROSS JOIN items
	ON items.feedid = feeds.id
	WHERE feeds.disabled = 0
	GROUP BY feedid) AS A
INNER JOIN items ON A.id = items.id`

	rows, err := tx.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]time.Time)

	for rows.Next() {
		var fid int64
		var t time.Time
		err = rows.Scan(&fid, &t)
		if err != nil {
			glog.Error(err)
			return nil, err
		}
		out[fid] = t
	}

	return out, nil
}

// Updates contains all the entities that have changed after a given time
type Updates struct {
	Timestamp int64           `json:"timestamp,omitempty"`
	Items     []*structs.Item `json:"items,omitempty"`
	Feeds     []*structs.Feed `json:"feeds,omitempty"`
	// Categoriess []*Category `json:"categories,omitempty"`
	// When the client's state is too old and they should refresh instead
	MustRefresh bool `json:"mustRefresh,omitempty"`
}

var maxClientStaleness = time.Duration(time.Hour * 24 * 7)

// GetUpdates gets all the updates since `t`
func (d *Database) GetUpdates(t time.Time) (*Updates, error) {
	// Lock the database only once to ensure we have a consistent view of the DB
	// and never miss updates
	d.lock.RLock()
	defer d.lock.RUnlock()

	// Match sqlite's format
	tstr := t.Format("2006-01-02 15:04:05")
	up := &Updates{}

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := d.db.Begin()
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Timestamp, err = d.getTransactionTimestamp(tx)
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

	up.Feeds, err = d.getUpdatedFeeds(tx, tstr)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Items, err = d.getUpdatedItems(tx, tstr)
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

func (d *Database) getUpdatedFeeds(tx *sql.Tx, tstr string) ([]*structs.Feed, error) {
	glog.V(5).Info("getUpdatedFeeds() started")

	sql := "SELECT " + structs.FeedSelectColumns + `
		FROM feeds INDEXED BY feeds_commit_index
		WHERE commit_timestamp > ?
		ORDER BY feeds.id ASC;`

	rows, err := tx.Query(sql, tstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds, err := structs.ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("getUpdatedFeeds() retrieved %d feeds", len(feeds))
	glog.V(5).Info("getUpdatedFeeds() completed")
	return feeds, nil
}

func (d *Database) getUpdatedItems(tx *sql.Tx, tstr string) ([]*structs.Item, error) {
	glog.V(5).Info("getUpdatedItems() started")

	sql := "SELECT " + structs.ItemSelectColumns + `
		FROM items INDEXED BY items_commit_index
		WHERE commit_timestamp > ?
		ORDER BY items.id ASC;`

	rows, err := tx.Query(sql, tstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := structs.ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("getUpdatedItems() retrieved %d items", len(items))
	glog.V(5).Info("getUpdatedItems() completed")
	return items, nil
}
