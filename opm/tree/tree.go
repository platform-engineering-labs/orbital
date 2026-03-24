package tree

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
)

var tpl = `amends "orbital:/tree.pkl"

os = "%s"
arch = "%s"
`

type Tree interface {
	Cache() *cache.Cache
	Config() *Config
	Current() *Entry
	Destroy(name string) (*Entry, error)
	Get(name string) (*Entry, error)
	Init(name string, pltfrm *platform.Platform, force bool) (*Entry, error)
	Ready() bool
	List() ([]*Entry, error)
	Lock() error
	Unlock() error
	Pki() *pki.Pki
	Pool(platforms []*platform.Platform, empty bool) (*ops.Pool, error)
	RepoLoad(platforms []*platform.Platform, repo *ops.Repository, all bool) error
	Security() security.Security
	State() *state.State
	StateToRepo() (*ops.Repository, error)
	Switch(string) error
}
type Entry struct {
	Name     string
	Path     string
	Platform *platform.Platform

	Current    bool
	Privileged bool
}

type Type string

const (
	Dynamic  Type = "dynamic"
	Embedded Type = "embedded"
	Root     Type = "root"
)

func CreateDefault(root string) error {
	tree := &TreeDynamic{root: root, platform: platform.Current()}

	if _, err := tree.Get(names.TreeDefault); err == nil {
		return nil
	}

	_, err := tree.Init(names.TreeDefault, platform.Current(), false)
	if err != nil {
		return err
	}

	return tree.Switch(names.TreeDefault)
}

func New(log *slog.Logger, root string, t Type, cfg *Config) (Tree, error) {
	var err error

	switch t {
	case Dynamic:
		tr := &TreeDynamic{Logger: log, root: root}

		tr.cfg, err = Load(filepath.Join(tr.Current().Path, names.TreeDataDir, names.TreeConfigFile))
		if err != nil {
			return nil, err
		}

		tr.platform = &platform.Platform{
			OS:   tr.cfg.OS,
			Arch: tr.cfg.Arch,
		}

		if tr.Current().Privileged && !sys.IsPrivilegedUser() {
			if !sys.SudoSessionActive() {
				log.Warn(fmt.Sprintf("privileged user required for path: %s", tr.Current().Path))
			}

			err := sys.InvokeSelfWithSudo()
			if err != nil {
				return nil, err
			}
		}

		if tr.Ready() {
			tr.cache = cache.New(paths.TreeCache(tr.Current().Path))
			tr.pki = pki.New(paths.TreePki(tr.Current().Path))
			tr.lock = flock.New(paths.TreeLock(tr.Current().Path))

			tr.sec, err = security.New(tr.Logger, tr.cfg.Security, tr.pki)
			if err != nil {
				return nil, err
			}

			tr.state = state.New(paths.TreeState(tr.Current().Path))
		}

		return tr, nil
	case Embedded:
		tr := &TreeEmbedded{Logger: log, root: root, platform: platform.Current()}

		tr.cfg = cfg

		if tr.Current().Privileged && !sys.IsPrivilegedUser() {
			if !sys.SudoSessionActive() {
				log.Warn(fmt.Sprintf("privileged user required for path: %s", tr.Current().Path))
			}

			err := sys.InvokeSelfWithSudo()
			if err != nil {
				return nil, err
			}
		}

		if tr.Ready() {
			tr.cache = cache.New(paths.TreeCache(tr.Current().Path))
			tr.pki = pki.New(paths.TreePki(tr.Current().Path))
			tr.lock = flock.New(paths.TreeLock(tr.Current().Path))

			tr.sec, err = security.New(tr.Logger, tr.cfg.Security, tr.pki)
			if err != nil {
				return nil, err
			}

			tr.state = state.New(paths.TreeState(tr.Current().Path))
		}

		return tr, nil
	default:
		panic("tree: unknown type")
	}
}

