package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
//	"strings"
	"time"

	"github.com/kgretzky/pwndrop/log"
        "github.com/kgretzky/pwndrop/storage"
)

const BLACKLIST_JAIL_TIME_SECS = 10 * 60
const BLACKLIST_HITS_LIMIT = 10

type BlacklistItem struct {
	hits     int
	last_hit time.Time
}

type Http struct {
	srv *Server
}

func NewHttp(srv *Server) (*Http, error) {
	s := &Http{
		srv: srv,
	}
	return s, nil
}

func (s *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data_dir := Cfg.GetDataDir()

//	from_ip := r.RemoteAddr
//	if strings.Contains(from_ip, ":") {
//		from_ip = strings.Split(from_ip, ":")[0]
//	}
        from_ip := realIP(r)

	if r.Method == "GET" {
		f, status, err := s.srv.GetFile(r.URL.Path)
		if err != nil {
			log.Error("http: get: %s: %s (%s)", r.URL.Path, err, from_ip)
			err := s.killConnection(w, status)
			if err != nil {
				log.Error("http: %s (%s)", err, from_ip)
			}
			return
		}

		if f.RedirectPath != "" && f.RedirectPath != r.URL.Path && !f.IsPaused {
			log.Error("http: get: %s: redirecting to '%s' (%s)", r.URL.Path, f.RedirectPath, from_ip)
			http.Redirect(w, r, f.RedirectPath, http.StatusFound)
		} else {
			mime_type := f.MimeType
			if f.IsPaused {
				mime_type = f.SubMimeType
			}
			fpath := filepath.Join(data_dir, "files", f.Filename)
			fo, err := os.Open(fpath)
			//data, err := ioutil.ReadFile(fpath)
			if err != nil {
				log.Error("http: file: %s: %s (%s)", f.Filename, err, from_ip)
				return
			}
			defer fo.Close()

			storage.FileIncrementDownload(f.ID)
			storage.DownloadLogCreate(&storage.DbDownloadLog{
				FileID:       f.ID,
				IP:           from_ip,
				UserAgent:    r.Header.Get("User-Agent"),
				DownloadTime: time.Now().Unix(),
			})

			w.Header().Set("Content-Type", mime_type)

			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")

			w.WriteHeader(200)
			io.Copy(w, fo)
		}
		return
	}
	err := s.killConnection(w, 404)
	if err != nil {
		log.Error("http: %s (%s)", err, from_ip)
	}
}

func (s *Http) killConnection(w http.ResponseWriter, status int) error {
	if status > 0 {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(status)
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("connection hijacking not supported")
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
