package mgr

import (
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/opm/records"
	"github.com/platform-engineering-labs/orbital/opm/tree"
	"github.com/platform-engineering-labs/orbital/platform"
)

type Manager struct {
	*slog.Logger
	path string
	cfg  *tree.Config
	orb  *orbital.Orbital
}

func New(log *slog.Logger, path string, cfg *tree.Config) (*Manager, error) {
	orb, err := orbital.Embedded(log, path, cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{log, path, cfg, orb}, nil
}

func (m *Manager) Available() (map[string]*records.Status, error) {
	var available = make(map[string]*records.Status)

	pool, err := m.orb.Tree.Pool(platform.Expanded(platform.Current()), true)
	if err != nil {
		return nil, err
	}

	for k, _ := range pool.Available() {
		available[k], err = m.orb.Status(k)
		if err != nil {
			return nil, err
		}
	}

	return available, nil
}

func (m *Manager) Initialize() (*tree.Entry, error) {
	entry, err := m.orb.Tree.Init("", nil, false)
	if err != nil {
		return nil, err
	}

	m.orb, err = orbital.Embedded(m.Logger, m.path, m.cfg)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (m *Manager) Install(name string) error {
	return m.orb.Install(name)
}

func (m *Manager) Ready() bool {
	return m.orb.Ready()
}

func (m *Manager) Refresh() error {
	return m.orb.Refresh()
}

func (m *Manager) Remove(name string) error {
	return m.orb.Remove(name)
}

func (m *Manager) Update(name string) error {
	return m.orb.Update(name)
}
