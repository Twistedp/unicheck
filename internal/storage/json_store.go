package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"unicheck/internal/model"
)

const appConfigDirName = "uni-organizer"

func DataFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir = filepath.Join(dir, appConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "data.json"), nil
}

func LoadData(path string) (model.AppData, error) {
	var data model.AppData

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}

	if len(content) == 0 {
		return data, nil
	}

	err = json.Unmarshal(content, &data)
	return data, err
}

func SaveDataAtomic(path string, data model.AppData) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
