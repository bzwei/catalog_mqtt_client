package request

import (
	"context"
	"net/http"
	"testing"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
)

type fakeHandler struct {
	timesCalled int
}

func (fh *fakeHandler) StartWork(ctx context.Context, config *common.CatalogConfig, params common.JobParam, client *http.Client, wc towerapiworker.WorkChannels) error {
	fh.timesCalled++
	return nil
}

type fakeCatalogTask struct{}

func (task *fakeCatalogTask) Get() (*common.RequestMessage, error) {
	message := common.RequestMessage{
		ID:     "12345",
		State:  "pending",
		Status: "ok",
		Input: common.RequestInput{
			ResponseFormat: "tar",
			Jobs: []common.JobParam{
				common.JobParam{Method: "monitor", HrefSlug: "/api/v2/jobs/7008"},
				common.JobParam{Method: "get", HrefSlug: "/api/v2/inventories/899"},
			},
		},
	}
	return &message, nil
}
func (task *fakeCatalogTask) Update(data map[string]interface{}) error { return nil }

type fakePageWriter struct{}

func (pw *fakePageWriter) Write(name string, b []byte) error { return nil }
func (pw *fakePageWriter) Flush() error                      { return nil }
func (pw *fakePageWriter) FlushErrors(msg []string) error    { return nil }

type fakePageWriterFactory struct{}

func (factory *fakePageWriterFactory) makePageWriter(ctx context.Context, format string, uploadURL string, task catalogtask.CatalogTask, metadata map[string]string) (PageWriter, error) {
	return &fakePageWriter{}, nil
}

func TestProcessRequest(t *testing.T) {
	fh := fakeHandler{}
	ct := fakeCatalogTask{}
	pwf := fakePageWriterFactory{}
	shutdown := make(chan struct{})
	processRequest(logger.CtxWithLoggerID(context.Background(), 123), "testurl", &common.CatalogConfig{}, &fh, &ct, &pwf, shutdown)
	if fh.timesCalled != 2 {
		t.Fatalf("2 workers should have been started only %d were started", fh.timesCalled)
	}
}
