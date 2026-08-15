package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/kgretzky/pwndrop/storage"
)

func DownloadLogOptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET,DELETE,OPTIONS")
}

func DownloadLogGetHandler(w http.ResponseWriter, r *http.Request) {
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

	logs, err := storage.DownloadLogGetByFile(id)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}

	type Response struct {
		Logs []storage.DbDownloadLog `json:"logs"`
	}
	DumpResponse(w, "ok", http.StatusOK, 0, &Response{Logs: logs})
}

func DownloadLogClearHandler(w http.ResponseWriter, r *http.Request) {
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

	storage.DownloadLogDeleteByFile(id)
	DumpResponse(w, "ok", http.StatusOK, 0, nil)
}
