package pki

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	"github.com/asdine/storm"
	bolt "go.etcd.io/bbolt"
)

type CertType string

const (
	CertCA     CertType = "CA"
	CertIssuer CertType = "Issuer"
	CertUser   CertType = "User"
	CertCRL    CertType = "CRL"
)

type Pki struct {
	Path         string
	Certificates *Certificates
	KeyPairs     *KeyPairs

	db *storm.DB
}

type Certificates struct {
	pki *Pki
}

type KeyPairs struct {
	pki *Pki
}

type CertEntry struct {
	SKI         string   `storm:"id"`
	Fingerprint string   `storm:"index"`
	Subject     string   `storm:"index"`
	Publisher   string   `storm:"index"`
	Type        CertType `storm:"index"`
	Cert        []byte
}

type KeyPairEntry struct {
	SKI         string `storm:"id"`
	Fingerprint string `storm:"index"`
	Subject     string `storm:"index"`
	Publisher   string `storm:"index"`
	Cert        []byte
	Key         []byte
}

func New(path string) *Pki {
	pki := &Pki{Path: path}
	pki.Certificates = &Certificates{pki}

	pki.KeyPairs = &KeyPairs{pki}

	return pki
}

func NewCertEntry(fingerprint string, publisher string, ctype CertType, cert []byte) *CertEntry {
	return &CertEntry{Fingerprint: fingerprint, Publisher: publisher, Type: ctype, Cert: cert}
}

func (p *Pki) getDb() (*storm.DB, error) {
	var err error

	if p.db == nil {
		p.db, err = storm.Open(p.Path, storm.BoltOptions(0600, &bolt.Options{Timeout: 10 * time.Second}))
		if err != nil {
			return nil, err
		}
	}

	return p.db, nil
}

func (p *Pki) Close() {
	_ = p.db.Close()
	p.db = nil
}

func (c *Certificates) All() ([]*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entries []*CertEntry

	err = db.All(&entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (c *Certificates) Get(ski string) (*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entry CertEntry

	err = db.One("SKI", ski, &entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &entry, err
}

func (c *Certificates) GetByFingerPrint(fingerprint string) (*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entry CertEntry

	err = db.One("Fingerprint", fingerprint, &entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &entry, err
}

func (c *Certificates) GetByPublisher(publisher string) ([]*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entries []*CertEntry

	err = db.Find("Publisher", publisher, &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (c *Certificates) GetBySubject(subject string) ([]*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entries []*CertEntry

	err = db.Find("Subject", subject, &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (c *Certificates) GetByType(ctype CertType) ([]*CertEntry, error) {
	db, err := c.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer c.pki.Close()

	var entries []*CertEntry

	err = db.Find("Type", ctype, &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (c *Certificates) Del(ski string) error {
	db, err := c.pki.getDb()
	if err != nil {
		return err
	}
	defer c.pki.Close()

	err = db.DeleteStruct(&CertEntry{SKI: ski})

	return err
}

func (c *Certificates) Put(ski string, fingerprint string, subject string, publisher string, ctype CertType, cert []byte) error {
	db, err := c.pki.getDb()
	if err != nil {
		return err
	}
	defer c.pki.Close()

	entry := &CertEntry{ski, fingerprint, subject, publisher, ctype, cert}

	err = db.Save(entry)
	return err
}

func (k *KeyPairs) All() ([]*KeyPairEntry, error) {
	db, err := k.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer k.pki.Close()

	var entries []*KeyPairEntry

	err = db.All(&entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (k *KeyPairs) Get(ski string) (*KeyPairEntry, error) {
	db, err := k.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer k.pki.Close()

	var entry KeyPairEntry

	err = db.One("SKI", ski, &entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &entry, err
}

func (k *KeyPairs) GetByFingerPrint(fingerprint string) (*KeyPairEntry, error) {
	db, err := k.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer k.pki.Close()

	var entry KeyPairEntry

	err = db.One("Fingerprint", fingerprint, &entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &entry, err
}

func (k *KeyPairs) GetByPublisher(publisher string) ([]*KeyPairEntry, error) {
	db, err := k.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer k.pki.Close()

	var entries []*KeyPairEntry

	err = db.Find("Publisher", publisher, &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (k *KeyPairs) GetBySubject(subject string) ([]*KeyPairEntry, error) {
	db, err := k.pki.getDb()
	if err != nil {
		return nil, err
	}
	defer k.pki.Close()

	var entries []*KeyPairEntry

	err = db.Find("Subject", subject, &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, err
}

func (k *KeyPairs) Del(ski string) error {
	db, err := k.pki.getDb()
	if err != nil {
		return err
	}
	defer k.pki.Close()

	err = db.DeleteStruct(&KeyPairEntry{SKI: ski})

	return err
}

func (k *KeyPairs) Put(ski string, fingerprint string, subject string, publisher string, cert []byte, key []byte) error {
	db, err := k.pki.getDb()
	if err != nil {
		return err
	}
	defer k.pki.Close()

	entry := &KeyPairEntry{ski, fingerprint, subject, publisher, cert, key}

	err = db.Save(entry)
	return err
}

func (k *KeyPairEntry) RSAKey() (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(k.Key)
	if block == nil {
		return nil, errors.New("failed to decode key")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
