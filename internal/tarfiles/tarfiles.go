package tarfiles

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func TarCompressDirectory(dir string, outfile string) error {

	f, err := os.Create(outfile)
	if err != nil {
		log.Errorf("Error creating file %s", outfile)
		return err
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
		hdr.Name = filepath.ToSlash(path)
		hdr.Name = strings.TrimPrefix(hdr.Name, dir)
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
		return err
	}

	if err := tw.Close(); err != nil {
		log.Errorf("Error closing tar file")
		return err
	}
	if err := zw.Close(); err != nil {
		log.Errorf("Error closing compressed file")
		return err
	}

	if err := f.Close(); err != nil {
		log.Errorf("Error closing file")
		return err
	}

	return nil
}
