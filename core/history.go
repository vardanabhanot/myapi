package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/storage"
)

func ListHistory() *[]map[string]string {
	localDir, err := os.UserCacheDir()

	var requests []map[string]string

	if err != nil {
		return &requests
	}

	myapiPath := filepath.Join(localDir, "/myapi")

	_, err = os.Stat(myapiPath)

	if err != nil {
		return &requests
	}

	uri := storage.NewFileURI(myapiPath)

	files, err := storage.List(uri)

	if err != nil {
		return &requests
	}

	for _, file := range files {
		if _, err := storage.CanRead(file); err != nil {
			return &requests
		}

		reader, _ := storage.Reader(file)

		defer reader.Close()

		var fileContent []byte
		fileContent, err = io.ReadAll(reader)

		if err != nil {
			continue
		}

		content := &Request{}
		if err = json.Unmarshal(fileContent, content); err != nil {
			continue
		}

		var request = make(map[string]string)

		request["ID"] = file.String()
		request["requestURL"] = content.URL
		request["method"] = content.Method
		requests = append(requests, request)
	}

	return &requests
}

func saveRequestData(request *Request) (bool, error) {
	localDir, err := os.UserCacheDir()

	if err != nil {
		//dialog.ShowError(err, *ui.Gui.Window)
		//TODO:: We need to shift this dialog to the ui package
		return false, err
	}

	myapiPath := filepath.Join(localDir, "/myapi")

	_, err = os.Stat(myapiPath)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = os.Mkdir(myapiPath, os.ModeDir)
		}

		// If we still have an error, we need to let the user know
		if err != nil {
			//dialog.ShowError(err, *ui.Gui.Window)
			//TODO:: We need to shift this dialog to the ui package
			return false, err
		}
	}

	var filename string
	if request.ID == "" {
		filename = fmt.Sprintf("%d", time.Now().Unix())
		request.ID = filename
	} else {
		filename = request.ID
	}

	requestFile := filepath.Join(myapiPath, "/"+filename+".json")

	uri := storage.NewFileURI(requestFile)

	writer, _ := storage.Writer(uri)
	defer writer.Close()

	jsondata, err := json.Marshal(request)

	if err != nil {
		return false, err
	}

	writer.Write(jsondata)

	return true, nil

}
