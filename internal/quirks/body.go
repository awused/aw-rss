package quirks

import (
	"strings"
)

// TODO -- move other quirks into this package

// HandleBodyQuirks operates on the body of the feed to correct issues
func HandleBodyQuirks(f Feed, body string) string {
	// NovelUpdates produces UTF-8 feeds but erroneously sets the encoding to
	// iso-8859-1
	if strings.HasPrefix(f.URL(), "https://www.novelupdates.com/") {
		return strings.TrimPrefix(
			body, `<?xml version="1.0" encoding="ISO-8859-1" ?>`)
	}

	return body
}
