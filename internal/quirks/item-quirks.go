package quirks

import (
	"strings"

	"github.com/mmcdole/gofeed"
)

// GetItemURL returns the link for an item
func GetItemURL(f feed, gfi *gofeed.Item) string {
	if strings.HasPrefix(gfi.Link, "http://konachan.com") {
		return "https" + strings.TrimPrefix(gfi.Link, "http")
	}
	return gfi.Link
}
