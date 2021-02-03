package jsonwriter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/RedHatInsights/rhc_catalog_worker/internal/common"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCatalogTask struct {
	mock.Mock
}

func (m *mockCatalogTask) Get() (*common.RequestMessage, error) { return nil, nil }

func (m *mockCatalogTask) Update(data map[string]interface{}) error {
	m.Called(data)
	return nil
}

func TestWrite(t *testing.T) {
	task := new(mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "running", "status": "ok", "output": &map[string]interface{}{"key1": "val1", "key2": "val2"}}).Return(nil)
	jwriter := MakeJSONWriter(logger.CtxWithLoggerID(context.Background(), 123), task)
	err := jwriter.Write("test page", []byte(`{"key1": "val1", "key2": "val2"}`))

	task.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestWriteError(t *testing.T) {
	jwriter := MakeJSONWriter(logger.CtxWithLoggerID(context.Background(), 123), nil)
	err := jwriter.Write("test page", []byte(`bad{"key1": "val1", "key2": "val2"}`))
	if assert.Error(t, err) {
		assert.IsType(t, &json.SyntaxError{}, err)
	}
}

func TestFlush(t *testing.T) {
	task := new(mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "completed", "status": "ok"}).Return(nil)
	jwriter := MakeJSONWriter(logger.CtxWithLoggerID(context.Background(), 123), task)
	err := jwriter.Flush()

	task.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestFlushError(t *testing.T) {
	task := new(mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "completed", "status": "error", "output": &map[string]interface{}{"errors": []string{"error 1", "error 2"}}}).Return(nil)
	jwriter := MakeJSONWriter(logger.CtxWithLoggerID(context.Background(), 123), task)
	err := jwriter.FlushErrors([]string{"error 1", "error 2"})

	task.AssertExpectations(t)
	assert.NoError(t, err)
}
