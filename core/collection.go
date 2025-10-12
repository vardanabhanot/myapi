package core

import (
	"fmt"
	"os"
	"path/filepath"
)

type Collection struct {
	name     string
	requests []Request
}

func NewCollection(collectionName string) bool {
	localDir, err := os.UserCacheDir()

	if err != nil {
		return false
	}

	collectionPath := filepath.Join(localDir, collectionName)
	if _, err = os.Stat(collectionPath); err != nil {
		if !os.IsNotExist(err) {
			return false
		}

		if err = os.Mkdir(collectionPath, os.ModeDir); err != nil {
			return false
		}
	}
	return true
}

// For first iteration the return type will be bool
// but will later need to make it an array of something
func ListCollections() bool {
	localDir, err := os.UserCacheDir()

	if err != nil {
		return false
	}

	fmt.Println(localDir)

	return true
}
