package resource

import (
	"context"
	"fmt"
	"golang.org/x/xerrors"
	"net/http"
	"net/url"
	"time"
)

// Factory is a lifecycle manager for URL-based resources
type Factory interface {
	PageFromURL(ctx context.Context, origURLtext string, options ...interface{}) (Content, error)
}

// NewFactory creates a new thread-safe resource factory
func NewFactory(options ...interface{}) *DefaultFactory {
	f := &DefaultFactory{}

	f.WarningTracker = f // we implemented a default version

	f.initOptions(options...)
	return f
}

type DetectRedirectsPolicy interface {
	DetectRedirectsInHTMLContent(context.Context, *url.URL) bool
}

type ParseMetaDataInHTMLContentPolicy interface {
	ParseMetaDataInHTMLContent(context.Context, *url.URL) bool
}

type ContentDownloader interface {
	DownloadContent(context.Context, *url.URL, *http.Response, Type) (bool, Attachment, error)
}

type ContentDownloaderErrorPolicy interface {
	StopOnDownloadError(context.Context, *url.URL, Type, error) bool
}

type HTTPClientProvider interface {
	HTTPClient(context.Context) *http.Client
}

type HTTPRequestPreparer interface {
	OnPrepareHTTPRequest(context.Context, *http.Client, *http.Request)
}

type WarningTracker interface {
	OnWarning(ctx context.Context, code, message string)
}

type DefaultFactory struct {
	ClientProvider                   HTTPClientProvider
	ProvideClientFunc                func(ctx context.Context) *http.Client
	ReqPreparer                      HTTPRequestPreparer
	PrepReqFunc                      func(ctx context.Context, client *http.Client, req *http.Request)
	WarningTracker                   WarningTracker
	DetectRedirectsPolicy            DetectRedirectsPolicy
	ParseMetaDataInHTMLContentPolicy ParseMetaDataInHTMLContentPolicy
	ContentDownloader                ContentDownloader
	ContentDownloaderErrorPolicy     ContentDownloaderErrorPolicy
	FileAttachmentCreator            FileAttachmentCreator
}

func (f *DefaultFactory) initOptions(options ...interface{}) {
	for _, option := range options {
		if wt, ok := option.(WarningTracker); ok {
			f.WarningTracker = wt
		}
		if instance, ok := option.(HTTPClientProvider); ok {
			f.ClientProvider = instance
		}
		if fn, ok := option.(func(ctx context.Context) *http.Client); ok {
			f.ProvideClientFunc = fn
		}
		if instance, ok := option.(HTTPRequestPreparer); ok {
			f.ReqPreparer = instance
		}
		if fn, ok := option.(func(ctx context.Context, client *http.Client, req *http.Request)); ok {
			f.PrepReqFunc = fn
		}
		if instance, ok := option.(DetectRedirectsPolicy); ok {
			f.DetectRedirectsPolicy = instance
		}
		if instance, ok := option.(ParseMetaDataInHTMLContentPolicy); ok {
			f.ParseMetaDataInHTMLContentPolicy = instance
		}
		if instance, ok := option.(ContentDownloader); ok {
			f.ContentDownloader = instance
		}
		if instance, ok := option.(ContentDownloaderErrorPolicy); ok {
			f.ContentDownloaderErrorPolicy = instance
		}
		if instance, ok := option.(FileAttachmentCreator); ok {
			f.FileAttachmentCreator = instance
		}
	}
}

func (f *DefaultFactory) httpClient(ctx context.Context) *http.Client {
	if f.ClientProvider != nil {
		return f.ClientProvider.HTTPClient(ctx)
	}

	if f.ProvideClientFunc != nil {
		return f.ProvideClientFunc(ctx)
	}

	return &http.Client{
		Timeout: time.Second * 90,
	}
}

func (f *DefaultFactory) prepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request) {
	if f.ReqPreparer != nil {
		f.ReqPreparer.OnPrepareHTTPRequest(ctx, client, req)
	}

	if f.PrepReqFunc != nil {
		f.PrepReqFunc(ctx, client, req)
	}
}

func (f *DefaultFactory) detectRedirectsInHTMLContent(ctx context.Context, url *url.URL) bool {
	if f.DetectRedirectsPolicy != nil {
		return f.DetectRedirectsPolicy.DetectRedirectsInHTMLContent(ctx, url)
	}
	return true
}

func (f *DefaultFactory) parseMetaDataInHTMLContent(ctx context.Context, url *url.URL) bool {
	if f.ParseMetaDataInHTMLContentPolicy != nil {
		f.ParseMetaDataInHTMLContentPolicy.ParseMetaDataInHTMLContent(ctx, url)
	}
	return true
}

// PageFromURL creates a content instance from the given URL and policy
func (f *DefaultFactory) PageFromURL(ctx context.Context, origURLtext string, options ...interface{}) (Content, error) {
	if len(origURLtext) == 0 {
		return nil, targetURLIsBlankError(xerrors.Caller(xErrorsFrameCaller))
	}

	// Use the standard Go HTTP library method to retrieve the Content; the default will automatically follow redirects (e.g. HTTP redirects)
	httpClient := f.httpClient(ctx)
	req, reqErr := http.NewRequest(http.MethodGet, origURLtext, nil)
	if reqErr != nil {
		return nil, xerrors.Errorf("Unable to create HTTP request: %w", reqErr)
	}
	f.prepareHTTPRequest(ctx, httpClient, req)
	resp, getErr := httpClient.Do(req)
	if getErr != nil {
		return nil, xerrors.Errorf("Unable to execute HTTP GET request: %w", getErr)
	}

	if resp.StatusCode != 200 {
		err := &InvalidHTTPRespStatusCodeError{
			Error{Message: fmt.Sprintf("Invalid HTTP Response Status Code: %d", resp.StatusCode),
				Code:  200,
				Frame: xerrors.Caller(xErrorsFrameCaller)},
			resp.StatusCode,
		}
		return nil, xerrors.Errorf("Unexpected HTTP Response Status Code: %w", err)
	}

	return f.pageFromHTTPResponse(ctx, resp.Request.URL, resp, options...)
}

// NewPageFromHTTPResponse will download and figure out what kind content we're dealing with
func (f *DefaultFactory) pageFromHTTPResponse(ctx context.Context, url *url.URL, resp *http.Response, options ...interface{}) (Content, error) {
	result := new(Page)
	result.MetaPropertyTags = make(map[string]interface{})
	result.TargetURL = url

	contentType := resp.Header.Get("Content-Type")
	if len(contentType) > 0 {
		var err error
		result.PageType, err = NewPageType(url, contentType)
		if err != nil {
			return result, err
		}
		if result.IsHTML() && (f.detectRedirectsInHTMLContent(ctx, url) || f.parseMetaDataInHTMLContent(ctx, url)) {
			result.parsePageMetaData(ctx, url, resp)
			result.HTMLParsed = true
			return result, nil
		}
	}

	if f.ContentDownloader != nil {
		ok, attachment, err := f.ContentDownloader.DownloadContent(ctx, url, resp, result.PageType)
		if err != nil {
			if f.ContentDownloaderErrorPolicy != nil {
				if f.ContentDownloaderErrorPolicy.StopOnDownloadError(ctx, url, result.PageType, err) {
					return result, err
				}
			}
		} else if ok && attachment != nil {
			result.DownloadedAttachment = attachment
		}
	}

	result.valid = true
	return result, nil
}

func (f *DefaultFactory) OnWarning(ctx context.Context, code string, message string) {
	// this is the default function if nothing else is provided in initOptions()
}
