package resource

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// metaRefreshContentRegEx is used to match the 'content' attribute in a tag like this:
//   <meta http-equiv="refresh" content="2;url=https://www.google.com">
var metaRefreshContentRegEx = regexp.MustCompile(`^(\d?)\s?;\s?url=(.*)$`)

// Page manages the content of a URL target
type Page struct {
	TargetURL                    *url.URL               `json:"url"`
	PageType                     Type                   `json:"type"`
	HTMLParsed                   bool                   `json:"htmlParsed"`
	IsHTMLRedirect               bool                   `json:"isHTMLRedirect"`
	MetaRefreshTagContentURLText string                 `json:"metaRefreshTagContentURLText"` // if IsHTMLRedirect is true, then this is the value after url= in something like <meta http-equiv='refresh' content='delay;url='>
	MetaPropertyTags             map[string]interface{} `json:"metaPropertyTags"`             // if IsHTML() is true, a collection of all meta data like <meta property="og:site_name" content="Netspective" /> or <meta name="twitter:title" content="text" />
	DownloadedAttachment         Attachment             `json:"attachment"`

	valid bool
}

func (p *Page) parsePageMetaData(ctx context.Context, url *url.URL, resp *http.Response) error {
	doc, parseError := html.Parse(resp.Body)
	if parseError != nil {
		return parseError
	}
	defer resp.Body.Close()

	var inHead bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "head") {
			inHead = true
		}
		if inHead && n.Type == html.ElementNode && strings.EqualFold(n.Data, "meta") {
			for _, attr := range n.Attr {
				if strings.EqualFold(attr.Key, "http-equiv") && strings.EqualFold(strings.TrimSpace(attr.Val), "refresh") {
					for _, attr := range n.Attr {
						if strings.EqualFold(attr.Key, "content") {
							contentValue := strings.TrimSpace(attr.Val)
							parts := metaRefreshContentRegEx.FindStringSubmatch(contentValue)
							if parts != nil && len(parts) == 3 {
								// the first part is the entire match
								// the second and third parts are the delay and URL
								// See for explanation: http://redirectdetective.com/redirection-types.html
								p.IsHTMLRedirect = true
								p.MetaRefreshTagContentURLText = parts[2]
							}
						}
					}
				}
				if strings.EqualFold(attr.Key, "property") || strings.EqualFold(attr.Key, "name") {
					propertyName := attr.Val
					for _, attr := range n.Attr {
						if strings.EqualFold(attr.Key, "content") {
							p.MetaPropertyTags[propertyName] = attr.Val
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nil
}

// URL is the resource locator for this content
func (p Page) URL() *url.URL {
	return p.TargetURL
}

// IsValid returns true if there are no errors
func (p Page) IsValid() bool {
	return p.valid
}

// Type returns results of content inspection
func (p Page) Type() Type {
	return p.PageType
}

// IsHTML returns true if this is HTML content
func (p Page) IsHTML() bool {
	return p.Type().MediaType() == "text/html"
}

// TargetURLText returns the text version of the TargetURL
func (p Page) TargetURLText() string {
	if p.TargetURL == nil {
		return "NilTargetURL"
	}
	return p.TargetURL.String()
}

// MetaTags returns tags that were parsed
func (p Page) MetaTags() (MetaTags, error) {
	if !p.IsHTML() {
		return nil, fmt.Errorf("Meta tags not available in non-HTML content")
	}
	if !p.HTMLParsed {
		return nil, fmt.Errorf("Meta tags not available in unparsed HTML (error or policy didn't request parsing)")
	}
	return p.MetaPropertyTags, nil
}

// MetaTag returns a specific parsed meta tag
func (p Page) MetaTag(key string) (interface{}, bool, error) {
	tags, issue := p.MetaTags()
	if issue != nil {
		return nil, false, issue
	}
	result, ok := tags[key]
	return result, ok, nil
}

// Redirect returns true if redirect was requested through via <meta http-equiv='refresh' content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (p Page) Redirect() (bool, string) {
	return p.IsHTMLRedirect, p.MetaRefreshTagContentURLText
}

// Attachment returns the any downloaded file
func (p Page) Attachment() Attachment {
	return p.DownloadedAttachment
}
