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

// The request data are saved in json files, in the OS's Cache dir
// And the time of creation of the tab of request is used as the key

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

	sort.Slice(requestFiles, func(i, j int) bool {
		infoI, errI := requestFiles[i].Info()
		infoJ, errJ := requestFiles[j].Info()

		if errI != nil || errJ != nil {
			// If error, keep original order (safe fallback)
			return false
		}

		return infoI.ModTime().After(infoJ.ModTime())
	})

	requests = LazyLoadHistory(0, &requestFiles)

	return &requests
}

func LazyLoadHistory(index int, requestFiles *[]os.DirEntry) []map[string]string {

	var requests []map[string]string

	for i, file := range *requestFiles {

		if i > 20 {
			break
		}

		var request = make(map[string]string)

		request["ID"] = file.Name()
		requests = append(requests, request)
	}

	return requests
}

func LoadMetaData(filename string, request *map[string]string) {
	localDir, _ := os.UserCacheDir()
	myapiPath := filepath.Join(localDir, "/myapi/")

	filePath := filepath.Join(myapiPath, filename)

	fileContent, err := os.ReadFile(filePath)

	if err != nil {
		return
	}

	fileStat, err := os.Stat(filePath)

	if err != nil {
		return
	}

	content := &Request{}
	if err = json.Unmarshal(fileContent, content); err != nil {
		return
	}

	(*request)["requestURL"] = content.URL
	(*request)["method"] = content.Method
	(*request)["mtime"] = timeAgo(fileStat.ModTime())
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

func CloneHistory(id string) error {
	localDir, err := os.UserCacheDir()

	if err != nil {
		return err
	}

	file := filepath.Join(localDir, "/myapi/"+id)

	fileContent, err := os.ReadFile(file)

	if err != nil {
		return err
	}

	var request Request
	if err = json.Unmarshal(fileContent, &request); err != nil {
		return err
	}

	request.ID = "" // emptying the old ID so it regenerates

	_, err = saveRequestData(&request)

	if err != nil {
		return err
	}

	return nil
}

func timeAgo(reqTime time.Time) string {

	duration := time.Since(reqTime)

	if duration.Hours() < 24 {
		if duration.Hours() > 1 {
			return fmt.Sprintf("%dh", int(duration.Hours()))
		} else if duration.Minutes() > 1 {
			return fmt.Sprintf("%dm", int(duration.Minutes()))
		} else if duration.Seconds() > 10 {
			return fmt.Sprintf("%ds", int(duration.Seconds()))
		} else {
			return "Now"
		}
	}

	if duration.Hours() >= 8760 {
		years := duration.Hours() / 8760
		return fmt.Sprintf("%dY", int(years))
	} else if duration.Hours() >= 730 {
		months := duration.Hours() / 730
		return fmt.Sprintf("%dM", int(months))
	} else if duration.Hours() >= 168 {
		weeks := duration.Hours() / 168
		return fmt.Sprintf("%dW", int(weeks))
	} else {
		days := duration.Hours() / 24
		return fmt.Sprintf("%dd", int(days))
	}
}
