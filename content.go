package resource

import (
	"net/url"
)

// MediaTypeParams contains what was parsed from MediaType
type MediaTypeParams map[string]string

// MetaTags contains HTML or other content meta tag properties
type MetaTags map[string]interface{}

// Content defines the target of a URL
type Content interface {
	URL() *url.URL
	Issues() Issues
	IsValid() bool
	Type() Type
	IsHTML() bool
	Redirect() (bool, string)
	MetaTags() (MetaTags, Issue)
	MetaTag(key string) (interface{}, bool, Issue)
	Attachment() Attachment
}

// Attachment is a single attachment to Content (perhaps PDF or other downloaded file)
type Attachment interface {
	Type() Type
	Issues() Issues
	IsValid() bool
	//Reader() io.Reader
}

// Type defines the kind of content
type Type interface {
	ContentType() string
	MediaType() string
	MediaTypeParams() MediaTypeParams
}
