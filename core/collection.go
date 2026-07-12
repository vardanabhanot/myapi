package core

import (
	"encoding/json"
	"os"
)

// A collection holds snapshot copies of requests: adding or opening always
// goes through Request.Clone, so collection entries never alias a live tab.
// ponytail: snapshots only — in-place editing of collection requests can
// come later if organizing alone proves insufficient.
type Collection struct {
	Name     string     `json:"Name"`
	Requests []*Request `json:"Requests"`
}

// Clone deep-copies a request via its JSON form. The ID is cleared; callers
// assign a fresh one when the copy becomes a tab or is sent.
func (r *Request) Clone() *Request {
	clone := &Request{}

	data, err := json.Marshal(r)
	if err != nil {
		return clone
	}

	json.Unmarshal(data, clone)
	clone.ID = ""

	return clone
}

// LoadCollections reads saved collections; nil on any error.
func LoadCollections() []*Collection {
	var collections []*Collection

	file, err := configFile("collections.json")
	if err != nil {
		return collections
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return collections
	}

	json.Unmarshal(content, &collections)

	return collections
}

func SaveCollections(collections []*Collection) error {
	file, err := configFile("collections.json")

	if err != nil {
		return err
	}

	data, err := json.Marshal(collections)

	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0o644)
}
