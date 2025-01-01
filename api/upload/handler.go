package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

func generateFileTagId(dir string) string {
	hash := sha256.Sum256([]byte(dir))
	return hex.EncodeToString(hash[:])
}

func Handle(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request",
		"content-type", r.Header.Get("Content-Type"),
		"method", r.Method)

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		slog.Error("Error parsing multipart form", "err", err)
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Debug the form fields
	if r.MultipartForm != nil {
		slog.Info("Form fields",
			"file_keys", fmt.Sprintf("%v", r.MultipartForm.File),
			"value_keys", fmt.Sprintf("%v", r.MultipartForm.Value))
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		slog.Error("Error retrieving file",
			"err", err,
			"form_size", r.ContentLength)
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	temp, err := os.CreateTemp("", "tag")
	if err != nil {
		slog.Error("Error creating temp file")
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer temp.Close()

	_, err = io.Copy(temp, file)
	if err != nil {
		slog.Info("Error copying file", "file", temp.Name())
		err := os.Remove(temp.Name())
		if err != nil {
			slog.Info("Error deleting temp file", "temp", temp.Name())
			return
		}
		http.Error(w, "Error copying data", http.StatusInternalServerError)
		return
	}

	slog.Info("Success", "temp file", temp.Name())
	w.WriteHeader(http.StatusOK)
}
