package resource

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// metaRefreshContentRegEx is used to match the 'content' attribute in a tag like this:
//   <meta http-equiv="refresh" content="2;url=https://www.google.com">
var metaRefreshContentRegEx = regexp.MustCompile(`^(\d?)\s?;\s?url=(.*)$`)

// HTTPUserAgent may be passed into getHTTPResult as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/resource"

// HTTPTimeout may be passed into getHTTPResult function as the default HTTP timeout parameter
const HTTPTimeout = time.Second * 90

// Page manages the content of a URL target
type Page struct {
	TargetURL                    *url.URL               `json:"url"`
	PageType                     Type                   `json:"type"`
	HTMLParsed                   bool                   `json:"htmlParsed"`
	IsHTMLRedirect               bool                   `json:"isHTMLRedirect"`
	MetaRefreshTagContentURLText string                 `json:"metaRefreshTagContentURLText"` // if IsHTMLRedirect is true, then this is the value after url= in something like <meta http-equiv='refresh' content='delay;url='>
	MetaPropertyTags             map[string]interface{} `json:"metaPropertyTags"`             // if IsHTML() is true, a collection of all meta data like <meta property="og:site_name" content="Netspective" /> or <meta name="twitter:title" content="text" />
	AllIssues                    []Issue                `json:"issues"`
	DownloadedAttachment         Attachment             `json:"attachment"`
}

// NewPageFromURL creates a content instance from the given URL and policy
func NewPageFromURL(origURLtext string, policy Policy) (Content, Issue) {
	if len(origURLtext) == 0 {
		return nil, newIssue("BlankTargetURL", TargetURLIsBlank, "TargetURL is blank in resource.NewPageFromURL", true)
	}

	if policy == nil {
		return nil, newIssue(origURLtext, PolicyIsNil, "Policy is nil in resource.NewPageFromURL", true)
	}

	// Use the standard Go HTTP library method to retrieve the Content; the
	// default will automatically follow redirects (e.g. HTTP redirects)
	// TODO: Consider using [HTTP Cache](https://github.com/gregjones/httpcache)
	httpClient := http.Client{
		Timeout: policy.HTTPTimeout(),
	}
	req, reqErr := http.NewRequest(http.MethodGet, origURLtext, nil)
	if reqErr != nil {
		return nil, newIssue(origURLtext, UnableToCreateHTTPRequest, fmt.Sprintf("Unable to create HTTP request: %v", reqErr), true)
	}
	req.Header.Set("User-Agent", policy.HTTPUserAgent())
	resp, getErr := httpClient.Do(req)
	if getErr != nil {
		return nil, newIssue(origURLtext, UnableToExecuteHTTPGETRequest, fmt.Sprintf("Unable to execute HTTP GET request: %v", getErr), true)
	}

	if resp.StatusCode != 200 {
		return nil, newHTTPResponseIssue(origURLtext, resp.StatusCode, fmt.Sprintf("Invalid HTTP Response Status Code: %d", resp.StatusCode), true)
	}

	return NewPageFromHTTPResponse(resp.Request.URL, resp, policy), nil
}

// NewPageFromHTTPResponse will download and figure out what kind content we're dealing with
func NewPageFromHTTPResponse(url *url.URL, resp *http.Response, policy Policy) Content {
	result := new(Page)
	result.MetaPropertyTags = make(map[string]interface{})
	result.TargetURL = url
	if result.TargetURL == nil {
		result.AllIssues = append(result.AllIssues, newIssue("NilTargetURL", TargetURLIsNil, "TargetURL is nil in resource.NewPageFromHTTPResponse", true))
		return result
	}
	if policy == nil {
		result.AllIssues = append(result.AllIssues, newIssue(url.String(), PolicyIsNil, "Policy is nil in resource.NewPageFromHTTPResponse", true))
		return result
	}

	contentType := resp.Header.Get("Content-Type")
	if len(contentType) > 0 {
		var issue Issue
		result.PageType, issue = NewPageType(url, contentType)
		if issue != nil {
			result.AllIssues = append(result.AllIssues, issue)
			return result
		}
		if result.IsHTML() && (policy.DetectRedirectsInHTMLContent(url) || policy.ParseMetaDataInHTMLContent(url)) {
			result.parsePageMetaData(url, resp)
			result.HTMLParsed = true
			return result
		}
	}

	ok, attachment, issues := policy.DownloadContent(url, resp, result.PageType)
	if issues != nil {
		for _, issue := range issues {
			result.AllIssues = append(result.AllIssues, issue)
		}
	}
	if ok && attachment != nil {
		result.DownloadedAttachment = attachment
	}

	return result
}

func (p *Page) parsePageMetaData(url *url.URL, resp *http.Response) error {
	doc, parseError := html.Parse(resp.Body)
	if parseError != nil {
		p.AllIssues = append(p.AllIssues, newIssue(url.String(), UnableToParseHTTPBody, parseError.Error(), true))
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

// Issues contains the problems in this link plus satisfies the Link interface
func (p Page) Issues() Issues {
	return p
}

// ErrorsAndWarnings contains the problems in this link plus satisfies the Link.Issues interface
func (p Page) ErrorsAndWarnings() []Issue {
	return p.AllIssues
}

// IssueCounts returns the total, errors, and warnings counts
func (p Page) IssueCounts() (uint, uint, uint) {
	if p.AllIssues == nil {
		return 0, 0, 0
	}
	var errors, warnings uint
	for _, i := range p.AllIssues {
		if i.IsError() {
			errors++
		} else {
			warnings++
		}
	}
	return uint(len(p.AllIssues)), errors, warnings
}

// HandleIssues loops through each issue and calls a particular handler
func (p Page) HandleIssues(errorHandler func(Issue), warningHandler func(Issue)) {
	if p.AllIssues == nil {
		return
	}
	for _, i := range p.AllIssues {
		if i.IsError() && errorHandler != nil {
			errorHandler(i)
		}
		if i.IsWarning() && warningHandler != nil {
			warningHandler(i)
		}
	}
}

// IsValid returns true if there are no errors
func (p Page) IsValid() bool {
	return p.AllIssues == nil || len(p.AllIssues) == 0
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
func (p Page) MetaTags() (MetaTags, Issue) {
	if !p.IsHTML() {
		return nil, newIssue(p.TargetURLText(), MetaTagsNotAvailableInNonHTMLContent, "Meta tags not available in non-HTML content", false)
	}
	if !p.HTMLParsed {
		return nil, newIssue(p.TargetURLText(), MetaTagsNotAvailableInUnparsedHTML, "Meta tags not available in unparsed HTML (error or policy didn't request parsing)", false)
	}
	return p.MetaPropertyTags, nil
}

// MetaTag returns a specific parsed meta tag
func (p Page) MetaTag(key string) (interface{}, bool, Issue) {
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
