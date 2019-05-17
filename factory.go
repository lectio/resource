package resource

import (
	"context"
	"fmt"
	"golang.org/x/xerrors"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Factory is a lifecycle manager for URL-based resources
type Factory interface {
	PageFromURL(ctx context.Context, origURLtext string, options ...interface{}) (Content, error)
}

// NewFactory creates a new thread-safe resource factory
func NewFactory(options ...interface{}) Factory {
	result := &factory{}
	result.initOptions(options...)
	return result
}

type detectRedirectsPolicy interface {
	DetectRedirectsInHTMLContent(context.Context, *url.URL) bool
}

type parseMetaDataInHTMLContentPolicy interface {
	ParseMetaDataInHTMLContent(context.Context, *url.URL) bool
}

type contentDownloader interface {
	DownloadContent(context.Context, *url.URL, *http.Response, Type) (bool, Attachment, error)
}

type contentDownloaderErrorPolicy interface {
	StopOnDownloadError(context.Context, *url.URL, Type, error) bool
}

type fileAttachmentCreator interface {
	CreateFile(context.Context, *url.URL, Type) (*os.File, error)
	AutoAssignExtension(context.Context, *url.URL, Type) bool
}

type httpClientProvider interface {
	HTTPClient(context.Context) *http.Client
}

type httpRequestPreparer interface {
	OnPrepareHTTPRequest(context.Context, *http.Client, *http.Request)
}

type warningTracker interface {
	OnWarning(ctx context.Context, code, message string)
}

type factory struct {
	clientProvider                   httpClientProvider
	provideClientFunc                func(ctx context.Context) *http.Client
	reqPreparer                      httpRequestPreparer
	prepReqFunc                      func(ctx context.Context, client *http.Client, req *http.Request)
	warningTracker                   warningTracker
	detectRedirectsPolicy            detectRedirectsPolicy
	parseMetaDataInHTMLContentPolicy parseMetaDataInHTMLContentPolicy
	contentDownloader                contentDownloader
	contentDownloaderErrorPolicy     contentDownloaderErrorPolicy
	fileAttachmentCreator            fileAttachmentCreator
}

func (f *factory) initOptions(options ...interface{}) {
	f.warningTracker = f // we implemented a default version

	for _, option := range options {
		if wt, ok := option.(warningTracker); ok {
			f.warningTracker = wt
		}
		if instance, ok := option.(httpClientProvider); ok {
			f.clientProvider = instance
		}
		if fn, ok := option.(func(ctx context.Context) *http.Client); ok {
			f.provideClientFunc = fn
		}
		if instance, ok := option.(httpRequestPreparer); ok {
			f.reqPreparer = instance
		}
		if fn, ok := option.(func(ctx context.Context, client *http.Client, req *http.Request)); ok {
			f.prepReqFunc = fn
		}
		if instance, ok := option.(detectRedirectsPolicy); ok {
			f.detectRedirectsPolicy = instance
		}
		if instance, ok := option.(parseMetaDataInHTMLContentPolicy); ok {
			f.parseMetaDataInHTMLContentPolicy = instance
		}
		if instance, ok := option.(contentDownloader); ok {
			f.contentDownloader = instance
		}
		if instance, ok := option.(contentDownloaderErrorPolicy); ok {
			f.contentDownloaderErrorPolicy = instance
		}
		if instance, ok := option.(fileAttachmentCreator); ok {
			f.fileAttachmentCreator = instance
		}
	}
}

func (f *factory) httpClient(ctx context.Context) *http.Client {
	if f.clientProvider != nil {
		return f.clientProvider.HTTPClient(ctx)
	}

	if f.provideClientFunc != nil {
		return f.provideClientFunc(ctx)
	}

	return &http.Client{
		Timeout: time.Second * 90,
	}
}

func (f *factory) prepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request) {
	if f.reqPreparer != nil {
		f.reqPreparer.OnPrepareHTTPRequest(ctx, client, req)
	}

	if f.prepReqFunc != nil {
		f.prepReqFunc(ctx, client, req)
	}
}

func (f *factory) detectRedirectsInHTMLContent(ctx context.Context, url *url.URL) bool {
	if f.detectRedirectsPolicy != nil {
		return f.detectRedirectsPolicy.DetectRedirectsInHTMLContent(ctx, url)
	}
	return true
}

func (f *factory) parseMetaDataInHTMLContent(ctx context.Context, url *url.URL) bool {
	if f.parseMetaDataInHTMLContentPolicy != nil {
		f.parseMetaDataInHTMLContentPolicy.ParseMetaDataInHTMLContent(ctx, url)
	}
	return true
}

// NewPageFromURL creates a content instance from the given URL and policy
func (f *factory) PageFromURL(ctx context.Context, origURLtext string, options ...interface{}) (Content, error) {
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
func (f *factory) pageFromHTTPResponse(ctx context.Context, url *url.URL, resp *http.Response, options ...interface{}) (Content, error) {
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

	if f.contentDownloader != nil {
		ok, attachment, err := f.contentDownloader.DownloadContent(ctx, url, resp, result.PageType)
		if err != nil {
			if f.contentDownloaderErrorPolicy != nil {
				if f.contentDownloaderErrorPolicy.StopOnDownloadError(ctx, url, result.PageType, err) {
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

func (f *factory) OnWarning(ctx context.Context, code string, message string) {
	// this is the default function if nothing else is provided in initOptions()
}
