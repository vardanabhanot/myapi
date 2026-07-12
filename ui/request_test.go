package ui

import (
	"net/url"
	"testing"

	"github.com/vardanabhanot/myapi/core"
)

func TestSyncQueryParams(t *testing.T) {
	rows := []core.FormType{
		{Checked: true, Key: "a", Value: "old"},  // value updated from URL
		{Checked: true, Key: "gone", Value: "x"}, // removed from URL → dropped
		{Checked: false, Key: "off", Value: "y"}, // unchecked → kept as-is
		{Checked: true},                          // trailing empty row
	}
	values, _ := url.ParseQuery("a=new&fresh=1")

	syncQueryParams(&rows, values)

	byKey := map[string]core.FormType{}
	for _, r := range rows {
		byKey[r.Key] = r
	}

	if r := byKey["a"]; r.Value != "new" || !r.Checked {
		t.Errorf("a not updated: %+v", r)
	}
	if _, ok := byKey["gone"]; ok {
		t.Error("removed key kept")
	}
	if r := byKey["off"]; r.Value != "y" || r.Checked {
		t.Errorf("unchecked row changed: %+v", r)
	}
	if r := byKey["fresh"]; r.Value != "1" || !r.Checked {
		t.Errorf("new key not added: %+v", r)
	}
	if last := rows[len(rows)-1]; last.Key != "" || !last.Checked {
		t.Errorf("no trailing empty row: %+v", last)
	}
	if len(rows) != 4 { // a, off, fresh, empty
		t.Errorf("want 4 rows, got %d: %+v", len(rows), rows)
	}
}
