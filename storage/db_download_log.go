package storage

type DbDownloadLog struct {
	ID           int    `json:"id" storm:"id,increment"`
	FileID       int    `json:"file_id" storm:"index"`
	IP           string `json:"ip"`
	UserAgent    string `json:"user_agent"`
	DownloadTime int64  `json:"download_time" storm:"index"`
}

func DownloadLogCreate(o *DbDownloadLog) error {
	return db.Save(o)
}

func DownloadLogGetByFile(file_id int) ([]DbDownloadLog, error) {
	var dbos []DbDownloadLog
	err := db.Find("FileID", file_id, &dbos)
	if err != nil {
		return []DbDownloadLog{}, nil
	}
	// reverse: newest first
	for i, j := 0, len(dbos)-1; i < j; i, j = i+1, j-1 {
		dbos[i], dbos[j] = dbos[j], dbos[i]
	}
	return dbos, nil
}

func DownloadLogDeleteByFile(file_id int) error {
	var dbos []DbDownloadLog
	db.Find("FileID", file_id, &dbos)
	for _, o := range dbos {
		db.DeleteStruct(&o)
	}
	return nil
}
