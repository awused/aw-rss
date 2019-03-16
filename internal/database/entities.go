package database

import (
	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

// GetFeeds is a legacy method, to be removed
func (d *Database) GetFeeds(includeDisabled bool) ([]*structs.Feed, error) {
	glog.V(5).Info("GetFeeds() started")
	d.lock.RLock()
	defer d.lock.RUnlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	sql := "SELECT " + structs.FeedSelectColumns + " FROM feeds"
	if !includeDisabled {
		sql = sql + " WHERE disabled = 0"
	}

	rows, err := d.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds, err := structs.ScanFeeds(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("GetFeeds() retrieved %d feeds", len(feeds))
	glog.V(5).Info("GetFeeds() completed")
	return feeds, nil
}

// GetItems is a legacy method, to be removed
func (d *Database) GetItems(includeRead bool) ([]*structs.Item, error) {
	glog.V(5).Info("GetItems() started")
	d.lock.RLock()
	defer d.lock.RUnlock()

	if err := d.checkClosed(); err != nil {
		glog.Error(err)
		return nil, err
	}

	sql := "SELECT " + structs.ItemSelectColumns + " FROM items INNER JOIN feeds ON items.feedid = feeds.id WHERE feeds.disabled = 0"
	if !includeRead {
		sql = sql + " AND items.read = 0"
	}
	sql = sql + " ORDER BY timestamp DESC"

	rows, err := d.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := structs.ScanItems(rows)

	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.V(3).Infof("GetItems() retrieved %d items", len(items))
	glog.V(5).Info("GetItems() completed")
	return items, nil
}

func entityGetSQL(table string, columns string) string {
	return "SELECT " + columns + " FROM " + table + " WHERE id = ?;\n"
}

func insertSQL(table string, columns string, placeholders string) string {
	return "INSERT OR IGNORE INTO " + table + " (" + columns +
		") VALUES (" + placeholders + ");\n"
}

// Generic Get and Mutate methods
// TODO -- these can eventually become real generic  methods

func updateEntity(dot dbOrTx, ent structs.Entity) error {
	glog.V(2).Infof("Writing updated entity [%s]", ent)

	sql, binds := ent.UpdateSQL().Get()
	_, err := dot.Exec(sql, binds...)
	return err
}

// TODO -- All these methods are sloppy with how much work they do in the
// critical sections
