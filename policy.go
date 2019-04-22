package resource

import (
	"net/http"
	"net/url"
	"time"
)

// HTTPUserAgent may be passed into getHTTPResult as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/resource"

// HTTPTimeout may be passed into getHTTPResult function as the default HTTP timeout parameter
const HTTPTimeout = time.Second * 90

// Policy indicates which actions we should perform
type Policy interface {
	HTTPUserAgent() string
	HTTPTimeout() time.Duration
	DetectRedirectsInHTMLContent(*url.URL) bool
	ParseMetaDataInHTMLContent(*url.URL) bool
	DownloadContent(*url.URL, *http.Response, Type) (bool, Attachment, []Issue)
}