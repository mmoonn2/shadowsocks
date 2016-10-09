package utils

import (
	"os"
	"path/filepath"
)

// CreateFileDir create the dir of file
func CreateFileDir(filePath string) (err error) {
	// filepath.Dir(filepath)
	if _, err = os.Stat(filePath); err != nil { // check log file if exist
		if os.IsNotExist(err) {
			filePathDir := filepath.Dir(filePath)
			if _, err = os.Stat(filePath); os.IsNotExist(err) { // check dir of file exist
				return os.MkdirAll(filePathDir, 0755)
			}
		}
		return
	}
	return
}
