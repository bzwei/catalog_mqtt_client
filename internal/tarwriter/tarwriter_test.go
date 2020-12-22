package tarwriter

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
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

func shareWriteTest(t *testing.T, httpStatus int, body string) (*httptest.Server, *tarWriter) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(httpStatus)
		w.Write([]byte(body))
	}))

	task := new(mockCatalogTask)
	twriter, err := MakeTarWriter(logger.CtxWithLoggerID(context.Background(), 123), task, ts.URL, map[string]string{"task_url": "taskURL"})
	err = twriter.Write("testpage", []byte(strings.Repeat("na", 512)))
	assert.NoError(t, err)

	// tar directory exists and contains 1 file
	writerObj := twriter.(*tarWriter)
	if assert.DirExists(t, writerObj.dir) {
		files, _ := ioutil.ReadDir(writerObj.dir)
		assert.Equal(t, 1, len(files))
	}

	return ts, writerObj
}

func shareFlushTest(t *testing.T, writerObj *tarWriter, output *map[string]interface{}, status string, errMsg string) {
	task := writerObj.task.(*mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "completed", "status": status, "output": output}).Return(nil)

	err := writerObj.Flush()
	task.AssertExpectations(t)
	if errMsg == "" {
		assert.NoError(t, err)
	} else {
		if assert.Error(t, err) {
			assert.True(t, strings.Contains(err.Error(), errMsg))
		}
	}

	// tar directory removed
	_, err = os.Stat(writerObj.dir)
	assert.False(t, os.IsExist(err))
}

func TestWriteAndFlush(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusAccepted, `{"upload":"accepted"}`)
	defer ts.Close()

	shareFlushTest(t, writerObj, &map[string]interface{}{"upload": "accepted"}, "ok", "")
}

func TestUploadFailed(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusNotFound, "")
	defer ts.Close()

	shareFlushTest(t, writerObj, &map[string]interface{}{"errors": []string{"Failed to upload the tar file"}}, "error", "Upload failed")
}

func TestUnmarshalFailed(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusAccepted, `bad{"upload":"accepted"}`)
	defer ts.Close()

	shareFlushTest(t, writerObj, &map[string]interface{}{"errors": []string{"Failed to unmarshal the body of uploading API call"}}, "error", "invalid character")
}

func TestFlushError(t *testing.T) {
	task := new(mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "completed", "status": "error", "output": &map[string]interface{}{"errors": []string{"error 1", "error 2"}}}).Return(nil)
	twriter, err := MakeTarWriter(logger.CtxWithLoggerID(context.Background(), 123), task, "uploadURL", map[string]string{"task_url": "taskURL"})
	err = twriter.FlushErrors([]string{"error 1", "error 2"})

	task.AssertExpectations(t)
	assert.NoError(t, err)
}
