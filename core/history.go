package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2/storage"
)

type fileMeta struct {
	modTime time.Time
	file    os.DirEntry
}

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

	requestFiles, err := os.ReadDir(myapiPath)

	if err != nil {
		return &requests
	}

	fileInfo := []fileMeta{}

	for _, file := range requestFiles {
		info, err := file.Info()

		if err == nil {
			fileInfo = append(fileInfo, fileMeta{info.ModTime(), file})
		}
	}

	sort.Slice(fileInfo, func(i, j int) bool {
		return fileInfo[i].modTime.After(fileInfo[j].modTime)
	})

	requests = LazyLoadHistory(0, &fileInfo)

	return &requests
}

func LazyLoadHistory(index int, fileInfo *[]fileMeta) []map[string]string {

	var requests []map[string]string

	localDir, _ := os.UserCacheDir()

	myapiPath := filepath.Join(localDir, "/myapi/")

	for i, file := range *fileInfo {

		if i > 20 {
			break
		}

		filePath := filepath.Join(myapiPath, file.file.Name())

		fileContent, err := os.ReadFile(filePath)

		if err != nil {
			continue
		}

		content := &Request{}
		if err = json.Unmarshal(fileContent, content); err != nil {
			continue
		}

		var request = make(map[string]string)

		request["ID"] = file.file.Name()
		request["requestURL"] = content.URL
		request["method"] = content.Method
		request["mtime"] = timeAgo(file.modTime)
		requests = append(requests, request)
	}

	return requests
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

func DeleteHistory(id string) error {

	localDir, err := os.UserCacheDir()

	if err != nil {
		return err
	}

	file := filepath.Join(localDir, "/myapi/"+id)

	return os.Remove(file)
}

func LoadRequest(id string) (*Request, error) {

	localDir, err := os.UserCacheDir()

	if err != nil {
		return nil, err
	}

	file := filepath.Join(localDir, "/myapi/"+id)

	fileContent, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	request := &Request{}
	err = json.Unmarshal(fileContent, request)

	if err != nil {
		return nil, err
	}

	return request, nil

}

func timeAgo(reqTime time.Time) string {

	duration := time.Since(reqTime)

	if duration.Hours() < 24 {
		if duration.Hours() > 1 {
			return fmt.Sprintf("%d Hours Ago", int(duration.Hours()))
		} else if duration.Minutes() > 1 {
			return fmt.Sprintf("%d Minutes Ago", int(duration.Hours()))
		} else {
			return "Now"
		}
	}

	if duration.Hours() >= 8760 {
		years := duration.Hours() / 8760
		return fmt.Sprintf("%d Years Ago", int(years))
	} else if duration.Hours() >= 730 {
		months := duration.Hours() / 730
		return fmt.Sprintf("%d Months Ago", int(months))
	} else if duration.Hours() >= 168 {
		weeks := duration.Hours() / 168
		return fmt.Sprintf("%d Weeks Ago", int(weeks))
	} else {
		days := duration.Hours() / 24
		return fmt.Sprintf("%d DaysAgo", int(days))
	}
}
