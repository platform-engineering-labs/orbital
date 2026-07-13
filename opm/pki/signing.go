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

type Signing struct {
	Path     string
	KeyPairs *KeyPairs

	db *storm.DB
}

type KeyPairs struct {
	signing *Signing
}

func NewSigning(path string) *Signing {
	signing := &Signing{Path: path}
	signing.KeyPairs = &KeyPairs{signing}

	return signing
}

func (s *Signing) getDb() (*storm.DB, error) {
	var err error

	if s.db == nil {
		s.db, err = storm.Open(s.Path, storm.BoltOptions(0600, &bolt.Options{Timeout: 10 * time.Second}))
		if err != nil {
			return nil, err
		}
	}

	return s.db, nil
}

func (s *Signing) Close() {
	_ = s.db.Close()
	s.db = nil
}

func (k *KeyPairs) All() ([]*KeyPairEntry, error) {
	db, err := k.signing.getDb()
	if err != nil {
		return nil, err
	}
	defer k.signing.Close()

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
	db, err := k.signing.getDb()
	if err != nil {
		return nil, err
	}
	defer k.signing.Close()

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
	db, err := k.signing.getDb()
	if err != nil {
		return nil, err
	}
	defer k.signing.Close()

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

// TODO: get rid of this
func (k *KeyPairs) FirstByPublisher(publisher string) (*KeyPairEntry, error) {
	pairs, err := k.GetByPublisher(publisher)
	if err != nil {
		return nil, err
	}

	if len(pairs) > 0 {
		return pairs[0], nil
	}

	return nil, nil
}

func (k *KeyPairs) GetByPublisher(publisher string) ([]*KeyPairEntry, error) {
	db, err := k.signing.getDb()
	if err != nil {
		return nil, err
	}
	defer k.signing.Close()

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
	db, err := k.signing.getDb()
	if err != nil {
		return nil, err
	}
	defer k.signing.Close()

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
	db, err := k.signing.getDb()
	if err != nil {
		return err
	}
	defer k.signing.Close()

	err = db.DeleteStruct(&KeyPairEntry{SKI: ski})

	return err
}

func (k *KeyPairs) Put(ski string, fingerprint string, subject string, publisher string, cert []byte, key []byte) error {
	db, err := k.signing.getDb()
	if err != nil {
		return err
	}
	defer k.signing.Close()

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
