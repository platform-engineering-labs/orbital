package state

import (
	"errors"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/platform-engineering-labs/orbital/action/actions"
	"github.com/platform-engineering-labs/orbital/ops"
	bolt "go.etcd.io/bbolt"
)

type State struct {
	Path         string
	Frozen       *Frozen
	Packages     *Packages
	Objects      *Objects
	Transactions *Transactions

	db *storm.DB
}

type Frozen struct {
	state *State
}

type Packages struct {
	state *State
}

type Objects struct {
	state *State
}

type Transactions struct {
	state *State
}

type PkgEntry struct {
	Name     string `storm:"id"`
	Manifest []byte
}

type FrozenEntry struct {
	PkgId string `storm:"id"`
}

type FsEntry struct {
	Key  string       `storm:"id"`
	Path string       `storm:"index"`
	Pkg  string       `storm:"index"`
	Type actions.Type `storm:"index"`
}

type TransactionEntry struct {
	Key       string `storm:"id"`
	Id        string `storm:"index"`
	PkgId     string
	Operation string
	Date      *time.Time `storm:"index"`
}

func New(path string) *State {
	state := &State{Path: path}
	state.Frozen = &Frozen{state}

	state.Packages = &Packages{state}

	state.Objects = &Objects{state}

	state.Transactions = &Transactions{state}

	return state
}

func NewFsEntry(path string, pkg string, typ actions.Type) *FsEntry {
	fs := &FsEntry{Path: path, Pkg: pkg, Type: typ}
	fs.Key = strings.Join([]string{path, pkg}, "\x00")

	return fs
}

func NewTransactionEntry(id string, pkgId string, operation string, date *time.Time) *TransactionEntry {
	ts := &TransactionEntry{Id: id, PkgId: pkgId, Operation: operation, Date: date}
	ts.Key = strings.Join([]string{id, pkgId}, "\x00")

	return ts
}

func (s *State) getDb() (*storm.DB, error) {
	var err error

	if s.db == nil {
		s.db, err = storm.Open(s.Path, storm.BoltOptions(0600, &bolt.Options{Timeout: 10 * time.Second}))
		if err != nil {
			return nil, err
		}
	}

	return s.db, nil
}

func (s *State) Close() {
	_ = s.db.Close()
	s.db = nil
}

func (p *Packages) All() ([]*ops.Manifest, error) {
	db, err := p.state.getDb()
	if err != nil {
		return nil, err
	}
	defer p.state.Close()

	var entries []*PkgEntry
	var packages []*ops.Manifest

	err = db.All(&entries)

	for _, pkg := range entries {
		manifest := &ops.Manifest{}
		err := manifest.Load(pkg.Manifest)
		if err != nil {
			return nil, err
		}

		packages = append(packages, manifest)
	}

	return packages, nil
}

func (p *Packages) Get(name string) (*ops.Manifest, error) {
	db, err := p.state.getDb()
	if err != nil {
		return nil, err
	}
	defer p.state.Close()

	var entry PkgEntry
	pkg := &ops.Manifest{}

	err = db.One("Name", name, &entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	err = pkg.Load(entry.Manifest)
	if err != nil {
		return nil, err
	}

	return pkg, err
}

func (p *Packages) Del(name string) error {
	db, err := p.state.getDb()
	if err != nil {
		return err
	}
	defer p.state.Close()

	err = db.DeleteStruct(&PkgEntry{Name: name})

	return err
}

func (p *Packages) Put(name string, pkg *ops.Manifest) error {
	db, err := p.state.getDb()
	if err != nil {
		return err
	}
	defer p.state.Close()

	entry := &PkgEntry{name, []byte(pkg.ToJson())}

	err = db.Save(entry)
	return err
}

func (o *Objects) All() ([]*FsEntry, error) {
	db, err := o.state.getDb()
	if err != nil {
		return nil, err
	}
	defer o.state.Close()

	var entries []*FsEntry

	err = db.All(&entries)

	return entries, nil
}

func (o *Objects) Get(path string) ([]*FsEntry, error) {
	db, err := o.state.getDb()
	if err != nil {
		return nil, err
	}
	defer o.state.Close()

	var entries []*FsEntry

	err = db.Find("Path", path, &entries)

	return entries, nil
}

func (o *Objects) Del(pkg string) error {
	db, err := o.state.getDb()
	if err != nil {
		return err
	}
	defer o.state.Close()

	query := db.Select(q.Eq("Pkg", pkg))
	err = query.Delete(&FsEntry{})

	return err
}

func (o *Objects) Put(path string, pkg string, typ actions.Type) error {
	db, err := o.state.getDb()
	if err != nil {
		return err
	}
	defer o.state.Close()

	err = db.Save(NewFsEntry(path, pkg, typ))

	return err
}

func (f *Frozen) All() ([]*FrozenEntry, error) {
	db, err := f.state.getDb()
	if err != nil {
		return nil, err
	}
	defer f.state.Close()

	var entries []*FrozenEntry

	err = db.All(&entries)

	return entries, nil
}

func (f *Frozen) Del(pkgId string) error {
	db, err := f.state.getDb()
	if err != nil {
		return err
	}
	defer f.state.Close()

	err = db.DeleteStruct(&FrozenEntry{pkgId})

	return err
}

func (f *Frozen) Put(pkgId string) error {
	db, err := f.state.getDb()
	if err != nil {
		return err
	}
	defer f.state.Close()

	err = db.Save(&FrozenEntry{pkgId})

	return err
}

func (t *Transactions) All() ([]*TransactionEntry, error) {
	db, err := t.state.getDb()
	if err != nil {
		return nil, err
	}
	defer t.state.Close()

	var entries []*TransactionEntry

	err = db.AllByIndex("Date", &entries)

	return entries, nil
}

func (t *Transactions) Get(id string) ([]*TransactionEntry, error) {
	db, err := t.state.getDb()
	if err != nil {
		return nil, err
	}
	defer t.state.Close()

	var entries []*TransactionEntry

	err = db.Find("Id", id, &entries)

	return entries, nil
}

func (t *Transactions) Put(id string, pkgId string, operation string, date *time.Time) error {
	db, err := t.state.getDb()
	if err != nil {
		return err
	}
	defer t.state.Close()

	err = db.Save(NewTransactionEntry(id, pkgId, operation, date))

	return err
}
