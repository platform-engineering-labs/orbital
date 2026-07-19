package tree

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/asdine/storm"
	"github.com/gofrs/flock"
	"github.com/platform-engineering-labs/orbital/opm/cache"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/opm/state"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/schema/names"
	"github.com/platform-engineering-labs/orbital/schema/paths"
	"github.com/platform-engineering-labs/orbital/sys"
	"github.com/platform-engineering-labs/orbital/x/collections"
	filepathx "github.com/platform-engineering-labs/orbital/x/filepath"
	bolt "go.etcd.io/bbolt"
)

type Type string

const (
	Dynamic  Type = "dynamic"
	Embedded Type = "embedded"
	Root     Type = "root"
)

var tpl = `amends "orbital:/tree.pkl"

os = "%s"
arch = "%s"
`

type Entry struct {
	Path string `storm:"id"`
	Name string `storm:"unique"`
}

type Tree struct {
	*slog.Logger

	Name string
	Path string

	Config   *Config
	Platform *platform.Platform

	Cache    *cache.Cache
	Signing  *pki.Signing
	Trust    *pki.Trust
	Security security.Security
	State    *state.State

	privileged bool
	writable   bool
	lock       *flock.Flock
}

func Add(entry *Entry) error {
	db, err := Store()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Save(entry)
}

func CreateDefault() error {
	path := filepath.Join(paths.TreeRootDefault(), names.TreeDefault)

	if _, err := Get(names.TreeDefault); err == nil {
		return nil
	}

	entry, err := Init(names.TreeDefault, path, platform.Current(), true, false)
	if err != nil {
		return err
	}

	err = Add(entry)
	if err != nil {
		return err
	}

	return Switch(names.TreeDefault)
}

func Current() (*Entry, error) {
	current, _ := os.Readlink(filepath.Join(paths.DataDefault(), names.TreeCurrent))

	if current == "" {
		current = filepath.Join(paths.DataDefault(), names.TreeDefault)
		_ = Switch(names.TreeDefault)
	}

	if tree := Virtual(); tree != nil {
		if current == tree.Path {
			return tree, nil
		}
	}

	tree, err := Lookup(current)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) && filepathx.FileExists(filepath.Join(current, names.TreeDataDir)) {
			return &Entry{
				Name: "$previous",
				Path: current,
			}, nil
		}

		return nil, err
	}

	return tree, nil
}

func Destroy(name string) (*Entry, error) {
	if name == "" {
		return nil, fmt.Errorf("error: tree name should not be empty")
	}

	tree, err := Get(name)
	if err != nil {
		return nil, err
	}

	db, err := Store()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = db.DeleteStruct(tree)
	if err != nil {
		return nil, err
	}

	return tree, os.RemoveAll(tree.Path)
}

