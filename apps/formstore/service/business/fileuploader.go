package business

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// FileRef represents an uploaded file reference that replaces inline file data.
type FileRef struct {
	Type        string `json:"_type"`
	MXCURI      string `json:"mxc_uri"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// DetectedFile holds decoded file data found during JSON walk.
type DetectedFile struct {
	Path        string
	Filename    string
	ContentType string
	Data        []byte
}

// FileUploader detects and processes file fields in JSON form data.
type FileUploader struct {
	uploadFn func(filename, contentType string, data []byte) (string, error)
}

// NewFileUploader creates a FileUploader with the given upload function.
// The uploadFn should upload data and return an MXC URI.
func NewFileUploader(uploadFn func(filename, contentType string, data []byte) (string, error)) *FileUploader {
	return &FileUploader{uploadFn: uploadFn}
}

// ProcessFields walks JSON data, detects file fields, uploads them, and replaces
// the field values with file references. Returns the processed data and file count.
func (u *FileUploader) ProcessFields(data map[string]any) (map[string]any, int, error) {
	result, count, err := u.walkAndReplace(data)
	if err != nil {
		return nil, 0, err
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return data, 0, nil
	}

	return resultMap, count, nil
}

//nolint:gocognit // recursive traversal requires branching by type
func (u *FileUploader) walkAndReplace(value any) (any, int, error) {
	totalFiles := 0

	switch v := value.(type) {
	case map[string]any:
		// Pattern A: explicit file object {"_type": "file", "data": "base64...", ...}
		if typeField, ok := v["_type"].(string); ok && typeField == "file" {
			ref, err := u.processExplicitFile(v)
			if err != nil {
				return nil, 0, err
			}

			return ref, 1, nil
		}

		// Recurse into map fields.
		for key, val := range v {
			processed, count, err := u.walkAndReplace(val)
			if err != nil {
				return nil, 0, fmt.Errorf("field %s: %w", key, err)
			}

			v[key] = processed
			totalFiles += count
		}

		return v, totalFiles, nil

	case []any:
		for i, val := range v {
			processed, count, err := u.walkAndReplace(val)
			if err != nil {
				return nil, 0, fmt.Errorf("index %d: %w", i, err)
			}

			v[i] = processed
			totalFiles += count
		}

		return v, totalFiles, nil

	case string:
		// Pattern B: data URI string "data:image/jpeg;base64,..."
		if strings.HasPrefix(v, "data:") {
			ref, err := u.processDataURI(v)
			if err != nil {
				return nil, 0, err
			}

			return ref, 1, nil
		}

		return v, 0, nil

	default:
		return value, 0, nil
	}
}

func (u *FileUploader) processExplicitFile(obj map[string]any) (map[string]any, error) {
	dataStr, _ := obj["data"].(string)
	if dataStr == "" {
		return nil, fmt.Errorf("%w: missing data field in file object", ErrInvalidFormData)
	}

	filename, _ := obj["filename"].(string)
	if filename == "" {
		filename = "upload"
	}

	contentType, _ := obj["content_type"].(string)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	decoded, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 data: %w", ErrInvalidFormData, err)
	}

	mxcURI, err := u.uploadFn(filename, contentType, decoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFileUploadFailed, err)
	}

	return map[string]any{
		"_type":        "file_ref",
		"mxc_uri":      mxcURI,
		"filename":     filename,
		"content_type": contentType,
	}, nil
}

func (u *FileUploader) processDataURI(uri string) (map[string]any, error) {
	// Parse "data:image/jpeg;base64,/9j/4AAQ..."
	const dataURIParts = 2
	parts := strings.SplitN(uri, ",", dataURIParts)
	if len(parts) != dataURIParts {
		return nil, fmt.Errorf("%w: malformed data URI", ErrInvalidFormData)
	}

	header := parts[0] // "data:image/jpeg;base64"
	dataStr := parts[1]

	contentType := "application/octet-stream"
	filename := "upload"

	headerParts := strings.TrimPrefix(header, "data:")
	mediaAndEncoding := strings.Split(headerParts, ";")

	if len(mediaAndEncoding) > 0 && mediaAndEncoding[0] != "" {
		contentType = mediaAndEncoding[0]
	}

	// Derive filename extension from content type.
	if ext := extensionFromContentType(contentType); ext != "" {
		filename = "upload" + ext
	}

	decoded, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 in data URI: %w", ErrInvalidFormData, err)
	}

	mxcURI, err := u.uploadFn(filename, contentType, decoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFileUploadFailed, err)
	}

	return map[string]any{
		"_type":        "file_ref",
		"mxc_uri":      mxcURI,
		"filename":     filename,
		"content_type": contentType,
	}, nil
}

func extensionFromContentType(ct string) string {
	extensions := map[string]string{
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"text/csv":        ".csv",
	}

	if ext, ok := extensions[ct]; ok {
		return ext
	}

	return ""
}
