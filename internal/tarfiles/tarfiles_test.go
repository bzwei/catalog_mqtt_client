package tarfiles

import (
	"os"
	"strings"
	"testing"
)

func check(e error, t *testing.T) {
	if e != nil {
		t.Error(e)
	}
}

func TestTarCompressDirectory(t *testing.T) {
	dirName, tarName := "testtarfiles", "testtarfiles.tar.gz"
	defer os.RemoveAll(dirName)
	defer os.Remove(tarName)

	check(os.Mkdir(dirName, 0755), t)

	for _, fname := range []string{"file1", "file2"} {
		data := []byte(strings.Repeat("na", 512))
		f, err := os.Create(dirName + "/" + fname)
		check(err, t)
		f.Write(data)
		f.Close()
	}

	check(TarCompressDirectory(dirName, tarName), t)
	info, err := os.Stat(tarName)
	check(err, t)

	// test the size is 172 or 173
	if info.Size()/10 != 17 {
		t.Error("tarfile size does not match expectation. reported size =", info.Size(), "expected 172")
	}
}
