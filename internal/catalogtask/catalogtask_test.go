package catalogtask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
)

func assert(actual interface{}, expected interface{}, name string, t *testing.T) {
	if actual != expected {
		t.Errorf("%s not equal. Expected %v but actual %v", name, expected, actual)
	}
}

func toTime(str string) time.Time {
	dateTime, _ := time.Parse("2006-01-02T15:04:05Z", str)
	return dateTime
}

func TestGet(t *testing.T) {
	b := []byte(`{"id":"12345","state":"pending","status":"unknown","created_at":"2020-11-04T16:12:09Z","updated_at":"2020-11-04T16:12:09Z",
	            "input":{"response_format":"tar","upload_url":"/ingress/upload", "jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008"}]}}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))

	task := MakeCatalogTask(logger.CtxWithLoggerID(context.Background(), 123), ts.URL)
	reqMessage, err := task.Get()
	if err != nil {
		t.Fatalf("Error parsing request data %v", err)
	}

	assert(reqMessage.ID, "12345", "ID", t)
	assert(reqMessage.State, "pending", "State", t)
	assert(reqMessage.Status, "unknown", "Status", t)
	assert(reqMessage.CreatedAt, toTime("2020-11-04T16:12:09Z"), "CreatedAt", t)
	assert(reqMessage.UpdatedAt, toTime("2020-11-04T16:12:09Z"), "UpdatedAt", t)
	assert(reqMessage.Input.ResponseFormat, "tar", "Input.ResponseFormat", t)
	assert(reqMessage.Input.UploadURL, "/ingress/upload", "Input.UploadURL", t)
	assert(reqMessage.Input.Jobs[0].Method, "monitor", "Job.Method", t)
	assert(reqMessage.Input.Jobs[0].HrefSlug, "/api/v2/jobs/7008", "Job.HrefSlug", t)
}