type TreeDynamic struct {
	*slog.Logger

	root     string
	platform *platform.Platform

	cfg *Config

	cache *cache.Cache
	pki   *pki.Pki
	sec   security.Security
	state *state.State

	lock *flock.Flock
}

func (t *TreeDynamic) Cache() *cache.Cache {
	return t.cache
}

func (t *TreeDynamic) Config() *Config {
	return t.cfg
}

func (t *TreeDynamic) Current() *Entry {
	current, _ := os.Readlink(filepath.Join(t.root, names.TreeCurrent))

	if current == "" {
		current = filepath.Join(t.root, names.TreeDefault)
		_ = t.Switch(names.TreeDefault)
	}

	return &Entry{
		Name:       filepath.Base(current),
		Path:       current,
		Platform:   t.platform,
		Current:    true,
		Privileged: privileged(current),
	}
}

func (t *TreeDynamic) Get(name string) (*Entry, error) {
	path := filepath.Join(t.root, name)

	if !filepathx.FileExists(path) {
		binPath, _ := os.Executable()
		if binPath != "" && !strings.HasPrefix(binPath, t.root) && strings.HasSuffix(filepath.Dir(filepath.Dir(binPath)), name) {
			if filepathx.FileExists(filepath.Join(filepath.Dir(filepath.Dir(binPath)), names.TreeDataDir)) {
				path = filepath.Dir(filepath.Dir(binPath))
			} else {
				return nil, fmt.Errorf("%s does not exist: at %s", name, path)
			}
		} else {
			return nil, fmt.Errorf("%s does not exist: at %s", name, path)
		}
	}

	cfg, err := Load(filepath.Join(path, names.TreeDataDir, names.TreeConfigFile))
	if err != nil {
		return nil, err
	}

	return &Entry{
		Name:       name,
		Path:       path,
		Platform:   cfg.Platform(),
		Privileged: privileged(path),
	}, nil
}

func (t *TreeDynamic) Destroy(name string) (*Entry, error) {
	if t.root == "" || name == "" {
		return nil, fmt.Errorf("error: tree root or name should not be empty")
	}

	if name == t.Current().Name {
		return nil, fmt.Errorf("error: cannot delete current tree (in use)")
	}

	path := filepath.Join(t.root, name)
	cfg, err := Load(filepath.Join(path, names.TreeDataDir, names.TreeConfigFile))
	if err != nil {
		return nil, err
	}

	return &Entry{
		Name:     name,
		Path:     path,
		Platform: cfg.Platform(),
	}, os.RemoveAll(path)
}

func (t *TreeDynamic) Init(name string, pltfrm *platform.Platform, force bool) (*Entry, error) {
	if t.root == "" || name == "" {
		return nil, fmt.Errorf("error: tree root or name should not be empty")
	}

	path := filepath.Join(t.root, name)

	if filepathx.FileExists(path) && !force {
		return nil, fmt.Errorf("%s already exists: %s", name, path)
	}

	if filepathx.FileExists(path) && force {
		err := os.RemoveAll(path)
		if err != nil {
			return nil, err
		}
	}

	err := os.MkdirAll(filepath.Join(t.root, name, names.TreeDataDir), 0755)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(
		filepath.Join(t.root, name, names.TreeDataDir, names.TreeConfigFile),
		[]byte(fmt.Sprintf(tpl, pltfrm.OS, pltfrm.Arch)),
		0644)
	if err != nil {
		return nil, err
	}

	return &Entry{
		Name:     name,
		Path:     filepath.Join(t.root, name),
		Platform: pltfrm,
	}, nil
}

