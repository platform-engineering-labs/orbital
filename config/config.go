package config

import (
	"context"
	"fmt"

	"github.com/apple/pkl-go/pkl"
	"github.com/platform-engineering-labs/orbital/schema"
	"github.com/platform-engineering-labs/orbital/schema/paths"
	filepathx "github.com/platform-engineering-labs/orbital/x/filepath"
)

func init() {
	pkl.RegisterMapping("ops.Config#Mode", new(Mode))
}

type Config struct {
	Mode     Mode   `pkl:"mode"`
	TreeRoot string `pkl:"treeRoot"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	var cfgSrc *pkl.ModuleSource
	if path == "" && !filepathx.FileExists(paths.ConfigFileDefault()) {
		cfgSrc = pkl.TextSource(`amends "orbital:/config.pkl"`)
	} else {
		path = filepathx.MustAbs(path)

		if filepathx.FileExists(path) {
			cfgSrc = pkl.FileSource(path)
		} else {
			return nil, fmt.Errorf("config file: %s does not exist", path)
		}
	}

	evaluator, err := pkl.NewEvaluator(context.Background(), pkl.WithFs(schema.Schema, "orbital"), pkl.PreconfiguredOptions, func(opts *pkl.EvaluatorOptions) {
		opts.Properties = map[string]string{
			"treeRoot": paths.TreeRootDefault(),
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
