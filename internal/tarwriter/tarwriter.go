package tarwriter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mkanoor/catalog_mqtt_client/internal/logger"
	"github.com/mkanoor/catalog_mqtt_client/internal/tarfiles"
	"github.com/mkanoor/catalog_mqtt_client/internal/taskupdater"
	"github.com/mkanoor/catalog_mqtt_client/internal/upload"
)

type TarWriter struct {
	dir       string
	Url       string
	uploadUrl string
	ctx       context.Context
	glog      logger.Logger
}

func MakeTarWriter(ctx context.Context, url string, uploadUrl string) (*TarWriter, error) {
	glog := logger.GetLogger(ctx)
	t := TarWriter{}
	dir, err := ioutil.TempDir("", "catalog_client")
	if err != nil {
		glog.Errorf("Error creating temp directory %v", err)
		return nil, err
	}
	t.dir = dir
	t.Url = url
	t.uploadUrl = uploadUrl
	t.ctx = ctx
	t.glog = glog
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

	tu := taskupdater.MakeTaskUpdater(tw.ctx, tw.Url)
	//_, err = upload.Upload(tw.uploadUrl, fname, "application/vnd.redhat.catalog.filename+tgz")
	b, uploadErr := upload.Upload(tw.uploadUrl, fname, "application/vnd.redhat.topological-inventory.filename+tgz")
	if uploadErr != nil {
		tw.glog.Errorf("Error uploading file %s %v", fname, uploadErr)
	}
	os.RemoveAll(tw.dir)
	os.RemoveAll(tmpdir)
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		tw.glog.Errorf("Unmarshaling byte array for %v", err)
		return err
	}

	if uploadErr == nil {
		_, err = tu.Do("completed", "ok", &m)
	} else {
		_, err = tu.Do("completed", "error", &map[string]interface{}{"message": uploadErr.Error()})
	}

	if err != nil {
		tw.glog.Errorf("Error updating task %s %v", tw.Url, err)
		return err
	}
	return nil
}

func (tw *TarWriter) FlushErrors(messages []string) error {
	os.RemoveAll(tw.dir)
	tu := taskupdater.TaskUpdater{Url: tw.Url}
	msg := map[string]interface{}{
		"messages": messages,
	}
	_, err := tu.Do("completed", "error", &msg)
	if err != nil {
		tw.glog.Errorf("Error updating task %s %v", tw.Url, err)
		return err
	}
	return nil
}
