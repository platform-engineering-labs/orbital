package fetcher

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
)

const File Type = "file"

type FileFetcher struct {
	*Ftch
}

func NewFileFetcher(ftch *Ftch) *FileFetcher {
	return &FileFetcher{ftch}
}

func (f *FileFetcher) Fetch(pkg *ops.Header) error {
	repoFile := filepath.Join(f.repo.Uri.Path, pkg.FileName())
	cacheFile := f.cache.GetFile(pkg.FileName())

	// Copy package if not in cache
	if !f.cache.Exists(pkg.FileName()) {
		src, err := os.Open(repoFile)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.OpenFile(cacheFile, os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	if f.sec.Mode() != security.None {
		validator := opkg.NewValidator(f.Logger, f.sec, true)

		err := validator.Validate(cacheFile)
		if err != nil {
			os.Remove(cacheFile)

			return errors.New(fmt.Sprintf("failed to validate signature and contents: %s", pkg.FileName()))
		}
	}

	return nil
}

func (f *FileFetcher) Refresh() error {
	return nil
}
