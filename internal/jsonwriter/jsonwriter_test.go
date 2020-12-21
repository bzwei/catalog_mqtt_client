package jsonwriter

import (
	"context"
	"fmt"
	"testing"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/testhelper"
)

type mockUpdate func(map[string]interface{}) error

var thisMockUpdate mockUpdate

type mockCatalogTask struct {
	numUpdate int
}

func (task *mockCatalogTask) Get() (*common.RequestMessage, error) { return nil, nil }

func (task *mockCatalogTask) Update(data map[string]interface{}) error {
	task.numUpdate++
	return thisMockUpdate(data)
}

func TestWrite(t *testing.T) {
	thisMockUpdate = func(data map[string]interface{}) error {
		// "state": "running", "status": "ok", "output": &
		testhelper.Assert(t, "state", "running", data["state"])
		testhelper.Assert(t, "status", "ok", data["status"])
		output := fmt.Sprintf("%v", data["output"])
		testhelper.Assert(t, "output", map[string]interface{}{"key1": "val1", "key2": "val2"}, output)
		return nil
	}
	catalogTask := mockCatalogTask{}
	jwriter := MakeJSONWriter(logger.CtxWithLoggerID(context.Background(), 123), &catalogTask)
	jwriter.Write("test page", []byte(`{"key1": "val1", "key2": "val2"}`))

	testhelper.Assert(t, "number of Update to be called", 1, catalogTask.numUpdate)
}

func TestFlush(t *testing.T) {

}

func TestFlushError(t *testing.T) {

}
