package structs

import "time"

// Category is a grouping of feeds. Feeds can be in at most one category.
type Category struct {
	id int64
	// A short name for the category
	// Consists of only lowercase letters and hyphens
	name string
	// The display title of the category
	title string
	// Hidden categories are not visible in the nav bar.
	// Unread items for feeds in hidden categories are not counted.
	hidden string
	// Disabled categories are effectively deleted, but hang around so
	// that frontends are not inconvenienced. Feeds will destroy their
	// relationships with this category.
	// Any new categories will completely overwrite a disabled category.
	disabled        bool
	commitTimestamp time.Time
}
