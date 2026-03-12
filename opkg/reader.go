package opkg

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/lunixbochs/struc"
	"github.com/platform-engineering-labs/orbital/opkg/payload"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Reader struct {
	path     string
	workPath string

	Header   *Header
	Manifest *ops.Manifest
	Payload  *payload.Reader
}

func NewReader(path string, workPath string) *Reader {
	reader := &Reader{}
	reader.path = path
	reader.workPath = workPath
	reader.Manifest = &ops.Manifest{}
	return reader
}

func (r *Reader) Read() error {
	file, err := os.Open(r.path)
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if r.workPath == "" {
		r.workPath = wd
	}

	reader := bufio.NewReader(file)

	r.Header = &Header{}
	err = struc.Unpack(reader, r.Header)
	if err != nil {
		return err
	}

	// Check magic
	if string(r.Header.Magic[:6]) != Magic {
		return errors.New("does not appear to be an opkg file")
	}

	var cManifestBytes bytes.Buffer
	_, err = io.CopyN(&cManifestBytes, reader, int64(r.Header.ManifestLength))
	if err != nil {
		return err
	}

	zstdReader, err := zstd.NewReader(&cManifestBytes)
	if err != nil {
		return err
	}

	var manifestBytes bytes.Buffer
	writer := io.Writer(&manifestBytes)

	_, err = io.Copy(writer, zstdReader)
	if err != nil {
		return err
	}

	err = r.Manifest.Load(manifestBytes.Bytes())
	if err != nil {
		return err
	}

	_ = file.Close()

	// TODO get byte size of header instead of just setting it
	offset := int64(r.Header.ManifestLength + 12)
	r.Payload = payload.NewReader(r.workPath, r.path, offset)

	return err
}

func (r *Reader) Close() {
	r.Payload.Close()
}
