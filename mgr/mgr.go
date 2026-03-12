package mgr

import (
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/config"
)

type Manager struct {
	*slog.Logger
	orb *orbital.Orbital
}

func New(log *slog.Logger, cfg *config.Config) (*Manager, error) {

	orb, err := orbital.Embedded(log, cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{log, orb}, nil
}
