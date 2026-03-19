package business

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestFileUploader_ProcessFields(t *testing.T) {
	t.Parallel()

	uploaded := make([]DetectedFile, 0, 2)
	uploader := NewFileUploader(func(filename, contentType string, data []byte) (string, error) {
		uploaded = append(uploaded, DetectedFile{
			Filename:    filename,
			ContentType: contentType,
			Data:        append([]byte(nil), data...),
		})
		return "mxc://server/" + filename, nil
	})

	pngBytes := []byte("png-bytes")
	payload := map[string]any{
		"profile": map[string]any{
			"avatar": map[string]any{
				"_type":        "file",
				"filename":     "avatar.png",
				"content_type": "image/png",
				"data":         base64.StdEncoding.EncodeToString(pngBytes),
			},
		},
		"attachments": []any{
			"data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte("hello")),
		},
	}

	processed, count, err := uploader.ProcessFields(payload)
	if err != nil {
		t.Fatalf("ProcessFields() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d", count)
	}
	if len(uploaded) != 2 {
		t.Fatalf("uploads = %d", len(uploaded))
	}
	if processed["profile"].(map[string]any)["avatar"].(map[string]any)["mxc_uri"] != "mxc://server/avatar.png" {
		t.Fatalf("processed = %+v", processed)
	}
	foundDataURIUpload := false
	for _, item := range uploaded {
		if item.Filename == "upload.txt" {
			foundDataURIUpload = true
			break
		}
	}
	if !foundDataURIUpload {
		t.Fatalf("uploads = %+v", uploaded)
	}
}

func TestFileUploader_ErrorPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		errIs error
	}{
		{
			name:  "explicit file missing data",
			value: map[string]any{"_type": "file", "filename": "missing"},
			errIs: ErrInvalidFormData,
		},
		{
			name:  "explicit file invalid base64",
			value: map[string]any{"_type": "file", "data": "%"},
			errIs: ErrInvalidFormData,
		},
		{
			name:  "data uri malformed",
			value: "data:image/png;base64",
			errIs: ErrInvalidFormData,
		},
		{
			name:  "data uri invalid base64",
			value: "data:image/png;base64,%",
			errIs: ErrInvalidFormData,
		},
	}

	uploader := NewFileUploader(func(string, string, []byte) (string, error) {
		return "mxc://ok", nil
	})

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := uploader.ProcessFields(map[string]any{"field": tc.value})
			if !errors.Is(err, tc.errIs) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	uploadErr := errors.New("upload failed")
	failing := NewFileUploader(func(string, string, []byte) (string, error) {
		return "", uploadErr
	})
	_, _, err := failing.ProcessFields(map[string]any{
		"file": map[string]any{
			"_type": "file",
			"data":  base64.StdEncoding.EncodeToString([]byte("x")),
		},
	})
	if !errors.Is(err, ErrFileUploadFailed) {
		t.Fatalf("error = %v", err)
	}
}

func TestExtensionFromContentType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		contentType string
		ext         string
	}{
		{contentType: "image/jpeg", ext: ".jpg"},
		{contentType: "application/pdf", ext: ".pdf"},
		{contentType: "unknown/type", ext: ""},
	}

	for _, tc := range cases {

		t.Run(tc.contentType, func(t *testing.T) {
			t.Parallel()
			if got := extensionFromContentType(tc.contentType); got != tc.ext {
				t.Fatalf("extensionFromContentType(%q) = %q want %q", tc.contentType, got, tc.ext)
			}
		})
	}
}
