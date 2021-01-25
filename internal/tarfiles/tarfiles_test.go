package tarfiles

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTarCompressDirectory(t *testing.T) {
	dirName, tarName := "testtarfiles", "testtarfiles.tar.gz"
	defer os.RemoveAll(dirName)
	defer os.Remove(tarName)

	assert.NoError(t, os.Mkdir(dirName, 0755))

	for _, fname := range []string{"file1", "file2"} {
		data := []byte(strings.Repeat("na", 512))
		f, err := os.Create(dirName + "/" + fname)
		assert.NoError(t, err)
		f.Write(data)
		f.Close()
	}

	sha, err := TarCompressDirectory(dirName, tarName)
	assert.NoError(t, err)
	info, err := os.Stat(tarName)
	assert.NoError(t, err)
	assert.Equal(t, int64(158), info.Size(), "Tar file size")
	assert.Equal(t, "986859044806a7e6cad8adeda2740cb081ebf895673c9bf9d3dbbf344d419bd5", sha, "Sha of tar file")
}
