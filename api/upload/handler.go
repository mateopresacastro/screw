package upload

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"tagg/cryptoutil"
	"tagg/session"
	"tagg/store"
)

type Upload struct {
	store store.Store
}

func New(store store.Store) *Upload {
	return &Upload{store: store}
}

func (u *Upload) Handle(w http.ResponseWriter, r *http.Request) {
	result, ok := session.FromContext(r.Context())
	if !ok {
		slog.Error("no session data on context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err := r.ParseMultipartForm(32 << 20) // 32mb
	if err != nil {
		slog.Error("Error parsing multipart form", "err", err)
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
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
		}
		http.Error(w, "Error copying data", http.StatusInternalServerError)
		return
	}

	slog.Info("Success", "temp file", temp.Name())
	ref, err := cryptoutil.Random()
	if err != nil {
		err := os.Remove(temp.Name())
		if err != nil {
			slog.Info("Error deleting temp file", "temp", temp.Name())
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	res := struct {
		Ref string `json:"ref"`
	}{Ref: ref}

	id := cryptoutil.ID(ref)
	tag := &store.Tag{ID: id, UserID: result.User.ID, FilePath: temp.Name()}
	err = u.store.CreateTag(tag)
	if err != nil {
		slog.Error("error creating tag")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("error encoding response", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

}
