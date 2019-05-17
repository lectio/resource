package resource

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ContentSuite struct {
	suite.Suite
	fileNum    int
	httpClient *http.Client
	factory    Factory
}

func (suite ContentSuite) HTTPClient(ctx context.Context) *http.Client {
	return suite.httpClient
}

func (suite ContentSuite) PrepareRequest(ctx context.Context, client *http.Client, req *http.Request) {
	req.Header.Set("User-Agent", "github.com/lectio/resource/test")
}

func (suite ContentSuite) DetectRedirectsInHTMLContent(ctx context.Context, url *url.URL) bool {
	return true
}

func (suite ContentSuite) ParseMetaDataInHTMLContent(ctx context.Context, url *url.URL) bool {
	return true
}

// DownloadContent satisfies Policy method
func (suite *ContentSuite) DownloadContent(ctx context.Context, url *url.URL, resp *http.Response, typ Type) (bool, Attachment, error) {
	return DownloadFile(ctx, suite, url, resp, typ)
}

// CreateFile satisfies FileAttachmentPolicy method
func (suite *ContentSuite) CreateFile(ctx context.Context, url *url.URL, t Type) (*os.File, error) {
	pathAndFileName := fmt.Sprintf("tempFile-%d", suite.fileNum)
	suite.fileNum++
	destFile, err := os.Create(pathAndFileName)
	if err != nil {
		return nil, err
	}
	return destFile, nil
}

func (suite ContentSuite) AutoAssignExtension(ctx context.Context, url *url.URL, t Type) bool {
	return true
}

func (suite *ContentSuite) SetupSuite() {
	suite.httpClient = &http.Client{}
	suite.factory = NewFactory(suite)
}

func (suite *ContentSuite) TearDownSuite() {
}

func (suite *ContentSuite) TestBlankTargetURL() {
	ctx := context.Background()
	_, err := suite.factory.PageFromURL(ctx, "")
	suite.NotNil(err, "Should get an error")
}

func (suite *ContentSuite) TestBadTargetURL() {
	ctx := context.Background()
	_, issue := suite.factory.PageFromURL(ctx, "https://t")
	suite.NotNil(issue, "Should get an error")
}

func (suite *ContentSuite) TestGoodTargetURLBadDest() {
	ctx := context.Background()
	_, issue := suite.factory.PageFromURL(ctx, "https://t.co/fDxPF")
	suite.NotNil(issue, "Should get an error")
}

func (suite *ContentSuite) TestOpenGraphMetaTags() {
	ctx := context.Background()
	page, issue := suite.factory.PageFromURL(ctx, "http://bit.ly/lectio_harvester_resource_test01")
	suite.Nil(issue, "Should not get an error")

	value, _, _ := page.MetaTag("og:site_name")
	suite.Equal(value, "Netspective")

	value, _, _ = page.MetaTag("og:title")
	suite.Equal(value, "Safety, privacy, and security focused technology consulting")

	value, _, _ = page.MetaTag("og:description")
	suite.Equal(value, "Software, technology, and management consulting focused on firms im pacted by FDA, ONC, NIST or other safety, privacy, and security regulations")
}

func (suite *ContentSuite) TestGoodURLWithContentBasedRedirect() {
	ctx := context.Background()

	page, issue := suite.factory.PageFromURL(ctx, "http://bit.ly/lectio_harvester_resource_test03", suite)
	suite.Nil(issue, "Should not get an error")

	isHTMLRedirect, htmlRedirectURLText := page.Redirect()
	suite.True(isHTMLRedirect, "There should have been an HTML redirect requested through <meta http-equiv='refresh' content='delay;url='>")
	suite.Equal(htmlRedirectURLText, "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
}

func (suite *ContentSuite) TestResolvedURLNotCleaned() {
	ctx := context.Background()

	page, issue := suite.factory.PageFromURL(ctx, "https://t.co/ELrZmo81wI", suite)
	suite.Nil(issue, "Should not see error")
	suite.NotNil(page, "The destination content should be available")
	suite.True(page.IsValid(), "The destination content should be valid")
	suite.True(page.IsHTML(), "The destination content should be HTML")
}

func (suite *ContentSuite) TestGoodURLWithAttachment() {
	ctx := context.Background()

	page, issue := suite.factory.PageFromURL(ctx, "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	attachment := page.Attachment()
	suite.Nil(issue, "Should not get an error")
	suite.NotNil(attachment, "Should have an attachment")
	suite.True(attachment.IsValid(), "First attachment should be valid")
	suite.Equal("application/pdf", attachment.Type().ContentType(), "Should be a PDF file")
	suite.Equal("application/pdf", attachment.Type().MediaType(), "Should be a PDF file")

	fa, ok := attachment.(*FileAttachment)
	suite.True(ok, "Attachment should be a FileAttachment type")
	if ok {
		fileExists := false
		if _, err := os.Stat(fa.DestPath); err == nil {
			fileExists = true
		}
		suite.True(fileExists, "File %s should exist", fa.DestPath)
		suite.Equal(path.Ext(fa.DestPath), ".pdf", "File's extension should be .pdf")

		fa.Delete()
		if _, err := os.Stat(fa.DestPath); err == nil {
			fileExists = true
		}
		suite.True(fileExists, "File %s should not exist", fa.DestPath)
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ContentSuite))
}
