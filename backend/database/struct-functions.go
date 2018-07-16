package database

import (
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
