package security

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/opm/pki"
)

const (
	DefaultDigestMethod = "sha256"
)

type CertMetadata struct {
	Subject     string
	Publisher   string
	Type        pki.CertType
	Fingerprint string
	SKI         string
}

type VerifyResult struct {
	Cert      *pki.CertEntry
	Signature *action.Signature
}

func CertMetadataFromBytes(certPem *[]byte) (*CertMetadata, error) {
	block, _ := pem.Decode(*certPem)
	if block == nil {
		return nil, errors.New("failed to parse certificate pem")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate: " + err.Error())
	}

	fingerprint := SpkiFingerprint(cert).String()
	ski := SpkiSKI(cert)

	if len(cert.Subject.Organization) == 0 {
		return nil, errors.New("invalid certificate organization")
	}

	if len(cert.Subject.OrganizationalUnit) == 0 {
		return nil, errors.New("invalid certificate organizational unit")
	}

	return &CertMetadata{
		SKI:         ski,
		Subject:     cert.Subject.CommonName,
		Publisher:   cert.Subject.Organization[0],
		Type:        pki.CertType(cert.Subject.OrganizationalUnit[0]),
		Fingerprint: fingerprint,
	}, nil
}

func SignBytes(content *[]byte, ski string, certFingerprint string, key *rsa.PrivateKey, algo string) (*action.Signature, error) {
	switch algo {
	case "sha256":
		digest := sha256.Sum256(*content)

		rng := rand.Reader

		signature, err := rsa.SignPKCS1v15(rng, key, crypto.SHA256, digest[:])
		if err != nil {
			return nil, err
		}

		return &action.Signature{
			SKI:         ski,
			FingerPrint: certFingerprint,
			Algo:        "sha256",
			Value:       hex.EncodeToString(signature),
		}, nil

	default:
		return nil, errors.New("unsupported signature algorithm")
	}
}

func SecuritySignFile(filePath string, sigPath string, ski string, fingerprint string, key *rsa.PrivateKey, algo string) error {
	cfgBytes, err := ioutil.ReadFile(filePath)

	sig, err := SignBytes(&cfgBytes, ski, fingerprint, key, algo)
	if err != nil {
		return err
	}

	sigBytes, err := json.Marshal(sig)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(sigPath, sigBytes, 0640)
}

func ValidateBytes(content *[]byte, cert *x509.Certificate, signature action.Signature) error {
	switch signature.Algo {
	case "sha256":
		sig, _ := hex.DecodeString(signature.Value)
		hash := sha256.Sum256(*content)

		err := rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, hash[:], sig)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported signature algorithm")
	}

	return nil
}

func ValidateKeyPair(certPath string, keyPath string) error {
	_, err := tls.LoadX509KeyPair(certPath, keyPath)

	return err
}

type Fingerprint []byte

func (f Fingerprint) String() string {
	var buf bytes.Buffer
	for i, b := range f {
		if i > 0 {
			fmt.Fprintf(&buf, ":")
		}
		fmt.Fprintf(&buf, "%02x", b)
	}
	return buf.String()
}

func ParseFingerprint(fp string) (Fingerprint, error) {
	s := strings.Join(strings.Split(fp, ":"), "")
	buf, err := hex.DecodeString(s)
	return Fingerprint(buf), err
}

func SpkiFingerprint(cert *x509.Certificate) Fingerprint {
	h := sha256.New()
	h.Write(cert.RawSubjectPublicKeyInfo)
	return Fingerprint(h.Sum(nil))
}

func SpkiSKI(cert *x509.Certificate) string {
	h := sha256.New()
	h.Write(cert.SubjectKeyId)
	return hex.EncodeToString(h.Sum(nil)[:16])
}
