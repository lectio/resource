package resource

import (
	"context"
	"fmt"
	"github.com/spf13/afero"
	"golang.org/x/xerrors"
	"io"
	"net/http"
	"net/url"
	"path"

	filetype "github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

// FileAttachmentCreator allows files of different types to be created
type FileAttachmentCreator interface {
	CreateFile(context.Context, *url.URL, Type) (afero.Fs, afero.File, error)
	AutoAssignExtension(context.Context, *url.URL, Type) bool
}

// FileAttachment manages any content that was downloaded for further inspection
type FileAttachment struct {
	ContentType Type       `json:"type"`
	TargetURL   *url.URL   `json:"url"`
	DestFS      afero.Fs   `json:"destFS"`
	DestPath    string     `json:"destPath"`
	FileType    types.Type `json:"fileType"`
	Valid       bool       `json:"valid"`
}

// URL is the resource locator for this content
func (a FileAttachment) URL() *url.URL {
	return a.TargetURL
}

// IsValid returns true if there are no errors
func (a FileAttachment) IsValid() bool {
	return a.Valid
}

// Type returns the results of content inspection
func (a FileAttachment) Type() Type {
	return a.ContentType
}

// Delete removes the file that was downloaded
func (a *FileAttachment) Delete() {
	a.DestFS.Remove(a.DestPath)
}

// DownloadFileFromHTTPResp will download the URL as an "attachment" to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func DownloadFileFromHTTPResp(ctx context.Context, creator FileAttachmentCreator, url *url.URL, resp *http.Response, typ Type, options ...interface{}) (bool, Attachment, error) {
	if url == nil {
		return false, nil, fmt.Errorf("url is nil in resource.DownloadFile")
	}
	if resp == nil {
		return false, nil, fmt.Errorf("http.Response is nil in resource.DownloadFile")
	}

	result := new(FileAttachment)
	result.TargetURL = url
	result.ContentType = typ

	if creator == nil {
		return false, result, fmt.Errorf("FileAttachmentCreator is nil in resource.DownloadFile")
	}

	fs, destFile, err := creator.CreateFile(ctx, url, typ)
	if err != nil {
		return false, result, xerrors.Errorf("Unable to create file in resource.DownloadFile: %w", err)
	}

	defer destFile.Close()
	defer resp.Body.Close()
	result.DestFS = fs
	result.DestPath = destFile.Name()
	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return false, result, xerrors.Errorf("Copy error during file download in resource.DownloadFile: %w", err)
	}
	destFile.Close()

	if creator.AutoAssignExtension(ctx, url, typ) {
		// Open the just-downloaded file again since it was closed already
		file, err := fs.Open(result.DestPath)
		if err != nil {
			return false, result, xerrors.Errorf("Unable to inspect file type in resource.DownloadFile: %w", err)
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
			fs.Rename(currentPath, newPath)
			result.DestPath = newPath
		}
	}

	result.Valid = true
	return true, result, nil
}
