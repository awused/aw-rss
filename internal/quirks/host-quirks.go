package quirks

import "time"

var hostDelays = map[string]time.Duration{
	"mangadex.org": 5 * time.Second,
	"mangadex.cc":  5 * time.Second,
}

// MaybeGetHostDelay returns the delay between fetches for a given host.
func MaybeGetHostDelay(host string) (time.Duration, bool) {
	dur, has := hostDelays[host]
	return dur, has
}
