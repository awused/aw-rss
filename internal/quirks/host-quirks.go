package quirks

import "time"

var hostDelays = map[string]time.Duration{
	"mangadex.org": 5 * time.Second,
	"mangadex.cc":  5 * time.Second,
}

func MaybeGetHostDelay(host string) (time.Duration, bool) {
	dur, has := hostDelays[host]
	return dur, has
}
