package tarwriter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/tarfiles"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/upload"
)

type TarWriter struct {
	dir       string
	task      catalogtask.CatalogTask
	uploadUrl string
	ctx       context.Context
	glog      logger.Logger
	metadata  map[string]string
}

func MakeTarWriter(ctx context.Context, task catalogtask.CatalogTask, uploadUrl string, metadata map[string]string) (*TarWriter, error) {
	glog := logger.GetLogger(ctx)
	t := TarWriter{}
	dir, err := ioutil.TempDir("", "catalog_client")
	if err != nil {
		glog.Errorf("Error creating temp directory %v", err)
		return nil, err
	}
	t.dir = dir
	t.task = task
	t.uploadUrl = uploadUrl
	t.ctx = ctx
	t.glog = glog
	t.metadata = metadata
	return &t, nil
}

// Write a Page given the name and the number of bytes to write
func (tw *TarWriter) Write(name string, b []byte) error {
	baseDir := filepath.Join(tw.dir, filepath.Dir(name))
	os.MkdirAll(baseDir, os.ModePerm)
	tw.glog.Infof("adding file %s", filepath.Join(tw.dir, name))
	err := ioutil.WriteFile(filepath.Join(tw.dir, name), b, 0644)
	if err != nil {
		tw.glog.Errorf("Error writing file %s %v", name, err)
		return err
	}
	return nil
}

func (tw *TarWriter) Flush() error {
	tmpdir, err := ioutil.TempDir("", "catalog_client_tgz")
	if err != nil {
		tw.glog.Errorf("Error creating temp directory %v", err)
		return err
	}
	fname := filepath.Join(tmpdir, "inventory.tgz")
	err = tarfiles.TarCompressDirectory(tw.dir, fname)
	if err != nil {
		tw.glog.Errorf("Error compressing directory %s %v", tw.dir, err)
	}

	b, uploadErr := upload.Upload(tw.uploadUrl, fname, "application/vnd.redhat.topological-inventory.filename+tgz", tw.metadata)
	os.RemoveAll(tw.dir)
	os.RemoveAll(tmpdir)
	if uploadErr != nil {
		tw.glog.Errorf("Error uploading file %s %v", fname, uploadErr)
		return uploadErr
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		tw.glog.Errorf("Unmarshaling byte array for %v", err)
		return err
	}

	var status string
	if uploadErr == nil {
		status = "ok"
	} else {
		status = "error"
		m = map[string]interface{}{"message": uploadErr.Error()}
	}
	err = tw.task.Update(map[string]interface{}{"state": "completed", "status": status, "output": &m})

	if err != nil {
		tw.glog.Errorf("Error updating task: %v", err)
		return err
	}
	return nil
}

func (tw *TarWriter) FlushErrors(messages []string) error {
	os.RemoveAll(tw.dir)
	msg := map[string]interface{}{
		"errors": messages,
	}
	err := tw.task.Update(map[string]interface{}{"state": "completed", "status": "error", "output": &msg})
	if err != nil {
		tw.glog.Errorf("Error updating task: %v", err)
		return err
	}
	return nil
}
