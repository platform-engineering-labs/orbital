package metadata

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/asdine/storm"
	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/ops"
	bolt "go.etcd.io/bbolt"
)

type Metadata struct {
	Path  string
	Prune int

	Packages   *Packages
	Channels   *Channels
	Signatures *Signatures

	db *storm.DB
}

type Packages struct {
	meta *Metadata
}

type Channels struct {
	meta *Metadata
}

type Signatures struct {
	meta *Metadata
}

func New(path string, prune int) *Metadata {
	meta := &Metadata{Path: path, Prune: prune}

	meta.Channels = &Channels{meta}
	meta.Packages = &Packages{meta}
	meta.Signatures = &Signatures{meta}

	return meta
}

func (m *Metadata) getDb() (*storm.DB, error) {
	var err error

	if m.db == nil {
		m.db, err = storm.Open(m.Path, storm.BoltOptions(0600, &bolt.Options{Timeout: 10 * time.Second}))
		if err != nil {
			return nil, err
		}
	}

	return m.db, nil
}

func (m *Metadata) Close() {
	_ = m.db.Close()
	m.db = nil
}

func (m *Metadata) Empty() error {
	return os.RemoveAll(m.Path)
}

func (m *Metadata) Exists() bool {
	if _, err := os.Stat(m.Path); os.IsNotExist(err) {
		return false
	}

	return true
}

func (m *Metadata) ToSigningJson() ([]byte, error) {
	var packages ops.Headers

	entries, err := m.Packages.Entries("")
	if err != nil {
		return nil, err
	}
	for _, pkg := range entries {
		packages = append(packages, pkg.Header)
	}

	sort.Sort(packages)

	return json.Marshal(packages)
}

func (p *Packages) Count(channel string) int {
	entries, err := p.Entries(channel)
	if err != nil {
		return 0
	}

	return len(entries)
}

func (p *Packages) Entries(channel string) ([]*Entry, error) {
	db, err := p.meta.getDb()
	if err != nil {
		return nil, err
	}

	var entries []*Entry

	err = db.All(&entries)
	if channel == "" || channel == "all" {
		return entries, nil
	}

	var filtered []*Entry

	chn, err := p.meta.Channels.Get(channel)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			return nil, err
		} else {
			return entries, nil
		}
	}

	for _, entry := range entries {
		if slices.ContainsFunc(chn.EntryIds, func(i *ops.Id) bool { return i.String() == entry.Header.Id().String() }) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, err
}

func (p *Packages) NameMap() map[string][]*Entry {
	nameMap := make(map[string][]*Entry)

	entries, err := p.Entries("")
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		nameMap[entry.Name] = append(nameMap[entry.Name], entry)
	}

	for name := range nameMap {
		sort.Slice(nameMap[name], func(i, j int) bool {
			return nameMap[name][i].Version.GT(nameMap[name][j].Version)
		})
	}

	return nameMap
}

func (p *Packages) Prune() ([]*Entry, error) {
	var candidates []*Entry
	var pruned []*Entry

	channels, err := p.meta.Channels.Entries()
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		channel.Prune(p.meta.Prune)

		err = p.meta.Channels.Update(channel)
		if err != nil {
			return nil, err
		}
	}

	nameMap := p.NameMap()

	for name := range nameMap {
		for len(nameMap[name]) > p.meta.Prune {
			var prune *Entry
			prune, nameMap[name] = nameMap[name][len(nameMap[name])-1], nameMap[name][:len(nameMap[name])-1]
			candidates = append(candidates, prune)
		}
	}

	for _, entry := range candidates {
		if !p.meta.Channels.HasChannel(entry.Id) {
			err := p.Del(entry.Id)
			if err != nil {
				return nil, err
			}

			pruned = append(pruned, entry)
		}
	}

	return pruned, nil
}

func (p *Packages) Get(name string) ([]*Entry, error) {
	db, err := p.meta.getDb()
	if err != nil {
		return nil, err
	}

	var entries []*Entry

	err = db.Prefix("Id", name+"@", &entries)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return entries, nil
}

func (p *Packages) Del(id string) error {
	db, err := p.meta.getDb()
	if err != nil {
		return err
	}

	err = db.DeleteStruct(&Entry{Id: id})

	return err
}

func (p *Packages) DelR(id string) (*Entry, error) {
	db, err := p.meta.getDb()
	if err != nil {
		return nil, err
	}

	chans, err := p.meta.Channels.Entries()
	if err != nil {
		return nil, err
	}

	for _, chn := range chans {
		err := p.meta.Channels.Remove(id, chn.Name)
		if err != nil {
			return nil, err
		}
	}

	entry := &Entry{}
	err = db.One("Id", id, entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	err = db.DeleteStruct(&Entry{Id: id})

	return entry, err
}

func (p *Packages) Put(pkg *ops.Header) (*Entry, error) {
	db, err := p.meta.getDb()
	if err != nil {
		return nil, err
	}

	lookup := &Entry{}
	entry := &Entry{Id: pkg.Id().String(), Header: pkg}

	err = db.One("Id", pkg.Id().String(), lookup)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {

			if p.Reject(entry) {
				return nil, nil
			}

			err = db.Save(entry)
			if err != nil {
				return nil, err
			}

			return entry, nil
		} else {
			return nil, err
		}
	}

	return nil, err
}

