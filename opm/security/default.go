package security

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/ops"
)

const (
	Default Mode = "default"
)

type SecurityDefault struct {
	*slog.Logger
	pki *pki.Pki

	caCache           *x509.CertPool
	intermediateCache *x509.CertPool
}

func (s *SecurityDefault) Mode() Mode {
	return Default
}

func (s *SecurityDefault) KeyPair(publisher string) (*pki.KeyPairEntry, error) {
	pairs, err := s.pki.KeyPairs.GetByPublisher(publisher)
	if err != nil {
		return nil, err
	}

	if len(pairs) > 0 {
		return pairs[0], nil
	}

	return nil, nil
}

func (s *SecurityDefault) Trust(content *[]byte) (*CertMetadata, error) {
	var ctype pki.CertType

	metaData, err := CertMetadataFromBytes(content)
	if err != nil {
		return nil, err
	}

	// Attempt to detect cert type
	switch metaData.Type {
	case pki.CertCA:
		ctype = pki.CertCA
	case pki.CertIssuer:
		ctype = pki.CertIssuer
	case pki.CertUser:
		ctype = pki.CertUser
	}

	err = s.pki.Certificates.Put(metaData.SKI, metaData.Fingerprint, metaData.Subject, metaData.Publisher, ctype, *content)
	if err != nil {
		return nil, err
	}

	return metaData, nil
}

func (s *SecurityDefault) Refresh() error {
	certs, err := s.pki.Certificates.All()
	if err != nil {
		return err
	}

	for _, cert := range certs {
		update, err := DNS.LookupOPSCERT(cert.Type, &LookupRequest{
			SKI:       cert.SKI,
			Publisher: cert.Publisher,
		})
		if err != nil {
			s.Error(fmt.Sprintf("failed to refresh OPS Cert: %s", err))
			continue
		}

		_, err = s.Trust(&update)
		if err != nil {
			s.Error(fmt.Sprintf("failed to update OPS Cert in trust store: %s", err))
		}

		s.Info(fmt.Sprintf("updated OPS cert: %s - %s", cert.Subject, cert.SKI))
	}

	return nil
}

func (s *SecurityDefault) Resolve(ski string, publisher string) (*pki.CertEntry, error) {
	cert, err := s.pki.Certificates.Get(ski)
	if err != nil {
		return nil, err
	}

	if cert != nil {
		return cert, nil
	}

	s.Info(fmt.Sprintf("no certificate found for SKI: %s scanning DNS ...", ski))

	result := DNS.Lookup(&LookupRequest{
		SKI:       ski,
		Publisher: publisher,
	})

	if result == nil {
		s.Error("public key lookup via DNS failed: dnssec failure or missing records")
		return nil, nil
	}

	ca, err := s.Trust(&result.CA)
	if err != nil {
		return nil, err
	}
	s.caCache.AppendCertsFromPEM(result.CA)
	s.Info(fmt.Sprintf("imported: %s - %s", ca.Subject, ca.SKI))

	iss, err := s.Trust(&result.Issuer)
	if err != nil {
		return nil, err
	}
	s.intermediateCache.AppendCertsFromPEM(result.Issuer)
	s.Info(fmt.Sprintf("imported: %s - %s", iss.Subject, iss.SKI))

	user, err := s.Trust(&result.SKI)
	if err != nil {
		return nil, err
	}
	s.Info(fmt.Sprintf("imported: %s - %s", user.Subject, user.SKI))

	return s.pki.Certificates.Get(ski)
}

func (s *SecurityDefault) VerifyManifest(manifest *ops.Manifest) ([]*VerifyResult, error) {
	signatures := manifest.Signatures()
	content := manifest.ToSigningJson()

	return s.verify(content, signatures, manifest.Publisher)
}

func (s *SecurityDefault) VerifyMetadata(metadata *metadata.Metadata, publisher string) ([]*VerifyResult, error) {
	var signatures []*action.Signature
	entries, err := metadata.Signatures.Entries()
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		signatures = append(signatures, entry.Signature)
	}

	content, err := metadata.ToSigningJson()
	if err != nil {
		return nil, err
	}

	return s.verify(content, signatures, publisher)
}

func (s *SecurityDefault) validateChain(opts x509.VerifyOptions, certificate *x509.Certificate) error {
	_, err := certificate.Verify(opts)

	return err
}

func (s *SecurityDefault) verify(content []byte, signatures []*action.Signature, publisher string) ([]*VerifyResult, error) {
	var validated []*VerifyResult

	opts := x509.VerifyOptions{
		Roots:         s.caCache,
		Intermediates: s.intermediateCache,
	}

	for _, sig := range signatures {
		certEntry, err := s.Resolve(sig.SKI, publisher)

		if err != nil || certEntry == nil {
			continue
		}

		asn, _ := pem.Decode(certEntry.Cert)
		if asn == nil {
			return nil, fmt.Errorf("failed to parse pem for cert entry: %s", certEntry.Fingerprint)
		}

		cert, err := x509.ParseCertificate(asn.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse asn for cert entry: %s", certEntry.Fingerprint)
		}

		// TODO Check CRL, if found

		if s.validateChain(opts, cert) == nil && ValidateBytes(&content, cert, *sig) == nil {
			validated = append(validated, &VerifyResult{certEntry, sig})
		}
	}

	if len(validated) == 0 {
		return nil, errors.New("no trusted certificates found for signatures")
	}

	return validated, nil
}
