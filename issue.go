package resource

import "fmt"

const (
	TargetURLIsBlank              string = "RESOURCE_E-0050"
	TargetURLIsNil                string = "RESOURCE_E-0051"
	UnableToCreateHTTPRequest     string = "RESOURCE_E-0100"
	UnableToExecuteHTTPGETRequest string = "RESOURCE_E-0200"
	InvalidHTTPRespStatusCode     string = "RESOURCE_E-0300"
	UnableToParseHTTPBody         string = "RESOURCE_E-0400"

	UnableToInspectMediaTypeFromContentType string = "RESOURCE_E-0500"
	PolicyIsNil                             string = "RESOURCE_E-0600"
	CopyErrorDuringFileDownload             string = "RESOURCE_E-0700"
	MetaTagsNotAvailableInNonHTMLContent    string = "RESOURCE_W-0100"
	MetaTagsNotAvailableInUnparsedHTML      string = "RESOURCE_W-0101"
	UnableToInspectFileType                 string = "RESOURCE_S-0200"
)

// Issue is a structured problem identification with context information
type Issue interface {
	IssueContext() interface{} // this will be the Link object plus location (item index, etc.), it's kept generic so it doesn't require package dependency
	IssueCode() string         // useful to uniquely identify a particular code
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
	code    string
	message string
	isError bool
}

func NewIssue(context string, code string, message string, isError bool) Issue {
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
	result.code = fmt.Sprintf("%s-HTTP-%d", InvalidHTTPRespStatusCode, httpRespStatusCode)
	result.message = message
	result.isError = isError
	return result
}

func (i issue) IssueContext() interface{} {
	return i.context
}

func (i issue) IssueCode() string {
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
