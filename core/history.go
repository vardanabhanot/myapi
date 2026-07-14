package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The request data are saved in json files, in the OS's Cache dir
// And the time of creation of the tab of request is used as the key

// NewRequestID is the single source of request/tab identity. Nanoseconds:
// second-resolution IDs collided when two tabs were created within the same
// second, silently sharing one history file.
func NewRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// HistoryEntry is one row in the history list. ID is the bare request ID —
// the ".json" filename suffix never leaves this file.
type HistoryEntry struct {
	ID     string
	URL    string
	Method string
	MTime  string
	Loaded bool // metadata lazy-loads per visible row; set by LoadMeta
}

// historyDir is the single place that knows where history lives.
func historyDir() (string, error) {
	localDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(localDir, "myapi"), nil
}

func historyFile(id string) (string, error) {
	dir, err := historyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

func ListHistory() []*HistoryEntry {
	var requests []*HistoryEntry

	myapiPath, err := historyDir()
	if err != nil {
		return requests
	}

	requestFiles, err := os.ReadDir(myapiPath)
	if err != nil {
		return requests
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

	// Rows only carry the ID here; the list lazy-loads metadata per visible
	// row, so listing everything is cheap even for a large history.
	for _, file := range requestFiles {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue // stray files (desktop.ini and friends) aren't history
		}

		requests = append(requests, &HistoryEntry{ID: strings.TrimSuffix(file.Name(), ".json")})
	}

	return requests
}

// LoadMeta fills URL/Method/MTime from the entry's file. Marks the entry
// Loaded even on failure so a broken file isn't re-read on every redraw.
func (e *HistoryEntry) LoadMeta() {
	e.Loaded = true

	filePath, err := historyFile(e.ID)
	if err != nil {
		return
	}

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

	e.URL = content.URL
	e.Method = content.Method
	e.MTime = timeAgo(fileStat.ModTime())
}

func saveRequestData(request *Request) (bool, error) {
	myapiPath, err := historyDir()

	if err != nil {
		//dialog.ShowError(err, *ui.Gui.Window)
		//TODO:: We need to shift this dialog to the ui package
		return false, err
	}

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

	if request.ID == "" {
		request.ID = NewRequestID()
	}

	requestFile, err := historyFile(request.ID)

	if err != nil {
		return false, err
	}

	jsondata, err := json.Marshal(request)

	if err != nil {
		return false, err
	}

	if err := os.WriteFile(requestFile, jsondata, 0o644); err != nil {
		return false, err
	}

	return true, nil

}

func ClearHistory() error {
	myapiPath, err := historyDir()

	if err != nil {
		return err
	}

	entries, err := os.ReadDir(myapiPath)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if err := os.Remove(filepath.Join(myapiPath, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func DeleteHistory(id string) error {
	file, err := historyFile(id)

	if err != nil {
		return err
	}

	return os.Remove(file)
}

func LoadRequest(id string) (*Request, error) {
	file, err := historyFile(id)

	if err != nil {
		return nil, err
	}

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
	file, err := historyFile(id)

	if err != nil {
		return err
	}

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