func (t *TreeDynamic) List() ([]*Entry, error) {
	var trees []*Entry

	dirs, err := os.ReadDir(t.root)
	if err != nil {
		return nil, err
	}

	current := t.Current()

	for _, dir := range dirs {
		if dir.IsDir() {
			cfg, _ := Load(filepath.Join(dir.Name(), names.TreeDataDir, names.TreeConfigFile))

			trees = append(trees, &Entry{
				Name:     filepath.Base(dir.Name()),
				Path:     filepath.Join(t.root, dir.Name()),
				Platform: cfg.Platform(),
				Current:  current.Name == filepath.Base(dir.Name()),
			})
		}
	}

	// Allow management of tree outside of root for current binary
	binPath, _ := os.Executable()
	if binPath != "" && !strings.HasPrefix(binPath, t.root) {
		extRoot := filepath.Dir(filepath.Dir(binPath))

		if filepathx.FileExists(filepath.Join(extRoot, names.TreeDataDir)) {
			cfg, _ := Load(filepath.Join(extRoot, names.TreeDataDir, names.TreeConfigFile))

			trees = append(trees, &Entry{
				Name:     filepath.Base(extRoot),
				Path:     extRoot,
				Platform: cfg.Platform(),
				Current:  current.Name == filepath.Base(extRoot),
			})
		}
	}

	return trees, nil
}

func (t *TreeDynamic) Lock() error {
	return t.lock.Lock()
}

func (t *TreeDynamic) Pki() *pki.Pki {
	return t.pki
}

func (t *TreeDynamic) Pool(platforms []*platform.Platform, empty bool) (*ops.Pool, error) {
	return pool(t, platforms, empty)
}

func (t *TreeDynamic) Ready() bool {
	if privileged(t.root) && !sys.IsPrivilegedUser() {
		return false
	}
	return filepathx.FileExists(filepath.Join(t.Current().Path, names.TreeDataDir))
}

func (t *TreeDynamic) RepoLoad(platforms []*platform.Platform, repo *ops.Repository, all bool) error {
	return load(t, platforms, repo, all)
}

func (t *TreeDynamic) Security() security.Security {
	return t.sec
}

func (t *TreeDynamic) State() *state.State {
	return t.state
}

func (t *TreeDynamic) StateToRepo() (*ops.Repository, error) {
	return stateToRepo(t)
}

func (t *TreeDynamic) Switch(name string) error {
	_ = os.Remove(filepath.Join(t.root, names.TreeCurrent))

	path := filepath.Join(t.root, name)
	binPath, _ := os.Executable()
	if binPath != "" && !strings.HasPrefix(binPath, t.root) && strings.HasSuffix(filepath.Dir(filepath.Dir(binPath)), name) {
		path = filepath.Dir(filepath.Dir(binPath))
	}

	err := os.Symlink(path, filepath.Join(t.root, names.TreeCurrent))
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeDynamic) Unlock() error {
	defer os.Remove(t.lock.Path())
	return t.lock.Unlock()
}

type TreeEmbedded struct {
	*slog.Logger

	root     string
	platform *platform.Platform

	cfg *Config

	cache *cache.Cache
	pki   *pki.Pki
	sec   security.Security
	state *state.State

	lock *flock.Flock
}

func (t *TreeEmbedded) Cache() *cache.Cache {
	return t.cache
}

func (t *TreeEmbedded) Config() *Config {
	return t.cfg
}

func (t *TreeEmbedded) Current() *Entry {
	return &Entry{Name: filepath.Base(t.root), Path: t.root, Current: true, Privileged: privileged(t.root)}
}

func (t *TreeEmbedded) Destroy(name string) (*Entry, error) {
	if t.root == "" || name == "" {
		return nil, fmt.Errorf("error: tree root or name should not be empty")
	}

	if name == t.Current().Name {
		return nil, fmt.Errorf("error: cannot delete current tree (in use)")
	}

	path := filepath.Join(t.root, name)

	return &Entry{
		Name:     name,
		Path:     path,
		Platform: platform.Current(),
	}, os.Remove(path)
}

func (t *TreeEmbedded) Get(name string) (*Entry, error) {
	path := filepath.Join(t.root, name)

	if !filepathx.FileExists(path) {
		return nil, fmt.Errorf("%s does not exist: at %s", name, path)
	}

	return &Entry{
		Name:     name,
		Path:     path,
		Platform: platform.Current(),
	}, nil
}

