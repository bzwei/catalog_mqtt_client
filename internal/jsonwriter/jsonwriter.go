package jsonwriter

import (
	"context"
	"encoding/json"

	"github.com/mkanoor/catalog_mqtt_client/internal/logger"
	"github.com/mkanoor/catalog_mqtt_client/internal/taskupdater"
)

type JSONWriter struct {
	Url  string
	glog logger.Logger
	ctx  context.Context
}

func MakeJSONWriter(ctx context.Context, url string) *JSONWriter {
	glog := logger.GetLogger(ctx)

	return &JSONWriter{Url: url, glog: glog, ctx: ctx}
}

// Write a Page given the name and the number of bytes to write
func (jw *JSONWriter) Write(name string, b []byte) error {
	tu := taskupdater.TaskUpdater{Url: jw.Url}
	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		jw.glog.Errorf("Unmarshaling byte array for %s %v", jw.Url, err)
		return err
	}
	_, err = tu.Do("running", "ok", &m)
	if err != nil {
		jw.glog.Errorf("Error updating task %s %v", jw.Url, err)
		return err
	}
	return nil
}

func (jw *JSONWriter) Flush() error {
	tu := taskupdater.TaskUpdater{Url: jw.Url}
	_, err := tu.Do("completed", "ok", nil)
	if err != nil {
		jw.glog.Errorf("Error updating task %s %v", jw.Url, err)
		return err
	}
	return nil
}

func (jw *JSONWriter) FlushErrors(messages []string) error {
	tu := taskupdater.TaskUpdater{Url: jw.Url}
	msg := map[string]interface{}{
		"messages": messages,
	}
	_, err := tu.Do("completed", "error", &msg)
	if err != nil {
		jw.glog.Errorf("Error updating task %s %v", jw.Url, err)
		return err
	}
	return nil
}
