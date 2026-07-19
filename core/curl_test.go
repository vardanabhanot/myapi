package core

import (
	"testing"
)

func TestParseCurl(t *testing.T) {
	// Chrome "Copy as cURL (bash)": multiline, --compressed, $'...'
	chromeGet := `curl 'https://api.example.com/users?page=2&sort=name' \
  -H 'accept: application/json' \
  -H $'cookie: sid=abc123' \
  --compressed`

	// Chrome "Copy as cURL (cmd)": ^ continuations, double quotes, \" escapes
	cmdPost := `curl "https://api.example.com/users" ^
  -H "content-type: application/json" ^
  --data-raw "{\"name\":\"bob\"}" ^
  --compressed`

	jsonPost := `curl 'https://api.example.com/items' -H 'Content-Type: application/json; charset=UTF-8' --data-raw '{"a":"b c"}'`

	t.Run("chrome bash GET", func(t *testing.T) {
		r, err := ParseCurl(chromeGet)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != "GET" || r.URL != "https://api.example.com/users?page=2&sort=name" {
			t.Fatalf("method=%q url=%q", r.Method, r.URL)
		}
		if len(*r.Headers) != 2 || (*r.Headers)[1].Key != "cookie" || (*r.Headers)[1].Value != "sid=abc123" {
			t.Fatalf("headers: %+v", *r.Headers)
		}
		if len(*r.QueryParams) != 2 || (*r.QueryParams)[0].Key != "page" || (*r.QueryParams)[0].Value != "2" {
			t.Fatalf("query params: %+v", *r.QueryParams)
		}
		if r.ID == "" || !r.IsDirty {
			t.Fatalf("needs fresh ID and dirty flag: %+v", r)
		}
	})

	t.Run("cmd POST json", func(t *testing.T) {
		r, err := ParseCurl(cmdPost)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" { // implied by --data-raw
			t.Fatalf("method=%q", r.Method)
		}
		if r.BodyType != "JSON" || r.Body.Json != `{"name":"bob"}` {
			t.Fatalf("bodytype=%q json=%q", r.BodyType, r.Body.Json)
		}
	})

	t.Run("json content type", func(t *testing.T) {
		r, err := ParseCurl(jsonPost)
		if err != nil {
			t.Fatal(err)
		}
		if r.BodyType != "JSON" || r.Body.Json != `{"a":"b c"}` {
			t.Fatalf("bodytype=%q json=%q", r.BodyType, r.Body.Json)
		}
	})

	t.Run("urlencoded default", func(t *testing.T) {
		r, err := ParseCurl(`curl https://x.test/login -d 'user=bob' -d 'pass=a%26b'`)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || r.BodyType != "URL Encoded" {
			t.Fatalf("method=%q bodytype=%q", r.Method, r.BodyType)
		}
		rows := *r.Body.Form
		if len(rows) != 2 || rows[0].Key != "user" || rows[1].Value != "a&b" {
			t.Fatalf("form rows: %+v", rows)
		}
	})

	t.Run("multipart form", func(t *testing.T) {
		r, err := ParseCurl(`curl https://x.test/upload -F 'name=bob' -F 'file=@photo.png'`)
		if err != nil {
			t.Fatal(err)
		}
		rows := *r.Body.Form
		if r.BodyType != "Form" || len(rows) != 2 || rows[0].IsFile {
			t.Fatalf("bodytype=%q form=%+v", r.BodyType, rows)
		}
		if !rows[1].IsFile || rows[1].Value != "photo.png" {
			t.Fatalf("-F file=@photo.png should be a file row: %+v", rows[1])
		}
	})

	t.Run("basic auth insecure cookie", func(t *testing.T) {
		r, err := ParseCurl(`curl -u alice:s3cret -k -b 'a=1' -b 'b=2' https://x.test/`)
		if err != nil {
			t.Fatal(err)
		}
		if r.AuthType != "Basic" || r.Auth.BasicUser != "alice" || r.Auth.BasicPass != "s3cret" {
			t.Fatalf("auth: %+v", r.Auth)
		}
		if !r.Settings.SkipTLSVerify {
			t.Fatal("-k should set SkipTLSVerify")
		}
		if (*r.Headers)[0].Key != "Cookie" || (*r.Headers)[0].Value != "a=1; b=2" {
			t.Fatalf("cookie header: %+v", *r.Headers)
		}
	})

	t.Run("explicit method wins", func(t *testing.T) {
		r, err := ParseCurl(`curl -X PUT https://x.test/thing -d 'a=1'`)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != "PUT" {
			t.Fatalf("method=%q", r.Method)
		}
	})

	t.Run("ignored arg flags don't eat the URL", func(t *testing.T) {
		r, err := ParseCurl(`curl -s -L -o out.bin -A 'Mozilla' https://x.test/dl`)
		if err != nil {
			t.Fatal(err)
		}
		if r.URL != "https://x.test/dl" {
			t.Fatalf("url=%q", r.URL)
		}
	})

	t.Run("text body on other content type", func(t *testing.T) {
		r, err := ParseCurl(`curl https://x.test/ -H 'content-type: text/csv' -d 'a,b,c'`)
		if err != nil {
			t.Fatal(err)
		}
		if r.BodyType != "Text" || r.Body.Text != "a,b,c" {
			t.Fatalf("bodytype=%q text=%q", r.BodyType, r.Body.Text)
		}
	})

	t.Run("errors", func(t *testing.T) {
		for _, bad := range []string{"wget https://x.test/", "curl", "curl -H 'a: b'", "curl 'unterminated"} {
			if _, err := ParseCurl(bad); err == nil {
				t.Errorf("ParseCurl(%q) should error", bad)
			}
		}
	})
}
