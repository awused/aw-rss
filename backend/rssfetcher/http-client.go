package rssfetcher

/**
 * Implements an http client that helps massage slightly invalid XML into a usable state.
 */

import (
	"github.com/golang/glog"
	"io"
	"net/http"
)

var invalid_bytes map[byte]bool = map[byte]bool{
	0x02: true,
	0x03: true,
	0x0c: true,
	0x0f: true,
	0x1d: true,
}

const replacement_byte byte = 0x20

type filteredReadCloser struct {
	io.ReadCloser
	base    io.ReadCloser
	feedUrl string
}

func (this filteredReadCloser) Read(p []byte) (n int, err error) {
	n, err = this.base.Read(p)

	// Replace known invalid characters in the input with safe replacement characters
	for i := 0; i < n; i++ {
		if invalid_bytes[p[i]] {
			glog.V(1).Infof("Found invalid byte [%d] in response for %s", p[i], this.feedUrl)
			glog.V(7).Infof("Found invalid byte in [%s]", p[:n])
			p[i] = replacement_byte
		}
	}

	return
}

func (this filteredReadCloser) Close() error {
	return this.base.Close()
}

type filteredRoundTripper struct {
	http.RoundTripper
}

func (this filteredRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return res, err
	}
	res.Body = &filteredReadCloser{base: res.Body, feedUrl: req.URL.String()}
	return res, err
}
