package storage

type DbFolder struct {
	ID         int    `json:"id" storm:"id,increment"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time" storm:"index"`
}

func FolderCreate(o *DbFolder) (*DbFolder, error) {
	err := db.Save(o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func FolderList() ([]DbFolder, error) {
	var dbos []DbFolder
	err := db.All(&dbos)
	if err != nil {
		return []DbFolder{}, nil
	}
	return dbos, nil
}

func FolderGet(id int) (*DbFolder, error) {
	var o DbFolder
	err := db.One("ID", id, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func FolderUpdate(id int, o *DbFolder) (*DbFolder, error) {
	if err := db.UpdateField(&DbFolder{ID: id}, "Name", o.Name); err != nil {
		return nil, err
	}
	return FolderGet(id)
}

func FolderDelete(id int) error {
	f := &DbFolder{ID: id}
	return db.DeleteStruct(f)
}
