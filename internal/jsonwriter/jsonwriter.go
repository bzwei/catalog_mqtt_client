package jsonwriter

import (
	"context"
	"encoding/json"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/catalogtask"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
)

type jsonWriter struct {
	task catalogtask.CatalogTask
	glog logger.Logger
	ctx  context.Context
}

// MakeJSONWriter creates a common.PageWriter that writes and flushes JSON type data
func MakeJSONWriter(ctx context.Context, task catalogtask.CatalogTask) common.PageWriter {
	glog := logger.GetLogger(ctx)

	return &jsonWriter{task: task, glog: glog, ctx: ctx}
}

// Write a Page given the name and the number of bytes to write
func (jw *jsonWriter) Write(name string, b []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		jw.glog.Errorf("Error unmarshaling byte array: %v", err)
		return err
	}
	err = jw.task.Update(map[string]interface{}{"state": "running", "status": "ok", "output": &m})
	if err != nil {
		jw.glog.Errorf("Error updating task: %v", err)
	}
	return err
}

// Flush updates the task to completed state
func (jw *jsonWriter) Flush() error {
	err := jw.task.Update(map[string]interface{}{"state": "completed", "status": "ok", "message": "Catalog Worker Ended Successfully"})
	if err != nil {
		jw.glog.Errorf("Error updating task: %v", err)
	}
	return err
}

// FlushErrors updates the task to completed state with given error messages
func (jw *jsonWriter) FlushErrors(messages []string) error {
	msg := map[string]interface{}{
		"errors": messages,
	}
	err := jw.task.Update(map[string]interface{}{"state": "completed", "status": "error", "output": &msg, "message": "Catalog Worker Ended with errors"})
	if err != nil {
		jw.glog.Errorf("Error updating task: %v", err)
	}
	return err
}
