package fetcher

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/schema/names"
)

const Fs Type = "fs"

type FsFetcher struct {
	*Ftch
}

func NewFsFetcher(ftch *Ftch) *FsFetcher {
	return &FsFetcher{ftch}
}

func (f *FsFetcher) Fetch(pkg *ops.Header) error {
	repoFile := filepath.Join(f.repo.Uri.Path, pkg.Platform().String(), pkg.FileName())
	cacheFile := f.cache.GetFile(pkg.FileName())

	lock := flock.New(filepath.Join(f.repo.Uri.Path, pkg.Platform().String(), ".lock"))
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()
	defer os.Remove(lock.Path())

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

func (f *FsFetcher) Refresh() error {
	for _, pltfrm := range platform.SupportedPlatforms {
		metadataPath := filepath.Join(f.repo.Uri.Path, pltfrm.String(), names.MetaDataDb)

		if _, err := os.Stat(filepath.Join(f.repo.Uri.Path, pltfrm.String())); os.IsNotExist(err) {
			os.Remove(f.cache.GetMeta(pltfrm.String(), f.repo.SafeUri()))
			continue
		}

		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			os.Remove(f.cache.GetMeta(pltfrm.String(), f.repo.SafeUri()))
			continue
		}

		lock := flock.New(filepath.Join(f.repo.Uri.Path, pltfrm.String(), ".lock"))
		if err := lock.Lock(); err != nil {
			return err
		}
		defer lock.Unlock()
		defer os.Remove(lock.Path())

		// Fetch meta
		src, err := os.Open(metadataPath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.OpenFile(f.cache.GetMeta(pltfrm.String(), f.repo.SafeUri()), os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}

		if f.sec.Mode() != security.None {
			metaData := metadata.New(f.cache.GetMeta(pltfrm.String(), f.repo.SafeUri()), f.repo.Prune)
			defer metaData.Close()

			_, err := f.sec.VerifyMetadata(metaData, *f.repo.Publisher())
			if err != nil {
				os.Remove(f.cache.GetMeta(pltfrm.String(), f.repo.SafeUri()))

				return errors.New(fmt.Sprintf("failed to validate signature and contents: %s %s", pltfrm.String(), f.repo.SafeUri()))
			}
		}

	}

	return nil
}
