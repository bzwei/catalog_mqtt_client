package upload

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUpload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("mock body"))

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Error("Header Content-Type does not contain multipart/form-data")
		}

		if r.RequestURI != "/upload" {
			t.Errorf("Expected uploaded to /upload, got %s", r.RequestURI)
		}
	}))

	defer ts.Close()

	data := []byte(strings.Repeat("na", 512))
	f, err := os.Create("testuploaddata.file")
	if err != nil {
		t.Errorf("Failed to create test datafile for uploading. Reason %s", err)
	}
	f.Write(data)
	f.Close()
	defer os.Remove("testuploaddata.file")

	body, err := Upload(ts.URL+"/upload", "testuploaddata.file", "")
	if err != nil {
		t.Error("ERROR from Upload:", err)
	}
	if string(body) != "mock body" {
		t.Error("Retrieved body is not expected")
	}
}
