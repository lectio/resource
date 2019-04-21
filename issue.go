package resource

import "fmt"

type IssueCode string

const (
	TargetURLIsBlank              IssueCode = "RESOURCE_E-0050"
	TargetURLIsNil                IssueCode = "RESOURCE_E-0051"
	UnableToCreateHTTPRequest     IssueCode = "RESOURCE_E-0100"
	UnableToExecuteHTTPGETRequest IssueCode = "RESOURCE_E-0200"
	InvalidHTTPRespStatusCode     IssueCode = "RESOURCE_E-0300"
	UnableToParseHTTPBody         IssueCode = "RESOURCE_E-0400"

	UnableToInspectMediaTypeFromContentType IssueCode = "RESOURCE_E-0500"
	PolicyIsNil                             IssueCode = "RESOURCE_E-0600"
	CopyErrorDuringFileDownload             IssueCode = "RESOURCE_E-0700"
	MetaTagsNotAvailableInNonHTMLContent    IssueCode = "RESOURCE_W-0100"
	MetaTagsNotAvailableInUnparsedHTML      IssueCode = "RESOURCE_W-0101"
	UnableToInspectFileType                 IssueCode = "RESOURCE_S-0200"
)

// Issue is a structured problem identification with context information
type Issue interface {
	IssueContext() interface{} // this will be the Link object plus location (item index, etc.), it's kept generic so it doesn't require package dependency
	IssueCode() IssueCode      // useful to uniquely identify a particular code
	Issue() string             // the

	IsError() bool   // this issue is an error
	IsWarning() bool // this issue is a warning
}

// Issues packages multiple issues into a container
type Issues interface {
	ErrorsAndWarnings() []Issue
	IssueCounts() (uint, uint, uint)
	HandleIssues(errorHandler func(Issue), warningHandler func(Issue))
}

type issue struct {
	context string
	code    IssueCode
	message string
	isError bool
}

func newIssue(context string, code IssueCode, message string, isError bool) Issue {
	result := new(issue)
	result.context = context
	result.code = code
	result.message = message
	result.isError = isError
	return result
}

func newHTTPResponseIssue(context string, httpRespStatusCode int, message string, isError bool) Issue {
	result := new(issue)
	result.context = context
	result.code = IssueCode(fmt.Sprintf("%s-HTTP-%d", InvalidHTTPRespStatusCode, httpRespStatusCode))
	result.message = message
	result.isError = isError
	return result
}

func (i issue) IssueContext() interface{} {
	return i.context
}

func (i issue) IssueCode() IssueCode {
	return i.code
}

func (i issue) Issue() string {
	return i.message
}

func (i issue) IsError() bool {
	return i.isError
}

func (i issue) IsWarning() bool {
	return !i.isError
}

// Error satisfies the Go error contract
func (i issue) Error() string {
	return i.message
}
