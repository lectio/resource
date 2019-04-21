package resource

import (
	"net/http"
	"net/url"
	"time"
)

// Policy indicates which actions we should perform
type Policy interface {
	HTTPUserAgent() string
	HTTPTimeout() time.Duration
	DetectRedirectsInHTMLContent(*url.URL) bool
	ParseMetaDataInHTMLContent(*url.URL) bool
	DownloadContent(*url.URL, *http.Response, Type) (bool, Attachment, []Issue)
}
