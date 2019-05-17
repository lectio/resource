package resource

import (
	"fmt"
	"golang.org/x/xerrors"
)

// xErrorsFrameCaller is passed into error functions to indicate the default stack frame
const xErrorsFrameCaller = 1

// Error is coded error for more granular tracking
type Error struct {
	Message string
	Code    int
	Frame   xerrors.Frame
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e Error) FormatError(p xerrors.Printer) error {
	p.Printf("LECTIORES-%d %s", e.Code, e.Message)
	e.Frame.Format(p)
	return nil
}

// Format provide backwards compatibility with pre-xerrors package
func (e Error) Format(f fmt.State, c rune) {
	xerrors.FormatError(e, f, c)
}

// Format provide backwards compatibility with pre-xerrors package
func (e Error) Error() string {
	return fmt.Sprint(e)
}

func targetURLIsBlankError(frame xerrors.Frame) *Error {
	return &Error{
		Message: "TargetURL is blank",
		Code:    50,
		Frame:   frame,
	}
}

func targetURLIsNilError(frame xerrors.Frame) *Error {
	return &Error{
		Message: "TargetURL is Nil",
		Code:    51,
		Frame:   frame,
	}
}

// InvalidHTTPRespStatusCodeError is thrown when the HTTP status code is not 200
type InvalidHTTPRespStatusCodeError struct {
	Error
	HTTPStatusCode int
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e InvalidHTTPRespStatusCodeError) FormatError(p xerrors.Printer) error {
	p.Printf("LECTIORES-%d:%d %s", e.Code, e.HTTPStatusCode, e.Message)
	e.Frame.Format(p)
	return nil
}

// const (
// 	TargetURLIsBlank              string = "RESOURCE_E-0050"
// 	TargetURLIsNil                string = "RESOURCE_E-0051"
// 	UnableToCreateHTTPRequest     string = "RESOURCE_E-0100"
// 	UnableToExecuteHTTPGETRequest string = "RESOURCE_E-0200"
// 	InvalidHTTPRespStatusCode     string = "RESOURCE_E-0300"
// 	UnableToParseHTTPBody         string = "RESOURCE_E-0400"

// 	UnableToInspectMediaTypeFromContentType string = "RESOURCE_E-0500"
// 	PolicyIsNil                             string = "RESOURCE_E-0600"
// 	CopyErrorDuringFileDownload             string = "RESOURCE_E-0700"
// 	MetaTagsNotAvailableInNonHTMLContent    string = "RESOURCE_W-0100"
// 	MetaTagsNotAvailableInUnparsedHTML      string = "RESOURCE_W-0101"
// 	UnableToInspectFileType                 string = "RESOURCE_S-0200"
// )
