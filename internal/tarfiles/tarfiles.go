package tarfiles

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// TarCompressDirectory compresses the whole directory into an output tar file and returns the sha256 of the tarfile
func TarCompressDirectory(dir string, outfile string) (string, error) {
	f, err := os.Create(outfile)
	if err != nil {
		log.Errorf("Error creating file %s", outfile)
		return "", err
	}

	zw := gzip.NewWriter(f)
	tw := tar.NewWriter(zw)

	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("Failure accessing a path %q: %v", path, err)
			return err
		}

		hdr, err := tar.FileInfoHeader(info, path)
		if err != nil {
			log.Errorf("Error creating file info header")
			return err
		}
		hdr.AccessTime = time.Unix(0, 0)
		hdr.ChangeTime = time.Unix(0, 0)
		hdr.ModTime = time.Unix(0, 0)
		hdr.Name = filepath.ToSlash(path)
		hdr.Name = strings.TrimPrefix(hdr.Name, dir)
		hdr.Uid = 0
		hdr.Uname = "unknown"
		hdr.Gid = 0
		hdr.Gname = "unknown"
		if hdr.Name == "" {
			hdr.Name = "./"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			log.Errorf("Error writing header")
			return err
		}

		if !info.IsDir() {
			source, err := os.Open(path)
			if err != nil {
				log.Errorf("Error opening file %q", path)
				return err
			}
			defer source.Close()

			_, err = io.Copy(tw, source)
			if err != nil {
				log.Errorf("Error copying file bytes")
				return err
			}
		}
		return nil
	}

	err = filepath.Walk(dir, fn)
	if err != nil {
		log.Errorf("error walking directory %v", err)
		return "", err
	}

	if err := tw.Close(); err != nil {
		log.Errorf("Error closing tar file")
		return "", err
	}
	if err := zw.Close(); err != nil {
		log.Errorf("Error closing compressed file")
		return "", err
	}

	f.Seek(0, 0)
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)

	if err := f.Close(); err != nil {
		log.Errorf("Error closing file")
		return "", err
	}

	return fmt.Sprintf("%x", sum), nil
}
