package tarwriter

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCatalogTask struct {
	mock.Mock
}

func (m *mockCatalogTask) Get() (*common.CatalogInventoryTask, error) { return nil, nil }

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
	twriter, _ := MakeTarWriter(logger.CtxWithLoggerID(context.Background(), "123"), task, common.RequestInput{UploadURL: ts.URL}, map[string]string{"task_url": "taskURL"})
	shareWriteOperation(t, twriter)

	return ts, twriter.(*tarWriter)
}

func shareWriteOperation(t *testing.T, writer common.PageWriter) {
	err := writer.Write("testpage", []byte(strings.Repeat("na", 512)))
	assert.NoError(t, err)

	// tar directory exists and contains 1 file
	writerObj := writer.(*tarWriter)
	if assert.DirExists(t, writerObj.dir) {
		files, _ := ioutil.ReadDir(writerObj.dir)
		assert.Equal(t, 1, len(files))
	}
}

func shareFlushTest(t *testing.T, writerObj *tarWriter, output *map[string]interface{}, status string, errMsg string, message string) {
	task := writerObj.task.(*mockCatalogTask)
	if output != nil {
		task.On("Update", map[string]interface{}{"state": "completed", "status": status, "output": output, "message": message}).Return(nil)
	} else {
		task.On("Update", map[string]interface{}{"state": "completed", "status": status, "message": message}).Return(nil)
	}

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
	assert.True(t, !os.IsExist(err))
}

func TestWriteAndFlush(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusAccepted, `{"upload":"accepted"}`)
	defer ts.Close()

	output := map[string]interface{}{
		"ingress":  map[string]interface{}{"upload": "accepted"},
		"sha256":   "8d97f6ddad8fb21b41a2d97079fbb371e590fc5c4afb9556faa9de1ba025d84c",
		"tar_size": int64(134),
	}
	shareFlushTest(t, writerObj, &output, "ok", "", "Catalog Worker Completed Successfully")
}

func TestNoUpload(t *testing.T) {
	task := new(mockCatalogTask)
	input := common.RequestInput{PreviousSHA: "8d97f6ddad8fb21b41a2d97079fbb371e590fc5c4afb9556faa9de1ba025d84c", PreviousSize: int64(134)}
	writer, _ := MakeTarWriter(logger.CtxWithLoggerID(context.Background(), "123"), task, input, map[string]string{"task_url": "taskURL"})
	shareWriteOperation(t, writer)

	shareFlushTest(t, writer.(*tarWriter), nil, "unchanged", "", "Upload skipped since nothing has changed from last refresh")
}

func TestUploadFailed(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusNotFound, "")
	defer ts.Close()

	shareFlushTest(t, writerObj, &map[string]interface{}{"errors": []string{"Failed to upload the tar file"}}, "error", "Upload failed", "Catalog Worker Ended with errors")
}

func TestUnmarshalFailed(t *testing.T) {
	ts, writerObj := shareWriteTest(t, http.StatusAccepted, `bad{"upload":"accepted"}`)
	defer ts.Close()

	shareFlushTest(t, writerObj, &map[string]interface{}{"errors": []string{"Failed to unmarshal the body of uploading API call"}}, "error", "invalid character", "Catalog Worker Ended with errors")
}

func TestFlushError(t *testing.T) {
	task := new(mockCatalogTask)
	task.On("Update", map[string]interface{}{"state": "completed", "status": "error", "output": &map[string]interface{}{"errors": []string{"error 1", "error 2"}}, "message": "Catalog Worker Ended with errors"}).Return(nil)
	twriter, err := MakeTarWriter(logger.CtxWithLoggerID(context.Background(), "123"), task, common.RequestInput{UploadURL: "uploadURL"}, map[string]string{"task_url": "taskURL"})
	err = twriter.FlushErrors([]string{"error 1", "error 2"})

	task.AssertExpectations(t)
	assert.NoError(t, err)
}