func Get(name string) (*Entry, error) {
	tree := Virtual()

	if tree := Virtual(); tree != nil {
		if name == "virtual" {
			return tree, nil
		}
	}

	db, err := Store()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tree = &Entry{}
	err = db.One("Name", name, tree)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func Init(name string, path string, pltfrm *platform.Platform, createConfig bool, force bool) (*Entry, error) {
	if path == "" || name == "" {
		return nil, fmt.Errorf("error: tree root or name should not be empty")
	}

	if filepathx.FileExists(filepath.Join(path, names.TreeDataDir)) && !force {
		return nil, fmt.Errorf("%s already exists: %s", name, path)
	}

	if filepathx.FileExists(path) && force {
		err := os.RemoveAll(path)
		if err != nil {
			return nil, err
		}
	}

	err := os.MkdirAll(filepath.Join(path, names.TreeDataDir), 0755)
	if err != nil {
		return nil, err
	}

	if createConfig {
		err = os.WriteFile(
			filepath.Join(path, names.TreeDataDir, names.TreeConfigFile),
			[]byte(fmt.Sprintf(tpl, pltfrm.OS, pltfrm.Arch)),
			0644)
		if err != nil {
			return nil, err
		}
	}

	st := state.New(paths.TreeState(path), false)
	err = st.Touch()
	if err != nil {
		return nil, err
	}

	si := pki.NewSigning(paths.TreeSigning(path))
	si.Touch()

	tru := pki.NewTrust(paths.TreeTrust(path), false)
	tru.Touch()

	return &Entry{
		Name: name,
		Path: path,
	}, nil
}

func List() ([]*Entry, error) {
	var trees []*Entry

	db, err := Store()
	if err != nil {
		return nil, err
	}

	err = db.All(&trees)
	if err != nil {
		return nil, err
	}
	db.Close()

	if virt := Virtual(); virt != nil {
		trees = append(trees, virt)
	}

	if current, err := Current(); err == nil {
		if !slices.ContainsFunc(
			trees, func(tree *Entry) bool {
				return tree.Path == current.Path
			}) {
			trees = append(trees, current)
		}
	}

	return trees, nil
}

func Lookup(path string) (*Entry, error) {
	db, err := Store()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tree := &Entry{}
	err = db.One("Path", path, tree)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func Privileged(path string) bool {
	if !filepathx.FileExists(path) {
		path = filepath.Dir(path)
	}

	stat, _ := os.Stat(path)
	if stat != nil {
		if stat.Sys().(*syscall.Stat_t).Uid != 0 {
			return false
		}
	}

	return true
}

func Store() (*storm.DB, error) {
	return storm.Open(paths.TreeStore(), storm.BoltOptions(0644, &bolt.Options{Timeout: 10 * time.Second}))
}

func Switch(name string) error {
	var err error

	tree, err := Get(name)
	if err != nil {
		return err
	}

	_ = os.Remove(filepath.Join(paths.DataDefault(), names.TreeCurrent))

	err = os.Symlink(tree.Path, filepath.Join(paths.DataDefault(), names.TreeCurrent))
	if err != nil {
		return err
	}

	return nil
}

func Virtual() *Entry {
	binPath, _ := os.Executable()
	binPathFinal, _ := filepath.EvalSymlinks(binPath)

	if binPathFinal != "" {
		extRoot := filepath.Dir(filepath.Dir(binPathFinal))
		_, err := Lookup(extRoot)

		if err != nil {
			if filepathx.FileExists(filepath.Join(extRoot, names.TreeDataDir)) {
				_, err := Load(filepath.Join(extRoot, names.TreeDataDir, names.TreeConfigFile))
				if err != nil {
					return nil
				}

				return &Entry{
					Name: "$virtual",
					Path: extRoot,
				}
			}
		}
	}

	return nil
}

func New(log *slog.Logger, name string, path string, writeable bool, cfg *Config) (*Tree, error) {
	var err error

	tr := &Tree{Logger: log, Name: name, Path: path, Config: cfg, Platform: platform.Current(), writable: writeable}

	if cfg == nil {
		tr.Config, err = Load(filepath.Join(tr.Path, names.TreeDataDir, names.TreeConfigFile))
		if err != nil {
			return nil, err
		}

		tr.Platform = &platform.Platform{
			OS:   tr.Config.OS,
			Arch: tr.Config.Arch,
		}
	}

	tr.privileged = Privileged(tr.Path)

	if tr.Ready() {
		tr.Cache = cache.New(paths.TreeCache(tr.Path))
		tr.Trust = pki.NewTrust(paths.TreeTrust(tr.Path), !writeable)

		if writeable {
			tr.Signing = pki.NewSigning(paths.TreeSigning(tr.Path))
			tr.lock = flock.New(paths.TreeLock(tr.Path))
		}

		tr.Security, err = security.New(tr.Logger, tr.Config.Security, tr.Trust)
		if err != nil {
			return nil, err
		}

		tr.State = state.New(paths.TreeState(tr.Path), !writeable)
	}
	return tr, nil
}

func (t *Tree) Lock() error {
	return t.lock.Lock()
}

func (t *Tree) Pool(platforms []*platform.Platform, empty bool, repos ...*ops.Repository) (*ops.Pool, error) {
	for _, repo := range t.Config.Repositories {
		if repo.Enabled {
			err := t.RepoLoad(platforms, &repo, false)
			if err != nil {
				return nil, err
			}

			repos = append(repos, &repo)
		}
	}

	frozenEntries, err := t.State.Frozen.All()
	if err != nil {
		return nil, err
	}

	frozen := make(map[string]bool)
	for _, entry := range frozenEntries {
		frozen[entry.PkgId] = true
	}

	uri, _ := url.Parse("tree://none")
	rpState := ops.NewRepo(*uri, true, -1)
	if !empty {
		rpState, err = t.StateToRepo()
		if err != nil {
			return nil, err
		}
	}

	pool, err := ops.NewPool(rpState, frozen, repos...)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (t *Tree) Privileged() bool {
	return t.privileged
}

func (t *Tree) Ready() bool {
	if t.writable {
		if t.privileged && !sys.IsPrivilegedUser() {
			return false
		}
	}
	return filepathx.FileExists(filepath.Join(t.Path, names.TreeDataDir))
}

func (t *Tree) RepoLoad(platforms []*platform.Platform, repo *ops.Repository, all bool) error {
	for _, pltfrm := range platforms {
		metadataPath := t.Cache.GetMeta(pltfrm.String(), repo.SafeUri())

		md := metadata.New(metadataPath, true, repo.Prune)
		if !md.Exists() {
			continue
		}
		defer md.Close()

		// TODO Validate signature

		channel := repo.Uri.Fragment
		if all {
			channel = ""
		}

		entries, err := md.Packages.Entries(channel)
		if err != nil {
			return err
		}
		headers := collections.Map(entries, func(entry *metadata.Entry) *ops.Header {
			return entry.Header
		})

		var channels []*metadata.Channel
		if all {
			channels, err = md.Channels.Entries()
			if err != nil {
				return err
			}
		} else {
			chn, err := md.Channels.Get(channel)
			if err != nil {
				if !errors.Is(storm.ErrNotFound, err) {
					return err
				}
			}

			if chn != nil {
				channels = append(channels, chn)
			}
		}

		repoChans := collections.Map(channels, func(entry *metadata.Channel) *ops.Channel {
			return &ops.Channel{
				Name:     entry.Name,
				EntryIds: entry.EntryIds,
			}
		})

		err = repo.Load(pltfrm, repoChans, headers)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tree) StateToRepo() (*ops.Repository, error) {
	packages, err := t.State.Packages.All()
	if err != nil {
		return nil, err
	}

	var headers ops.Headers
	for _, manifest := range packages {
		headers = append(headers, manifest.Header)
	}

	uri, _ := url.Parse("tree://" + t.Name)
	repo := ops.NewRepo(*uri, true, -1)

	err = repo.Load(t.Config.Platform(), nil, headers)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (t *Tree) Unlock() error {
	defer os.Remove(t.lock.Path())
	return t.lock.Unlock()
}
