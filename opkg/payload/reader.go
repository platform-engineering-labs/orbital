package payload

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

type Reader struct {
	WorkPath string
	Path     string

	offset int64
	file   *os.File
}

func NewReader(workPath string, path string, offset int64) *Reader {
	reader := &Reader{}

	reader.WorkPath = workPath
	reader.Path = path

	reader.offset = offset

	return reader
}

func (r *Reader) Get(path string, offset int64, size int64) (string, error) {
	var err error

	if r.file == nil {
		r.file, err = os.Open(r.Path)
		if err != nil {
			return "", err
		}
	}

	_, err = r.file.Seek(r.offset+offset, 0)
	if err != nil {
		return "", err
	}

	target, err := os.Create(path)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(r.file)

	var zstdReader *zstd.Decoder

	zstdReader, err = zstd.NewReader(reader, zstd.WithDecoderConcurrency(0))
	if err != nil {
		return "", err
	}

	writer := bufio.NewWriter(target)
	hasher := sha256.New()

	multi := io.MultiWriter(writer, hasher)

	_, err = io.CopyN(multi, zstdReader, size)

	if err != nil {
		return "", err
	}

	_ = writer.Flush()
	_ = target.Close()

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (r *Reader) Verify(offset int64, size int64) (string, error) {
	var err error

	if r.file == nil {
		r.file, err = os.Open(r.Path)
		if err != nil {
			return "", err
		}
	}

	_, err = r.file.Seek(r.offset+offset, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(r.file)

	var zstdReader *zstd.Decoder

	zstdReader, err = zstd.NewReader(reader, zstd.WithDecoderConcurrency(0))
	if err != nil {
		return "", err
	}

	hasher := sha256.New()

	_, err = io.CopyN(hasher, zstdReader, size)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (r *Reader) Offset() int64 {
	return r.offset
}

func (r *Reader) Close() {
	_ = r.file.Close()
}
