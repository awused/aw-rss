package structs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// Category is a grouping of feeds. Feeds can be in at most one category.
// Categories do not affect the backend at all.
type Category struct {
	id int64
	// A short name for the category
	// Consists of only lowercase letters and hyphens
	// The frontend will do its best to redirect /:name to /category/:name
	name string
	// The display title of the category
	title string
	// Hide the category in the nav bar.
	hiddenNav bool
	// Hide the feeds and items in this category in the main view
	hiddenMain bool
	// Disabled categories are effectively deleted, but hang around so
	// that frontends are not inconvenienced. Feeds will destroy their
	// relationships with this category.
	// Any new categories will completely overwrite a disabled category.
	disabled        bool
	commitTimestamp time.Time
}

// MarshalJSON is used by the JSON marshaller
func (c *Category) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              int64  `json:"id"`
		Disabled        bool   `json:"disabled"`
		Name            string `json:"name"`
		Title           string `json:"title"`
		HiddenNav       bool   `json:"hiddenNav"`
		HiddenMain      bool   `json:"hiddenMain"`
		CommitTimestamp int64  `json:"commitTimestamp"`
	}{
		ID:              c.id,
		Disabled:        c.disabled,
		Name:            c.name,
		Title:           c.title,
		HiddenNav:       c.hiddenNav,
		HiddenMain:      c.hiddenMain,
		CommitTimestamp: c.commitTimestamp.Unix(),
	})
}

// CategorySelectColumns is used by the database when reading categories
const CategorySelectColumns string = `
categories.id,
categories.disabled,
categories.name,
categories.title,
categories.hidden_nav,
categories.hidden_main,
categories.commit_timestamp`

func scanCategory(c *Category) []interface{} {
	return []interface{}{
		&c.id,
		&c.disabled,
		&c.name,
		&c.title,
		&c.hiddenNav,
		&c.hiddenMain,
		&c.commitTimestamp}
}

// ScanCategory converts one row into a category
func ScanCategory(row *sql.Row) (*Category, error) {
	var cat Category
	err := row.Scan(scanCategory(&cat)...)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &cat, nil
}

// ScanCategories converts multiple rows into categories
func ScanCategories(rows *sql.Rows) ([]*Category, error) {
	cats := []*Category{}
	for rows.Next() {
		var cat Category
		err := rows.Scan(scanCategory(&cat)...)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		cats = append(cats, &cat)
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	return cats, nil
}

func (c *Category) String() string {
	str := fmt.Sprintf("Category %d: %s (%s)", c.id, c.name, c.title)
	if c.disabled {
		str += " disabled"
	} else {
		if c.hiddenNav {
			str += ", hidden_nav"
		}
		if c.hiddenMain {
			str += ", hidden_main"
		}
	}
	return str
}
