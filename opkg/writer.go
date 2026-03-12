package opkg

import (
	"bufio"
	"bytes"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/lunixbochs/struc"
	"github.com/platform-engineering-labs/orbital/opkg/payload"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Writer struct{}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) Write(filename string, header *Header, manifest *ops.Manifest, payload *payload.Writer) error {
	var manifestBuffer bytes.Buffer

	// compress manifest
	zsw, _ := zstd.NewWriter(&manifestBuffer)

	if _, err := zsw.Write(manifest.ToJson()); err != nil {
		return err
	}

	if err := zsw.Close(); err != nil {
		return err
	}

	// Finalize Header
	header.ManifestLength = uint32(manifestBuffer.Len())

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)

	// Writer header
	_ = struc.Pack(writer, header)
	_ = writer.Flush()

	// Write manifest
	_, _ = writer.Write(manifestBuffer.Bytes())
	_ = writer.Flush()

	// Finish Payload
	if payload.HasContents() {
		payloadName := payload.Name()
		payload.Close()

		// Copy Payload to zpkg file
		payloadTmpFile, err := os.Open(payloadName)
		if err != nil {
			return err
		}

		reader := bufio.NewReader(payloadTmpFile)
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
		_ = writer.Flush()

		_ = payloadTmpFile.Close()
		_ = file.Close()

		// Cleanup
		err = os.Remove(payloadName)
		if err != nil {
			return err
		}
	}

	return err
}
