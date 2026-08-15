package storage

import (
	"github.com/kgretzky/pwndrop/log"
)

type DbFile struct {
	ID           int    `json:"id" storm:"id,increment"`
	Uid          int    `json:"uid" storm:"index"`
	Name         string `json:"name"`
	Filename     string `json:"fname"`
	FileSize     int64  `json:"fsize"`
	UrlPath      string `json:"url_path" storm:"unique"`
	MimeType     string `json:"mime_type"`
	OrigMimeType string `json:"orig_mime_type"`
	CreateTime   int64  `json:"create_time" storm:"index"`
	IsEnabled    bool   `json:"is_enabled"`
	IsPaused     bool   `json:"is_paused"`
	RedirectPath string `json:"redirect_path" storm:"unique"`
	SubName      string `json:"sub_name"`
	SubMimeType  string `json:"sub_mime_type"`
	RefSubFile   int    `json:"ref_sub_file"`
	MaxDownloads   int    `json:"max_downloads"`
	DownloadCount  int    `json:"download_count"`
	DownloadsLeft  int    `json:"downloads_left"`
	FolderID       int    `json:"folder_id" storm:"index"`
	Comment        string `json:"comment"`
}

func FileCreate(o *DbFile) (*DbFile, error) {
	err := db.Save(o)
	if err != nil {
		return nil, err
	}
	log.Debug("file id: %d", o.ID)
	return o, nil
}

func FileList() ([]DbFile, error) {
	var dbos []DbFile

	err := db.All(&dbos)
	if err != nil {
		return nil, err
	}
	/*
		for _, dbo := range dbos {
			log.Debug("filelist: sub_id: %d", dbo.RefSubFile)
			if dbo.RefSubFile > 0 {
				subf, err := SubFileGet(f.RefSubFile)
				if err == nil {
					jf.SubFile = subf
				}
			}
			ret = append(ret, dbo)
		}*/
	return dbos, nil
}

func FileGet(id int) (*DbFile, error) {
	var o DbFile
	err := db.One("ID", id, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func FileGetByUrl(url string) (*DbFile, error) {
	var o DbFile
	err := db.One("UrlPath", url, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func FileGetByRedirectUrl(url string) (*DbFile, error) {
	var o DbFile
	err := db.One("RedirectPath", url, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func FileDirExists(url string) bool {
	var o []DbFile
	if url == "" {
		return false
	}
	if url[len(url)-1] != '/' {
		url += "/"
	}
	err := db.Prefix("UrlPath", url, &o)
	if err != nil {
		return false
	}
	return true
}

func FileDelete(id int) error {
	f := &DbFile{
		ID: id,
	}
	err := db.DeleteStruct(f)
	if err != nil {
		return err
	}
	return nil
}

func FileUpdate(id int, o *DbFile) (*DbFile, error) {
	existing, err := FileGet(id)
	if err != nil {
		return nil, err
	}

	if err := db.Update(&DbFile{ID: id, Name: o.Name, UrlPath: o.UrlPath, MimeType: o.MimeType, RefSubFile: o.RefSubFile, SubName: o.SubName, RedirectPath: o.RedirectPath, SubMimeType: o.SubMimeType}); err != nil {
		return nil, err
	}
	if err := db.UpdateField(&DbFile{ID: id}, "RedirectPath", o.RedirectPath); err != nil {
		return nil, err
	}
//	return o, nil

	if err := db.UpdateField(&DbFile{ID: id}, "MaxDownloads", o.MaxDownloads); err != nil {
		return nil, err
	}
	// when limit changes, reset remaining to the new limit
	if o.MaxDownloads != existing.MaxDownloads {
		if err := db.UpdateField(&DbFile{ID: id}, "DownloadsLeft", o.MaxDownloads); err != nil {
			return nil, err
		}
	}
	if err := db.UpdateField(&DbFile{ID: id}, "FolderID", o.FolderID); err != nil {
		return nil, err
	}
	if err := db.UpdateField(&DbFile{ID: id}, "Comment", o.Comment); err != nil {
		return nil, err
	}
	f, err := FileGet(id)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func FileIncrementDownload(id int) error {
	f, err := FileGet(id)
	if err != nil {
		return err
	}
	// always track total count
	if err := db.UpdateField(&DbFile{ID: id}, "DownloadCount", f.DownloadCount+1); err != nil {
		return err
	}
	// decrement remaining and auto-disable at 0
	if f.MaxDownloads > 0 {
		left := f.DownloadsLeft - 1
		if left < 0 {
			left = 0
		}
		if err := db.UpdateField(&DbFile{ID: id}, "DownloadsLeft", left); err != nil {
			return err
		}
		if left == 0 {
			db.UpdateField(&DbFile{ID: id}, "IsEnabled", false)
		}
	}
	return nil
}

func FileGetByFolder(folder_id int) ([]DbFile, error) {
	var dbos []DbFile
	err := db.Find("FolderID", folder_id, &dbos)
	if err != nil {
		return []DbFile{}, nil
	}
	return dbos, nil
}

func FileResetSubFile(id int) (*DbFile, error) {
	if err := db.UpdateField(&DbFile{ID: id}, "RefSubFile", 0); err != nil {
		return nil, err
	}
	o, err := FileGet(id)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func FileEnable(id int, enable bool) (*DbFile, error) {
	if err := db.UpdateField(&DbFile{ID: id}, "IsEnabled", enable); err != nil {
		return nil, err
	}
	// on manual re-enable with a limit set but no remaining — give 1 download
	if enable {
		o, err := FileGet(id)
		if err != nil {
			return nil, err
		}
		if o.MaxDownloads > 0 && o.DownloadsLeft <= 0 {
			if err := db.UpdateField(&DbFile{ID: id}, "DownloadsLeft", 1); err != nil {
				return nil, err
			}
		}
	}
	o, err := FileGet(id)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func FilePause(id int, pause bool) (*DbFile, error) {
	if err := db.UpdateField(&DbFile{ID: id}, "IsPaused", pause); err != nil {
		return nil, err
	}
	o, err := FileGet(id)
	if err != nil {
		return nil, err
	}
	return o, nil
}
