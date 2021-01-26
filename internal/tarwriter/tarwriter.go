package tarwriter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/tarfiles"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/upload"
)

type tarWriter struct {
	dir       string
	task      catalogtask.CatalogTask
	uploadURL string
	ctx       context.Context
	glog      logger.Logger
	metadata  map[string]string
}

// MakeTarWriter creates a common.PageWriter that zip data as a tar file and upload to an URL.
func MakeTarWriter(ctx context.Context, task catalogtask.CatalogTask, uploadURL string, metadata map[string]string) (common.PageWriter, error) {
	glog := logger.GetLogger(ctx)
	t := tarWriter{}
	dir, err := ioutil.TempDir("", "catalog_client")
	if err != nil {
		glog.Errorf("Error creating temp directory %v", err)
		return nil, err
	}
	t.dir = dir
	t.task = task
	t.uploadURL = uploadURL
	t.ctx = ctx
	t.glog = glog
	t.metadata = metadata
	return &t, nil
}

// Write a Page given the name and the number of bytes to write
func (tw *tarWriter) Write(name string, b []byte) error {
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

func (tw *tarWriter) Flush() error {
	var statusErrors []string
	defer func() {
		if len(statusErrors) > 0 {
			tw.FlushErrors(statusErrors)
		}
	}()

	tmpdir, err := ioutil.TempDir("", "catalog_client_tgz")
	if err != nil {
		tw.glog.Errorf("Error creating temp directory %v", err)
		statusErrors = append(statusErrors, "Failed to create temp directory for the tar file creation")
		return err
	}
	fname := filepath.Join(tmpdir, "inventory.tgz")
	sha, err := tarfiles.TarCompressDirectory(tw.dir, fname)
	if err != nil {
		tw.glog.Errorf("Error compressing directory %s %v", tw.dir, err)
		statusErrors = append(statusErrors, "Failed to compress directory to a tar file")
		return err
	}
	info, _ := os.Stat(fname)

	b, uploadErr := upload.Upload(tw.uploadURL, fname, "application/vnd.redhat.topological-inventory.filename+tgz", tw.metadata)
	os.RemoveAll(tw.dir)
	os.RemoveAll(tmpdir)
	if uploadErr != nil {
		tw.glog.Errorf("Error uploading file %s %v", fname, uploadErr)
		statusErrors = append(statusErrors, "Failed to upload the tar file")
		return uploadErr
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		tw.glog.Errorf("Unmarshaling byte array for %v", err)
		statusErrors = append(statusErrors, "Failed to unmarshal the body of uploading API call")
		return err
	}

	output := map[string]interface{}{"ingress": m, "sha256": sha, "tar_size": info.Size()}

	err = tw.task.Update(map[string]interface{}{"state": "completed", "status": "ok", "output": &output})

	if err != nil {
		tw.glog.Errorf("Error updating task: %v", err)
		return err
	}
	return nil
}

func (tw *tarWriter) FlushErrors(messages []string) error {
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
