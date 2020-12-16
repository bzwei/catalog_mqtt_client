package jsonwriter

import (
	"context"
	"encoding/json"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
)

type JSONWriter struct {
	task catalogtask.CatalogTask
	glog logger.Logger
	ctx  context.Context
}

func MakeJSONWriter(ctx context.Context, task catalogtask.CatalogTask) *JSONWriter {
	glog := logger.GetLogger(ctx)

	return &JSONWriter{task: task, glog: glog, ctx: ctx}
}

// Write a Page given the name and the number of bytes to write
func (jw *JSONWriter) Write(name string, b []byte) error {
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

func (jw *JSONWriter) Flush() error {
	err := jw.task.Update(map[string]interface{}{"state": "completed", "status": "ok"})
	if err != nil {
		jw.glog.Errorf("Error updating task: %v", err)
	}
	return err
}

func (jw *JSONWriter) FlushErrors(messages []string) error {
	msg := map[string]interface{}{
		"messages": messages,
	}
	err := jw.task.Update(map[string]interface{}{"state": "completed", "status": "error", "output": &msg})
	if err != nil {
		jw.glog.Errorf("Error updating task: %v", err)
	}
	return err
}
