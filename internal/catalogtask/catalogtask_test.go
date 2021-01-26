package catalogtask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
)

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
	if assert.NoError(t, err) {
		assert.Equal(t, "12345", reqMessage.ID, "ID")
		assert.Equal(t, "pending", reqMessage.State, "State")
		assert.Equal(t, "unknown", reqMessage.Status, "Status")
		assert.Equal(t, toTime("2020-11-04T16:12:09Z"), reqMessage.CreatedAt, "CreateAt")
		assert.Equal(t, toTime("2020-11-04T16:12:09Z"), reqMessage.UpdatedAt, "UpdatedAt")
		assert.Equal(t, "tar", reqMessage.Input.ResponseFormat, "Input.ResponseFormat")
		assert.Equal(t, "/ingress/upload", reqMessage.Input.UploadURL, "Input.UploadURL")
		assert.Equal(t, "monitor", reqMessage.Input.Jobs[0].Method, "Job.Method")
		assert.Equal(t, "/api/v2/jobs/7008", reqMessage.Input.Jobs[0].HrefSlug, "Job.HrefSlug")
	}
}

func TestGetBad(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	task := MakeCatalogTask(logger.CtxWithLoggerID(context.Background(), 123), ts.URL)
	_, err := task.Get()

	if assert.Error(t, err, "Func Get") {
		assert.True(t, strings.Contains(err.Error(), "Invalid HTTP Status code"))
	}
}

func TestUpdate(t *testing.T) {
	data := map[string]interface{}{"state": "completed", "status": "ok"}
	retCode := http.StatusNoContent
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(retCode)
	}))

	task := MakeCatalogTask(logger.CtxWithLoggerID(context.Background(), 123), ts.URL)
	err := task.Update(data)
	assert.NoError(t, err, "Func Update")

	retCode = http.StatusBadGateway
	err = task.Update((data))
	if assert.Error(t, err, "Func Update") {
		assert.True(t, strings.Contains(err.Error(), "Invalid HTTP Status code"))
	}
}
