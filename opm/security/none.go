package security

import (
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/ops"
)

const (
	None Mode = "none"
)

type SecurityNone struct{}

func (s *SecurityNone) Mode() Mode {
	return None
}

func (s *SecurityNone) KeyPair(publisher string) (*pki.KeyPairEntry, error) {
	return nil, nil
}

func (s *SecurityNone) Refresh() error { return nil }
func (s *SecurityNone) Resolve(ski string, publisher string) (*pki.CertEntry, error) {
	return nil, nil
}

func (s *SecurityNone) Trust(content *[]byte) (*CertMetadata, error) {
	return nil, nil
}

func (s *SecurityNone) VerifyManifest(manifest *ops.Manifest) ([]*VerifyResult, error) {
	return nil, nil
}

func (s *SecurityNone) VerifyMetadata(metadata *metadata.Metadata, publisher string) ([]*VerifyResult, error) {
	return nil, nil
}
