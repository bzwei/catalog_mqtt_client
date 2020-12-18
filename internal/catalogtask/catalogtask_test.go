package catalogtask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/testhelper"
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
	if err != nil {
		t.Fatalf("Error parsing request data %v", err)
	}

	testhelper.Assert(t, "ID", "12345", reqMessage.ID)
	testhelper.Assert(t, "State", "pending", reqMessage.State)
	testhelper.Assert(t, "Status", "unknown", reqMessage.Status)
	testhelper.Assert(t, "CreatedAt", toTime("2020-11-04T16:12:09Z"), reqMessage.CreatedAt)
	testhelper.Assert(t, "UpdatedAt", toTime("2020-11-04T16:12:09Z"), reqMessage.UpdatedAt)
	testhelper.Assert(t, "Input.ResponseFormat", "tar", reqMessage.Input.ResponseFormat)
	testhelper.Assert(t, "Input.UploadURL", "/ingress/upload", reqMessage.Input.UploadURL)
	testhelper.Assert(t, "Job.Method", "monitor", reqMessage.Input.Jobs[0].Method)
	testhelper.Assert(t, "Job.HrefSlug", "/api/v2/jobs/7008", reqMessage.Input.Jobs[0].HrefSlug)
}

func TestUpdate(t *testing.T) {
	data := map[string]interface{}{"state": "completed", "status": "ok"}
	retCode := http.StatusNoContent
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(retCode)
	}))

	task := MakeCatalogTask(logger.CtxWithLoggerID(context.Background(), 123), ts.URL)
	err := task.Update(data)
	testhelper.AssertErrorMessage(t, "Func Update", "Environmental variable X_RH_IDENTITY is not set", err)

	os.Setenv("X_RH_IDENTITY", "x-rh-id")
	defer os.Unsetenv("X_RH_IDENTITY")
	err = task.Update(data)
	testhelper.AssertNoError(t, "Func Update", err)

	retCode = http.StatusBadGateway
	err = task.Update((data))
	testhelper.AssertErrorMessage(t, "Func Update", "Invalid HTTP Status code", err)
}
