package mgr

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/opm/records"
	"github.com/platform-engineering-labs/orbital/opm/tree"
	"github.com/platform-engineering-labs/orbital/platform"
)

type Manager struct {
	*slog.Logger
	Path string

	cfg *tree.Config
	orb *orbital.Orbital
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

func (m *Manager) AvailableFor(name string) (*records.Status, error) {
	var available *records.Status

	available, err := m.orb.Status(name)
	if err != nil {
		return nil, err
	}

	if len(available.Available) == 0 {
		return nil, fmt.Errorf("no available packages for: %s", name)
	}

	return available, nil
}

func (m *Manager) Clear() error {
	return m.orb.Cache.Clear()
}

func (m *Manager) Clean() error {
	return m.orb.Cache.Clean()
}

func (m *Manager) Initialize() (*tree.Entry, error) {
	entry, err := m.orb.Tree.Init("", nil, false)
	if err != nil {
		return nil, err
	}

	m.orb, err = orbital.Embedded(m.Logger, m.Path, m.cfg)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (m *Manager) Install(packages ...string) error {
	return m.orb.Install(packages...)
}

func (m *Manager) List() ([]*records.Package, error) {
	return m.orb.List()
}

func (m *Manager) Privileged() bool {
	return m.orb.Privileged()
}

func (m *Manager) Ready() bool {
	return m.orb.Ready()
}

func (m *Manager) Refresh() error {
	return m.orb.Refresh()
}

func (m *Manager) Remove(packages ...string) error {
	return m.orb.Remove(packages...)
}

func (m *Manager) Update(packages ...string) error {
	return m.orb.Update(packages...)
}
