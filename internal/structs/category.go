package structs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	disabled bool
	// Categories with nil sort positions are sorted by their IDs, after any
	// categories with non-nil sort positions
	sortPosition    *int64
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
		SortPosition    *int64 `json:"sortPosition,omitempty"`
		CommitTimestamp int64  `json:"commitTimestamp"`
	}{
		ID:              c.id,
		Disabled:        c.disabled,
		Name:            c.name,
		Title:           c.title,
		HiddenNav:       c.hiddenNav,
		HiddenMain:      c.hiddenMain,
		SortPosition:    c.sortPosition,
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
categories.sort_position,
categories.commit_timestamp`

func scanCategory(c *Category) []interface{} {
	return []interface{}{
		&c.id,
		&c.disabled,
		&c.name,
		&c.title,
		&c.hiddenNav,
		&c.hiddenMain,
		&c.sortPosition,
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

const categoryUpdateSQL string = `
UPDATE
	categories
SET
	disabled = ?,
	name = ?,
	title = ?,
	hidden_nav = ?,
	hidden_main = ?,
	sort_position = ?,
	commit_timestamp = CURRENT_TIMESTAMP
WHERE
	id = ?;`

func (c *Category) update() EntityUpdate {
	return EntityUpdate{
		c,
		false,
		categoryUpdateSQL,
		[]interface{}{
			c.disabled,
			c.name,
			c.title,
			c.hiddenNav,
			c.hiddenMain,
			c.sortPosition,
			c.id}}
}

// ID gets the ID
func (c *Category) ID() int64 { return c.id }

// CategorySetSortPosition mutates a category to change the sort position.
func CategorySetSortPosition(c *Category, sortPos int64) EntityUpdate {
	newC := *c

	if c.sortPosition != nil && sortPos == *c.sortPosition {
		return noopEntityUpdate(&newC)
	}

	newC.sortPosition = &sortPos
	return newC.update()
}

// CategoryEdit represents new values for a category from a user edit.
type CategoryEdit struct {
	// Categories can never be un-disabled
	Disabled   bool    `json:"disabled"`
	Name       *string `json:"name"`
	Title      *string `json:"title"`
	HiddenNav  *bool   `json:"hiddenNav"`
	HiddenMain *bool   `json:"hiddenMain"`
}

// CategoryNameRE is the regular expression matching all valid category names
var CategoryNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)

// CategoryApplyEdit returns a mutation function that applies the given
// CategoryEdit to a category after validating it.
func CategoryApplyEdit(edit CategoryEdit) (
	func(*Category) EntityUpdate, error) {
	if !edit.Disabled &&
		edit.Name != nil &&
		!CategoryNameRE.MatchString(*edit.Name) {

		m := "Tried to change category to invalid name [" + *edit.Name + "]"
		return nil, errors.New(m)
	}

	return func(c *Category) EntityUpdate {
		noop := true
		newC := *c

		if newC.disabled {
			return noopEntityUpdate(&newC)
		}

		if edit.Disabled {
			newC.disabled = true
			newC.name = strconv.FormatInt(newC.id, 10)
			return newC.update()
		}

		if edit.Name != nil && *edit.Name != c.name {
			noop = false
			newC.name = *edit.Name
		}

		if edit.Title != nil && *edit.Title != c.title {
			noop = false
			newC.title = *edit.Title
		}

		if edit.HiddenNav != nil && *edit.HiddenNav != c.hiddenNav {
			noop = false
			newC.hiddenNav = *edit.HiddenNav
		}

		if edit.HiddenMain != nil && *edit.HiddenMain != c.hiddenMain {
			noop = false
			newC.hiddenMain = *edit.HiddenMain
		}

		if noop {
			return noopEntityUpdate(&newC)
		}

		return newC.update()
	}, nil
}
