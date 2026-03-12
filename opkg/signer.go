package opkg

import (
	"bufio"
	"crypto/rsa"
	"io"
	"os"

	"github.com/platform-engineering-labs/orbital/opkg/payload"
	"github.com/platform-engineering-labs/orbital/opm/security"
)

type Signer struct {
	reader   *Reader
	workPath string
}

func NewSigner(path string, workPath string) *Signer {
	signer := &Signer{}
	signer.reader = NewReader(path, workPath)
	signer.workPath = workPath

	return signer
}

func (s *Signer) Sign(ski, fingerprint string, key *rsa.PrivateKey) error {
	err := s.reader.Read()
	if err != nil {
		return err
	}

	content := s.reader.Manifest.ToSigningJson()

	// Get signature action
	sigAction, err := security.SignBytes(&content, ski, fingerprint, key, "sha256")
	if err != nil {
		return err
	}

	// Modify manifest
	manifest := s.reader.Manifest
	manifest.Add(sigAction)

	payloadOffset := s.reader.Payload.Offset()
	s.reader.Close()

	// Write new header and manifest
	writer := NewWriter()
	tmpFile := s.reader.path + ".signing"

	err = writer.Write(tmpFile, NewHeader(Version, Compression), manifest, payload.NewWriter("", 0))
	if err != nil {
		return err
	}

	// Copy payload to temp file

	source, err := os.Open(s.reader.path)
	if err != nil {
		return err
	}
	_, err = source.Seek(payloadOffset, 0)
	if err != nil {
		return err
	}

	srcReader := bufio.NewReader(source)

	dest, err := os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}

	_, err = dest.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	destWriter := bufio.NewWriter(dest)

	_, err = io.Copy(destWriter, srcReader)
	if err != nil {
		return err
	}

	destWriter.Flush()
	source.Close()
	dest.Close()

	// TODO Probably doesn't work cross mount

	os.Remove(s.reader.path)
	os.Rename(tmpFile, s.reader.path)

	return nil
}
