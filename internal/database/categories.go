package database

import (
	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

// InsertNewCategory creates and inserts a new category
func (d *Database) InsertNewCategory(name string, title string) (
	*structs.Category, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	log.Infof("Adding new category [%s, %s]", name, title)

	sql := `INSERT INTO categories(name, title) VALUES (?, ?);`
	res, err := d.db.Exec(sql, name, title)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return getCategory(d.db, id)
}

func getCategory(dot dbOrTx, id int64) (*structs.Category, error) {
	sql := entityGetSQL("categories", structs.CategorySelectColumns)

	return structs.ScanCategory(dot.QueryRow(sql, id))
}

// TODO -- as part of disabling a category set its name to the ID, which is not
// a legal name
