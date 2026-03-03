package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"strings"
)

// ExtractModelFromRequest extracts the "model" field from either
// a JSON body or a multipart/form-data body.
// Returns empty string if model cannot be found.
func ExtractModelFromRequest(body []byte, contentType string) string {
	if strings.HasPrefix(contentType, "multipart/") {
		return extractModelFromMultipart(body, contentType)
	}

	// JSON body
	var peek struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &peek); err != nil {
		return ""
	}
	return peek.Model
}

func extractModelFromMultipart(body []byte, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	boundary := params["boundary"]
	if boundary == "" {
		return ""
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "model" {
			modelBytes, _ := io.ReadAll(part)
			part.Close()
			return strings.TrimSpace(string(modelBytes))
		}
		part.Close()
	}
	return ""
}
