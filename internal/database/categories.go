package database

import (
	"strings"

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

func getCategories(dot dbOrTx, ids []int64) ([]*structs.Category, error) {
	sql := entityBatchGetSQL(
		"categories", structs.CategorySelectColumns, len(ids))
	// Ugly
	binds := make([]interface{}, len(ids), len(ids))
	for i, v := range ids {
		binds[i] = v
	}

	rows, err := dot.Query(sql, binds...)
	if err != nil {
		return nil, err
	}
	return structs.ScanCategories(rows)
}

// ReorderCategories sets the sortPosition on all references categories
func (d *Database) ReorderCategories(ids []int64) (
	[]*structs.Category, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.checkClosed(); err != nil {
		log.Error(err)
		return nil, err
	}

	tx, err := d.db.Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer tx.Rollback()

	categories, err := getCategories(tx, ids)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	sortPositions := make(map[int64]int64)
	for i, id := range ids {
		sortPositions[id] = int64(i)
	}

	updateSQL := []string{}
	updateBinds := []interface{}{}

	for _, cat := range categories {
		update := structs.CategorySetSortPosition(cat, sortPositions[cat.ID()])
		sql, binds := update.Get()
		updateSQL = append(updateSQL, sql)
		updateBinds = append(updateBinds, binds...)
	}

	_, err = tx.Exec(strings.Join(updateSQL, "\n"), updateBinds...)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	updatedCategories, err := getCategories(tx, ids)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return updatedCategories, nil
}

// TODO -- as part of disabling a category set its name to the ID, which is not
// a legal name
