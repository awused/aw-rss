package database

import (
	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

// AddCategoryRequest contains the information needed to create a category
type AddCategoryRequest struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	HiddenNav  bool   `json:"hiddenNav"`
	HiddenMain bool   `json:"hiddenMain"`
}

// InsertNewCategory creates and inserts a new category
func (d *Database) InsertNewCategory(req AddCategoryRequest) (
	*structs.Category, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	log.Infof("Adding new category [%s, %s]", req.Name, req.Title)

	sql := `
			INSERT INTO
					categories(name, title, hidden_nav, hidden_main)
			VALUES (?, ?, ?, ?);`
	res, err := d.db.Exec(sql, req.Name, req.Title, req.HiddenNav, req.HiddenMain)
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
