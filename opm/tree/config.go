package tree

import (
	"context"
	"fmt"

	"github.com/apple/pkl-go/pkl"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
	"github.com/platform-engineering-labs/orbital/schema"
	filepathx "github.com/platform-engineering-labs/orbital/x/filepath"
)

func init() {
	pkl.RegisterMapping("ops.Tree#Security", new(security.Mode))
}

type Config struct {
	OS   os.OS     `pkl:"os"`
	Arch arch.Arch `pkl:"arch"`

	Security security.Mode `pkl:"security"`

	Repositories []ops.Repository `pkl:"repositories"`
}

func (c *Config) Platform() *platform.Platform {
	return &platform.Platform{OS: c.OS, Arch: c.Arch}
}
func (c *Config) Repository(name string) (*ops.Repository, error) {
	for _, r := range c.Repositories {
		if *r.Name() == name {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("repository %q not found", name)
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	var cfgSrc *pkl.ModuleSource
	if path == "" || !filepathx.FileExists(path) {
		cfgSrc = pkl.TextSource(`amends "orbital:/tree.pkl"`)
	} else {
		cfgSrc = pkl.FileSource(path)
	}

	evaluator, err := pkl.NewEvaluator(context.Background(), pkl.WithFs(schema.Schema, "orbital"), pkl.PreconfiguredOptions, func(opts *pkl.EvaluatorOptions) {
		opts.Properties = map[string]string{
			"os":   platform.Current().OS.String(),
			"arch": platform.Current().Arch.String(),
		}
	})
	if err != nil {
		return nil, err
	}
	defer evaluator.Close()

	if err = evaluator.EvaluateModule(context.Background(), cfgSrc, &cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
