package orbital

import (
	"path/filepath"

	"github.com/platform-engineering-labs/orbital/config"
	"github.com/platform-engineering-labs/orbital/opm/tree"
)

type Option func(*Orbital) error

func WithConfig(path string) Option {
	return func(o *Orbital) error {
		var err error

		o.config, err = config.Load(path)
		if err != nil {
			return err
		}

		return nil
	}
}

func WithEmbedded(path string, cfg *tree.Config) Option {
	return func(o *Orbital) error {
		var err error

		o.config = &config.Config{
			Mode: config.EmbeddedMode,
		}

		o.tree, err = tree.New(o.Logger, filepath.Base(path), path, o.writeable, cfg)
		if err != nil {
			return err
		}

		return nil
	}
}

func WithCurrent(path string) Option {
	return func(o *Orbital) error {
		var err error

		o.tree, err = tree.New(o.Logger, "$current", path, o.writeable, nil)

		return err
	}
}

func WithWritable() Option {
	return func(o *Orbital) error {
		o.writeable = true

		return nil
	}
}

func WithSudo() Option {
	return func(o *Orbital) error {
		o.sudo = true

		return nil
	}
}
