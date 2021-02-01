package logger

import (
	"bytes"
	"context"
	"os"
	"regexp"
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
	res, _ := regexp.MatchString(`level=info msg="\[\S*\] \[id=101] hello work!`, logs)
	if !res {
		t.Error("Failed with Info()")
	}
	res, _ = regexp.MatchString(`level=info msg="\[\S*\] \[id=101] flag is true`, logs)
	if !res {
		t.Error("Failed with Infof()")
	}
	res, _ = regexp.MatchString(`level=error msg="\[\S*\] \[id=101] unexpected happens`, logs)
	if !res {
		t.Error("Failed with Error()")
	}
	res, _ = regexp.MatchString(`level=error msg="\[\S*\] \[id=101] error bad input`, logs)
	if !res {
		t.Error("Failed with Errorf()")
	}
	res, _ = regexp.MatchString(`level=info msg="\[\S*\] \[id=101] This is number 100`, logs)
	if !res {
		t.Error("Failed with Printf()")
	}
	res, _ = regexp.MatchString(`level=info msg="\[\S*\] \[id=101] One line`, logs)
	if !res {
		t.Error("Failed with Println()")
	}
}
