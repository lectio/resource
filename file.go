package resource

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	filetype "github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

// FileAttachmentPolicy allows files of different types to be created
type FileAttachmentPolicy interface {
	CreateFile(*url.URL, Type) (*os.File, Issue)
	AutoAssignExtension(*url.URL, Type) bool
}

// FileAttachment manages any content that was downloaded for further inspection
type FileAttachment struct {
	ContentType Type       `json:"type"`
	TargetURL   *url.URL   `json:"url"`
	DestPath    string     `json:"destPath"`
	FileType    types.Type `json:"fileType"`
	AllIssues   []Issue    `json:"issues"`
}

// URL is the resource locator for this content
func (a FileAttachment) URL() *url.URL {
	return a.TargetURL
}

// Issues contains the problems in this link plus satisfies the Link interface
func (a FileAttachment) Issues() Issues {
	return a
}

// ErrorsAndWarnings contains the problems in this link plus satisfies the Link.Issues interface
func (a FileAttachment) ErrorsAndWarnings() []Issue {
	return a.AllIssues
}

// IssueCounts returns the total, errors, and warnings counts
func (a FileAttachment) IssueCounts() (uint, uint, uint) {
	if a.AllIssues == nil {
		return 0, 0, 0
	}
	var errors, warnings uint
	for _, i := range a.AllIssues {
		if i.IsError() {
			errors++
		} else {
			warnings++
		}
	}
	return uint(len(a.AllIssues)), errors, warnings
}

// HandleIssues loops through each issue and calls a particular handler
func (a FileAttachment) HandleIssues(errorHandler func(Issue), warningHandler func(Issue)) {
	if a.AllIssues == nil {
		return
	}
	for _, i := range a.AllIssues {
		if i.IsError() && errorHandler != nil {
			errorHandler(i)
		}
		if i.IsWarning() && warningHandler != nil {
			warningHandler(i)
		}
	}
}

// IsValid returns true if there are no errors
func (a FileAttachment) IsValid() bool {
	return a.AllIssues == nil || len(a.AllIssues) == 0
}

// Type returns the results of content inspection
func (a FileAttachment) Type() Type {
	return a.ContentType
}

// Delete removes the file that was downloaded
func (a *FileAttachment) Delete() {
	os.Remove(a.DestPath)
}

// DownloadFile will download the URL as an "attachment" to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func DownloadFile(policy FileAttachmentPolicy, url *url.URL, resp *http.Response, typ Type) (bool, Attachment, []Issue) {
	var issues []Issue
	if url == nil {
		issues = append(issues, newIssue("NilTargetURL", TargetURLIsNil, "url is nil in resource.DownloadFile", true))
		return false, nil, issues
	}
	if resp == nil {
		issues = append(issues, newIssue("NilHTTPResponse", TargetURLIsNil, "http.Response is nil in resource.DownloadFile", true))
		return false, nil, issues
	}

	result := new(FileAttachment)
	result.TargetURL = url
	result.ContentType = typ

	if policy == nil {
		result.AllIssues = append(result.AllIssues, newIssue(url.String(), PolicyIsNil, "Policy is nil in resource.DownloadFile", true))
		return false, result, result.AllIssues
	}

	destFile, issue := policy.CreateFile(url, typ)
	if issue != nil {
		result.AllIssues = append(result.AllIssues, issue)
		return false, result, result.AllIssues
	}

	defer destFile.Close()
	defer resp.Body.Close()
	result.DestPath = destFile.Name()
	_, err := io.Copy(destFile, resp.Body)
	if err != nil {
		result.AllIssues = append(result.AllIssues, newIssue(url.String(), CopyErrorDuringFileDownload, err.Error(), true))
		return false, result, result.AllIssues
	}
	destFile.Close()

	if policy.AutoAssignExtension(url, typ) {
		// Open the just-downloaded file again since it was closed already
		file, err := os.Open(result.DestPath)
		if err != nil {
			result.AllIssues = append(result.AllIssues, newIssue(url.String(), UnableToInspectFileType, err.Error(), true))
			return false, result, result.AllIssues
		}

		// We only have to pass the file header = first 261 bytes
		head := make([]byte, 261)
		file.Read(head)
		file.Close()

		fileType, fileTypeError := filetype.Match(head)
		if fileTypeError == nil {
			// change the extension so that it matches the file type we found
			result.FileType = fileType
			currentPath := result.DestPath
			currentExtension := path.Ext(currentPath)
			newPath := currentPath[0:len(currentPath)-len(currentExtension)] + "." + fileType.Extension
			os.Rename(currentPath, newPath)
			result.DestPath = newPath
		}
	}

	return true, result, result.AllIssues
}
