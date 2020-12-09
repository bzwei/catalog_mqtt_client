package logger

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestGetLogger2(t *testing.T) {
	defer func() {
		log.SetOutput(os.Stdout)
	}()

	var strBuf bytes.Buffer

	log.SetOutput(&strBuf)
	logger := GetLogger(CtxWithLoggerID(context.Background(), 101))
	logger.Info("hello work!")
	logger.Infof("flag is %v", true)
	logger.Error("unexpected happens")
	logger.Errorf("error %s", "bad input")
	logger.Printf("This is number %d", 100)
	logger.Println("One line")

	logs := strBuf.String()
	if !strings.Contains(logs, `level=info msg="[id=101] hello work!`) {
		t.Error("Failed with Info()")
	}
	if !strings.Contains(logs, `level=info msg="[id=101] flag is true`) {
		t.Error("Failed with Infof()")
	}
	if !strings.Contains(logs, `level=error msg="[id=101] unexpected happens`) {
		t.Error("Failed with Error()")
	}
	if !strings.Contains(logs, `level=error msg="[id=101] error bad input`) {
		t.Error("Failed with Errorf()")
	}
	if !strings.Contains(logs, `level=info msg="[id=101] This is number 100`) {
		t.Error("Failed with Printf()")
	}
	if !strings.Contains(logs, `level=info msg="[id=101] One line`) {
		t.Error("Failed with Println()")
	}
}
