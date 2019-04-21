package resource

import (
	"mime"
	"net/url"
)

// PageType encapsulates the various descriptions of the kind of page / content
type PageType struct {
	ContType      string          `json:"contentType"`
	MedType       string          `json:"mediaType"`
	MedTypeParams MediaTypeParams `json:"mediaTypeParams"`
}

func NewPageType(url *url.URL, contentType string) (Type, Issue) {
	result := new(PageType)
	result.ContType = contentType
	var mediaTypeError error
	result.MedType, result.MedTypeParams, mediaTypeError = mime.ParseMediaType(contentType)
	if mediaTypeError != nil {
		return result, newIssue(url.String(), UnableToInspectMediaTypeFromContentType, mediaTypeError.Error(), true)
	}
	return result, nil
}

func (t PageType) ContentType() string {
	return t.ContType
}

func (t PageType) MediaType() string {
	return t.MedType
}

func (t PageType) MediaTypeParams() MediaTypeParams {
	return t.MedTypeParams
}
