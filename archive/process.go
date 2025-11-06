package archive

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type ProcessFileFn = func(name string, w io.Writer, r io.Reader) error

func Process(w io.Writer, r io.Reader, contentType string, processFile ProcessFileFn) error {
	switch contentType {
	case "":
		return errors.New("Content-Type not specified")
	case "application/zip":
		return ProcessZip(w, r, processFile)
	default:
		return fmt.Errorf("Content-Type '%s' not supported as an archive", contentType)

	}
}

func ProcessZip(w io.Writer, r io.Reader, processFile ProcessFileFn) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "failed to read data")
	}

	if len(data) == 0 {
		return errors.New("no data")
	}

	reader := bytes.NewReader(data)
	zr, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return errors.Wrap(err, "failed to read data as .zip archive")
	}

	buf := bytes.Buffer{}
	zw := zip.NewWriter(&buf)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.DefaultCompression)
	})

	for _, meta := range zr.File {
		fileReader, err := zr.Open(meta.Name)
		if err != nil {
			return errors.Wrapf(err, "could not open '%s'", meta.Name)
		}

		fileWriter, err := zw.Create(meta.Name)
		if err != nil {
			return errors.Wrapf(err, "could not create file '%s' in new archive", meta.Name)
		}

		err = processFile(meta.Name, fileWriter, fileReader)
		if err != nil {
			return errors.Wrap(err, "failed to process file")
		}
	}

	err = zw.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close/flush resulting archive")
	}

	_, err = io.Copy(w, &buf)
	return err
}
