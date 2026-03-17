package opkg

import (
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/opkg/payload"
	"github.com/platform-engineering-labs/orbital/opm/phase"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/provider"
	"github.com/platform-engineering-labs/orbital/schema/names"
)

type Builder struct {
	log *slog.Logger

	options *provider.Options

	opfPath string

	version uint8

	manifest *ops.Manifest
	platform *platform.Platform

	header  *Header
	payload *payload.Writer

	writer *Writer
}

func NewBuilder(log *slog.Logger) *Builder {
	builder := &Builder{log: log}

	builder.version = Version

	builder.options = &provider.Options{}

	builder.manifest = &ops.Manifest{}

	builder.header = NewHeader(Version, Compression)
	builder.payload = payload.NewWriter("", 0)
	builder.writer = NewWriter()

	return builder
}

func (b *Builder) Platform(p *platform.Platform) *Builder {
	if p != nil {
		b.platform = p
	}

	return b
}

func (b *Builder) TargetPath(tp string) *Builder {
	b.options.TargetPath = tp
	return b
}

func (b *Builder) Restrict(r bool) *Builder {
	b.options.Restrict = r
	return b
}

func (b *Builder) Secure(s bool) *Builder {
	b.options.Secure = s
	return b
}

func (b *Builder) WorkPath(wp string) *Builder {
	b.options.WorkPath = wp
	b.payload.WorkPath = wp
	return b
}

func (b *Builder) OutputPath(op string) *Builder {
	b.options.OutputPath = op
	return b
}

func (b *Builder) Version(version uint8) *Builder {
	b.version = version
	b.header.Version = version
	return b
}

func (b *Builder) Build(path string) (*ops.Manifest, string, error) {
	var err error

	b.manifest, b.opfPath, err = OpkgFile.Load(path, b.platform)
	if err != nil {
		return nil, "", err
	}

	err = b.setPaths()
	if err != nil {
		return nil, "", err
	}

	err = b.resolve()
	if err != nil {
		return nil, "", err
	}

	err = b.realize()
	if err != nil {
		return nil, "", err
	}

	pkgPath := filepath.Join(b.options.OutputPath, b.manifest.FileName())

	err = b.writer.Write(pkgPath, b.header, b.manifest, b.payload)
	if err != nil {
		return nil, "", err
	}

	return b.manifest, pkgPath, nil
}

func (b *Builder) setPaths() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if b.options.TargetPath == "" {
		dir, _ := path.Split(b.opfPath)
		b.options.TargetPath = path.Join(dir, names.OpkgTarget)
	} else {
		b.options.TargetPath, _ = filepath.Abs(b.options.TargetPath)
	}
	if b.options.WorkPath == "" {
		b.options.WorkPath = wd
		b.payload.WorkPath = wd
	}
	if b.options.OutputPath == "" {
		b.options.OutputPath, _ = path.Split(b.opfPath)
	} else {
		b.options.OutputPath, _ = filepath.Abs(b.options.OutputPath)
	}

	return err
}

func (b *Builder) resolve() error {
	// If restrict is set don't walk the target path
	// this will result in only defined file system objects being added
	// to the package
	if b.options.Restrict == true {
		return nil
	}

	err := filepath.Walk(b.options.TargetPath, func(path string, f os.FileInfo, err error) error {
		objectPath := strings.Replace(path, b.options.TargetPath+string(os.PathSeparator), "", 1)

		if objectPath != b.options.TargetPath {
			if f.IsDir() {
				var dir = action.NewDir()
				dir.Path = objectPath

				if !b.manifest.Exists(dir) {
					b.manifest.Add(dir)
				}
			}

			if f.Mode().IsRegular() {
				var file = action.NewFile()
				file.Path = objectPath

				if !b.manifest.Exists(file) {
					b.manifest.Add(file)
				}
			}

			if f.Mode()&os.ModeSymlink == os.ModeSymlink {
				var symlink = action.NewSymLink()
				symlink.Path = objectPath

				if !b.manifest.Exists(symlink) {
					b.manifest.Add(symlink)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return b.manifest.Validate()
}

// Completes manifest, builds payload
func (b *Builder) realize() error {
	var err error

	// Setup context
	ctx := context.WithValue(context.Background(), "options", b.options)
	ctx = context.WithValue(ctx, "phase", phase.PACKAGE)
	ctx = context.WithValue(ctx, "payload", b.payload)

	factory := provider.DefaultFactory(b.log)

	for _, act := range b.manifest.Actions {
		err = factory.Get(act).Realize(ctx)
		if err != nil {
			return err
		}
	}

	return err
}
