package opkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/schema"
	"github.com/platform-engineering-labs/orbital/schema/names"
	filepathx "github.com/platform-engineering-labs/orbital/x/filepath"
)

var OpkgFile = opkgFile{}

type opkgFile struct{}

func (opkg opkgFile) Load(path string, pltfrm *platform.Platform) (*ops.Manifest, string, error) {
	manifest := &ops.Manifest{}
	if pltfrm == nil {
		return nil, "", fmt.Errorf("platform cannot be nil")
	}

	var err error
	var manifestSrc *pkl.ModuleSource

	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}

	if !strings.HasSuffix(path, names.OpkgFile) {
		path = filepath.Join(path, names.OpkgFile)
	}

	path = filepathx.MustAbs(path)

	if filepathx.FileExists(path) {
		manifestSrc = pkl.FileSource(path)
	} else {
		return nil, "", fmt.Errorf("OpkgFile: %s does not exist", path)
	}

	evaluator, err := pkl.NewEvaluator(context.Background(), pkl.WithFs(schema.Schema, "orbital"), pkl.PreconfiguredOptions, func(opts *pkl.EvaluatorOptions) {
		opts.Properties = map[string]string{
			"os":   pltfrm.OS.String(),
			"arch": pltfrm.Arch.String(),
		}
	})
	if err != nil {
		return nil, "", err
	}
	defer evaluator.Close()

	if err = evaluator.EvaluateModule(context.Background(), manifestSrc, &manifest); err != nil {
		return nil, "", err
	}

	// Set the Time in the manifest version
	manifest.Version.Timestamp = time.Now().UTC()

	return manifest, path, nil
}