func (t *TreeEmbedded) Init(_ string, _ *platform.Platform, force bool) (*Entry, error) {
	if t.root == "" {
		return nil, fmt.Errorf("error: tree root should not be empty")
	}

	if filepathx.FileExists(t.root) && !force {
		return nil, fmt.Errorf("already exists: %s", t.root)
	}

	if filepathx.FileExists(t.root) && force {
		err := os.RemoveAll(t.root)
		if err != nil {
			return nil, err
		}
	}

	return &Entry{
		Name:     filepath.Base(t.root),
		Path:     t.root,
		Platform: platform.Current(),
	}, os.MkdirAll(filepath.Join(t.root, names.TreeDataDir), 0755)
}

func (t *TreeEmbedded) List() ([]*Entry, error) {
	return []*Entry{
		{
			Name:     filepath.Base(t.root),
			Path:     t.root,
			Platform: t.platform,
			Current:  true,
		},
	}, nil
}

func (t *TreeEmbedded) Lock() error {
	return t.lock.Lock()
}

func (t *TreeEmbedded) Pki() *pki.Pki {
	return t.pki
}

func (t *TreeEmbedded) Pool(platforms []*platform.Platform, empty bool) (*ops.Pool, error) {
	return pool(t, platforms, empty)
}

func (t *TreeEmbedded) Ready() bool {
	if privileged(t.root) && !sys.IsPrivilegedUser() {
		return false
	}
	return filepathx.FileExists(filepath.Join(t.root, names.TreeDataDir))
}

func (t *TreeEmbedded) RepoLoad(platforms []*platform.Platform, repo *ops.Repository, all bool) error {
	return load(t, platforms, repo, all)
}

func (t *TreeEmbedded) Security() security.Security {
	return t.sec
}

func (t *TreeEmbedded) State() *state.State {
	return t.state
}

func (t *TreeEmbedded) StateToRepo() (*ops.Repository, error) {
	return stateToRepo(t)
}

func (t *TreeEmbedded) Switch(_ string) error {
	return nil
}

func (t *TreeEmbedded) Unlock() error {
	defer os.Remove(t.lock.Path())
	return t.lock.Unlock()
}

func pool(tree Tree, platforms []*platform.Platform, empty bool) (*ops.Pool, error) {
	var repos []*ops.Repository

	for _, repo := range tree.Config().Repositories {
		if repo.Enabled {
			err := tree.RepoLoad(platforms, &repo, false)
			if err != nil {
				return nil, err
			}

			repos = append(repos, &repo)
		}
	}

	frozenEntries, err := tree.State().Frozen.All()
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
		rpState, err = tree.StateToRepo()
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

func load(tree Tree, platforms []*platform.Platform, repo *ops.Repository, all bool) error {
	for _, pltfrm := range platforms {
		metadataPath := tree.Cache().GetMeta(pltfrm.String(), repo.SafeUri())

		md := metadata.New(metadataPath, repo.Prune)
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

		channels, err := md.Channels.Entries()
		if err != nil {
			return err
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

func privileged(root string) bool {
	if !filepathx.FileExists(root) {
		root = filepath.Dir(root)
	}

	stat, _ := os.Stat(root)
	if stat != nil {
		if stat.Sys().(*syscall.Stat_t).Uid != 0 {
			return false
		}
	}

	return true
}

func stateToRepo(tree Tree) (*ops.Repository, error) {
	packages, err := tree.State().Packages.All()
	if err != nil {
		return nil, err
	}

	var headers ops.Headers
	for _, manifest := range packages {
		headers = append(headers, manifest.Header)
	}

	uri, _ := url.Parse("tree://" + tree.Current().Name)
	repo := ops.NewRepo(*uri, true, -1)

	err = repo.Load(tree.Config().Platform(), nil, headers)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
