package pki

import (
	"errors"
	"os"
	"time"

	"github.com/asdine/storm"
	bolt "go.etcd.io/bbolt"
)

type Trust struct {
	Path         string
	Certificates *Certificates

	readOnly bool
	db       *storm.DB
}

type Certificates struct {
	trust *Trust
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

func NewTrust(path string, readOnly bool) *Trust {
	trust := &Trust{Path: path, readOnly: readOnly}
	trust.Certificates = &Certificates{trust}

	return trust
}

func NewCertEntry(fingerprint string, publisher string, ctype CertType, cert []byte) *CertEntry {
	return &CertEntry{Fingerprint: fingerprint, Publisher: publisher, Type: ctype, Cert: cert}
}

func (t *Trust) getDb() (*storm.DB, error) {
	var err error

	if t.db == nil {
		_ = os.Chmod(t.Path, 0644)
		t.db, err = storm.Open(t.Path, storm.BoltOptions(0644, &bolt.Options{Timeout: 10 * time.Second, ReadOnly: t.readOnly}))
		if err != nil {
			return nil, err
		}
	}

	return t.db, nil
}

func (t *Trust) Close() {
	_ = t.db.Close()
	t.db = nil
}

func (t *Trust) Touch() error {
	_, err := t.getDb()
	if err != nil {
		return err
	}
	defer t.Close()

	return nil
}

func (c *Certificates) All() ([]*CertEntry, error) {
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return nil, err
	}
	defer c.trust.Close()

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
	db, err := c.trust.getDb()
	if err != nil {
		return err
	}
	defer c.trust.Close()

	err = db.DeleteStruct(&CertEntry{SKI: ski})

	return err
}

func (c *Certificates) Put(ski string, fingerprint string, subject string, publisher string, ctype CertType, cert []byte) error {
	db, err := c.trust.getDb()
	if err != nil {
		return err
	}
	defer c.trust.Close()

	entry := &CertEntry{ski, fingerprint, subject, publisher, ctype, cert}

	err = db.Save(entry)
	return err
}
