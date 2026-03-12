package security

import (
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Security interface {
	Mode() Mode
	VerifyManifest(manifest *ops.Manifest) ([]*VerifyResult, error)
	VerifyMetadata(metadata *metadata.Metadata, publisher string) ([]*VerifyResult, error)
	KeyPair(publisher string) (*pki.KeyPairEntry, error)
	Refresh() error
	Resolve(ski string, publisher string) (*pki.CertEntry, error)
	Trust(content *[]byte) (*CertMetadata, error)
}

func New(log *slog.Logger, mode Mode, store *pki.Pki) (Security, error) {
	// Short circuit for none
	if mode == None {
		return &SecurityNone{}, nil
	}

	// Setup initial PKI verify opts, loading CAs and Intermediates from pki store
	cas := x509.NewCertPool()
	intermediates := x509.NewCertPool()

	caEntries, err := store.Certificates.GetByType(pki.CertCA)
	if err != nil {
		return nil, err
	}

	intEntries, err := store.Certificates.GetByType(pki.CertIssuer)
	if err != nil {
		return nil, err
	}

	for _, crt := range caEntries {
		ok := cas.AppendCertsFromPEM(crt.Cert)
		if !ok {
			return nil, fmt.Errorf("%s loaded from pki db did not parse", crt.Fingerprint)
		}
	}

	for _, crt := range intEntries {
		ok := intermediates.AppendCertsFromPEM(crt.Cert)
		if !ok {
			return nil, fmt.Errorf("%s loaded from pki db did not parse", crt.Fingerprint)
		}
	}

	switch mode {
	case Default:
		return &SecurityDefault{log, store, cas, intermediates}, nil
	default:
		return nil, errors.New("security mode does not exist")
	}
}
