package resource

import (
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
}

func (suite ContentSuite) HTTPClient() *http.Client {
	return suite.httpClient
}

func (suite ContentSuite) PrepareRequest(client *http.Client, req *http.Request) {
	req.Header.Set("User-Agent", "github.com/lectio/resource/test")
}

func (suite ContentSuite) DetectRedirectsInHTMLContent(url *url.URL) bool {
	return true
}

func (suite ContentSuite) ParseMetaDataInHTMLContent(url *url.URL) bool {
	return true
}

// DownloadContent satisfies Policy method
func (suite *ContentSuite) DownloadContent(url *url.URL, resp *http.Response, typ Type) (bool, Attachment, []Issue) {
	return DownloadFile(suite, url, resp, typ)
}

// CreateFile satisfies FileAttachmentPolicy method
func (suite *ContentSuite) CreateFile(url *url.URL, t Type) (*os.File, Issue) {
	pathAndFileName := fmt.Sprintf("tempFile-%d", suite.fileNum)
	suite.fileNum++
	var issue Issue
	destFile, err := os.Create(pathAndFileName)
	if err != nil {
		issue = NewIssue(url.String(), "SUITE_E-0001", fmt.Sprintf("Unable to create file %q", pathAndFileName), true)
	}
	return destFile, issue
}

func (suite ContentSuite) AutoAssignExtension(url *url.URL, t Type) bool {
	return true
}

func (suite *ContentSuite) SetupSuite() {
	suite.httpClient = &http.Client{
		Timeout: HTTPTimeout,
	}
}

func (suite *ContentSuite) TearDownSuite() {
}

func (suite *ContentSuite) TestBlankTargetURL() {
	_, issue := NewPageFromURL("", suite)
	suite.NotNil(issue, "Should get an error")
	suite.Equal(TargetURLIsBlank, issue.IssueCode(), "Should get proper error code")
}

func (suite *ContentSuite) TestBadTargetURL() {
	_, issue := NewPageFromURL("https://t", suite)
	suite.NotNil(issue, "Should get an error")
	suite.Equal(UnableToExecuteHTTPGETRequest, issue.IssueCode(), "Should get proper error code")
}

func (suite *ContentSuite) TestGoodTargetURLBadDest() {
	_, issue := NewPageFromURL("https://t.co/fDxPF", suite)
	suite.NotNil(issue, "Should get an error")
	suite.Equal("RESOURCE_E-0300-HTTP-404", issue.IssueCode(), "Should get proper error code")
}

func (suite *ContentSuite) TestOpenGraphMetaTags() {
	page, issue := NewPageFromURL("http://bit.ly/lectio_harvester_resource_test01", suite)
	suite.Nil(issue, "Should not get an error")

	value, _, _ := page.MetaTag("og:site_name")
	suite.Equal(value, "Netspective")

	value, _, _ = page.MetaTag("og:title")
	suite.Equal(value, "Safety, privacy, and security focused technology consulting")

	value, _, _ = page.MetaTag("og:description")
	suite.Equal(value, "Software, technology, and management consulting focused on firms im pacted by FDA, ONC, NIST or other safety, privacy, and security regulations")
}

func (suite *ContentSuite) TestGoodURLWithContentBasedRedirect() {
	page, issue := NewPageFromURL("http://bit.ly/lectio_harvester_resource_test03", suite)
	suite.Nil(issue, "Should not get an error")

	isHTMLRedirect, htmlRedirectURLText := page.Redirect()
	suite.True(isHTMLRedirect, "There should have been an HTML redirect requested through <meta http-equiv='refresh' content='delay;url='>")
	suite.Equal(htmlRedirectURLText, "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
}

func (suite *ContentSuite) TestGoodURLWithAttachment() {
	page, issue := NewPageFromURL("http://ceur-ws.org/Vol-1401/paper-05.pdf", suite)
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
