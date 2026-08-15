package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/kgretzky/pwndrop/storage"
)

func FolderOptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
}

func FolderListHandler(w http.ResponseWriter, r *http.Request) {
	_, err := AuthSession(r)
	if err != nil {
		DumpResponse(w, "unauthorized", http.StatusUnauthorized, API_ERROR_BAD_AUTHENTICATION, nil)
		return
	}

	folders, err := storage.FolderList()
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}
	type Response struct {
		Folders []storage.DbFolder `json:"folders"`
	}
	DumpResponse(w, "ok", http.StatusOK, 0, &Response{Folders: folders})
}

func FolderCreateHandler(w http.ResponseWriter, r *http.Request) {
	_, err := AuthSession(r)
	if err != nil {
		DumpResponse(w, "unauthorized", http.StatusUnauthorized, API_ERROR_BAD_AUTHENTICATION, nil)
		return
	}

	var req storage.DbFolder
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}
	if req.Name == "" {
		DumpResponse(w, "name is required", http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	o := &storage.DbFolder{
		Name:       req.Name,
		CreateTime: time.Now().Unix(),
	}
	f, err := storage.FolderCreate(o)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}
	DumpResponse(w, "ok", http.StatusOK, 0, f)
}

func FolderUpdateHandler(w http.ResponseWriter, r *http.Request) {
	_, err := AuthSession(r)
	if err != nil {
		DumpResponse(w, "unauthorized", http.StatusUnauthorized, API_ERROR_BAD_AUTHENTICATION, nil)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	var req storage.DbFolder
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	f, err := storage.FolderUpdate(id, &req)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}
	DumpResponse(w, "ok", http.StatusOK, 0, f)
}

func FolderDeleteHandler(w http.ResponseWriter, r *http.Request) {
	_, err := AuthSession(r)
	if err != nil {
		DumpResponse(w, "unauthorized", http.StatusUnauthorized, API_ERROR_BAD_AUTHENTICATION, nil)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	// Move all files in this folder to root (folder_id = 0)
	files, _ := storage.FileGetByFolder(id)
	for _, f := range files {
		f.FolderID = 0
		storage.FileUpdate(f.ID, &f)
	}

	if err := storage.FolderDelete(id); err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}
	DumpResponse(w, "ok", http.StatusOK, 0, nil)
}
