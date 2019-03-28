package database

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

// TODO -- move these request/response objects to a separate package

// CurrentState contains the initial data sent to the client
type CurrentState struct {
	Timestamp   int64               `json:"timestamp"`
	Items       []*structs.Item     `json:"items,omitempty"`
	Feeds       []*structs.Feed     `json:"feeds,omitempty"`
	Categoriess []*structs.Category `json:"categories,omitempty"`
	// Timestamps of the newest items in each feed
	NewestTimestamps map[int64]time.Time `json:"newestTimestamps,omitempty"`
	// For now, at least, this is always all of the data at once
	// If pagination support is added it will only be for items
}

func getTransactionTimestamp(tx *sql.Tx) (int64, error) {
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
		log.Error(err)
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Timestamp, err = getTransactionTimestamp(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Categoriess, err = getCurrentCategories(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Feeds, err = getCurrentFeeds(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	cs.Items, err = getCurrentItems(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	// This is actually the slowest query but latency is dominated by reading
	// Item content
	cs.NewestTimestamps, err = getNewestTimestamps(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}
	return cs, err
}

func getCurrentCategories(tx *sql.Tx) ([]*structs.Category, error) {
	sql := "SELECT " + structs.CategorySelectColumns + `
	    FROM
					categories
			WHERE
					categories.disabled = 0
			ORDER BY categories.id ASC`

	rows, err := tx.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats, err := structs.ScanCategories(rows)

	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debugf("getCurrentCategories() retrieved %d categories", len(cats))
	return cats, nil
}

func getCurrentItems(tx *sql.Tx) ([]*structs.Item, error) {
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
		log.Error(err)
		return nil, err
	}
	log.Debugf("getCurrentItems() retrieved %d items", len(items))
	return items, nil
}

func getNewestTimestamps(tx *sql.Tx) (
	map[int64]time.Time, error) {
	sql := `
SELECT
		feedid,
		MAX(items.timestamp)
FROM items
INNER JOIN feeds
		ON feeds.id = items.feedid
WHERE feeds.disabled = 0
GROUP BY feedid;`

	rows, err := tx.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]time.Time)

	for rows.Next() {
		var fid int64
		var t []uint8
		err = rows.Scan(&fid, &t)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		out[fid], err = time.Parse("2006-01-02 15:04:05Z07:00", string(t))
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

// Updates contains all the entities that have changed after a given time
type Updates struct {
	Timestamp  int64               `json:"timestamp,omitempty"`
	Items      []*structs.Item     `json:"items,omitempty"`
	Feeds      []*structs.Feed     `json:"feeds,omitempty"`
	Categories []*structs.Category `json:"categories,omitempty"`
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
		log.Error(err)
		return nil, err
	}

	// A transaction minimizes the number of locks and prevents external modifications
	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Timestamp, err = getTransactionTimestamp(tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	newT := time.Unix(up.Timestamp, 0).UTC()
	if t.Add(maxClientStaleness).Before(newT) {
		up.MustRefresh = true
		err = tx.Commit()
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return nil, err
		}
		return up, nil
	}

	up.Categories, err = getUpdatedCategories(tx, tstr)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Feeds, err = getUpdatedFeeds(tx, tstr)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	up.Items, err = getUpdatedItems(tx, tstr)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, err
	}
	return up, err
}

func getUpdatedCategories(tx *sql.Tx, tstr string) ([]*structs.Category, error) {
	sql := "SELECT " + structs.CategorySelectColumns + `
		FROM categories INDEXED BY categories_commit_index
		WHERE commit_timestamp > ?
		ORDER BY categories.id ASC;`

	rows, err := tx.Query(sql, tstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats, err := structs.ScanCategories(rows)

	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debugf("getUpdatedCategories() retrieved %d feeds", len(cats))
	return cats, nil
}

func getUpdatedFeeds(tx *sql.Tx, tstr string) ([]*structs.Feed, error) {
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
		log.Error(err)
		return nil, err
	}
	log.Debugf("getUpdatedFeeds() retrieved %d feeds", len(feeds))
	return feeds, nil
}

func getUpdatedItems(tx *sql.Tx, tstr string) ([]*structs.Item, error) {

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
		log.Error(err)
		return nil, err
	}
	log.Debugf("getUpdatedItems() retrieved %d items", len(items))
	return items, nil
}
