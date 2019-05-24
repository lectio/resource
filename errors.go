package resource

import (
	"fmt"
	"golang.org/x/xerrors"
)

// xErrorsFrameCaller is passed into error functions to indicate the default stack frame
const xErrorsFrameCaller = 1

// Error is coded error for more granular tracking
type Error struct {
	URL string
	Message string
	Code    int
	Frame   xerrors.Frame
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e Error) FormatError(p xerrors.Printer) error {
	if len(e.URL) > 0 {
		p.Printf("LECTIORES-%d %s (%s)", e.Code, e.Message, e.URL)
	} else {
		p.Printf("LECTIORES-%d %s", e.Code, e.Message)
	}
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
	URL string
	HTTPStatusCode int
	Frame   xerrors.Frame
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e InvalidHTTPRespStatusCodeError) FormatError(p xerrors.Printer) error {
	p.Printf("LECTIORES-200 Expected HTTP Response Status Code 200, got %d (%s)", e.HTTPStatusCode, e.URL)
	e.Frame.Format(p)
	return nil
}

// Format provide backwards compatibility with pre-xerrors package
func (e InvalidHTTPRespStatusCodeError) Format(f fmt.State, c rune) {
	xerrors.FormatError(e, f, c)
}

// Format provide backwards compatibility with pre-xerrors package
func (e InvalidHTTPRespStatusCodeError) Error() string {
	return fmt.Sprint(e)
}