func (p *Packages) Reject(entry *Entry) bool {
	versions := p.NameMap()[entry.Name]
	versions = append(versions, entry)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version.GT(versions[j].Version)
	})

	if slices.IndexFunc(versions, func(e *Entry) bool {
		return e.Id == entry.Id
	})+1 > p.meta.Prune {
		return true
	}

	return false
}

func (p *Packages) PutAll(pkgs []*ops.Header, channels []string) (saved, pruned []*Entry, err error) {
	for _, pkg := range pkgs {
		entry, err := p.Put(pkg)
		if err != nil {
			return nil, nil, err
		}

		if entry != nil {
			saved = append(saved, entry)
			for _, channel := range channels {
				err := p.meta.Channels.Add(entry.Id, channel)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	pruned, err = p.Prune()
	if err != nil {
		return nil, nil, err
	}

	var added []*Entry
	var rejected []*Entry
	var removed []*Entry

	for _, entry := range saved {
		if !slices.ContainsFunc(pruned, func(e *Entry) bool {
			return e.Version.EXQ(entry.Version)
		}) {
			added = append(added, entry)
		} else {
			rejected = append(rejected, entry)
		}
	}

	for _, entry := range pruned {
		if !slices.ContainsFunc(rejected, func(e *Entry) bool {
			return e.Version.EXQ(entry.Version)
		}) {
			removed = append(removed, entry)
		}
	}

	return added, removed, nil
}

func (c *Channels) Get(name string) (*Channel, error) {
	db, err := c.meta.getDb()
	if err != nil {
		return nil, err
	}

	chn := &Channel{}
	err = db.One("Name", name, chn)
	if err != nil {
		return nil, err
	}

	return chn, nil
}

func (c *Channels) Add(id string, channel string) error {
	db, err := c.meta.getDb()
	if err != nil {
		return err
	}

	entry := &Entry{}

	err = db.One("Id", id, entry)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}

		return err
	}

	chn := &Channel{}

	err = db.One("Name", channel, chn)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			return err
		} else {
			chn = &Channel{Name: channel}
		}
	}

	if !slices.ContainsFunc(chn.EntryIds, func(i *ops.Id) bool { return i.String() == id }) {
		chn.EntryIds = append(chn.EntryIds, entry.Header.Id())
	}

	chn.Prune(c.meta.Prune)

	return db.Save(chn)
}

func (c *Channels) Update(channel *Channel) error {
	db, err := c.meta.getDb()
	if err != nil {
		return err
	}

	err = db.Save(channel)
	return err
}

func (c *Channels) Remove(id string, channel string) error {
	db, err := c.meta.getDb()
	if err != nil {
		return err
	}

	var chn *Channel

	err = db.One("Name", channel, &chn)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}
	}

	if chn == nil {
		return nil
	}

	slices.DeleteFunc(chn.EntryIds, func(d *ops.Id) bool {
		return d.String() == id
	})

	err = db.Save(chn)
	return err
}

func (c *Channels) Entries() ([]*Channel, error) {
	db, err := c.meta.getDb()
	if err != nil {
		return nil, err
	}

	var channels []*Channel

	err = db.All(&channels)

	return channels, err
}

func (c *Channels) HasChannel(id string) bool {
	entries, err := c.Entries()
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if slices.ContainsFunc(entry.EntryIds, func(i *ops.Id) bool { return i.String() == id }) {
			return true
		}
	}

	return false
}

func (c *Channels) List() ([]string, error) {
	db, err := c.meta.getDb()
	if err != nil {
		return nil, err
	}

	var channels []*Channel
	var channelNames []string

	err = db.All(&channels)

	for _, c := range channels {
		channelNames = append(channelNames, c.Name)
	}

	sort.Strings(channelNames)

	return channelNames, nil
}

func (s *Signatures) Get(fingerprint string) (*Signature, error) {
	db, err := s.meta.getDb()
	if err != nil {
		return nil, err
	}

	sig := &Signature{}
	err = db.One("Id", fingerprint, sig)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func (s *Signatures) Put(sig *action.Signature) error {
	db, err := s.meta.getDb()
	if err != nil {
		return err
	}

	entry := &Signature{Id: sig.FingerPrint, Signature: sig}

	return db.Save(entry)
}

func (s *Signatures) Entries() ([]*Signature, error) {
	db, err := s.meta.getDb()
	if err != nil {
		return nil, err
	}

	var signatures []*Signature

	err = db.All(&signatures)

	return signatures, err
}
