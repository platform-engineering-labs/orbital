package fetcher

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"resty.dev/v3"
)

const HTTPS Type = "https"

type HTTPSFetcher struct {
	*Ftch
	client *resty.Client
}

func NewHTTPSFetcher(ftch *Ftch) (*HTTPSFetcher, error) {
	client := resty.New()
	client.SetTimeout(time.Duration(10) * time.Second)

	return &HTTPSFetcher{ftch, client}, nil
}

func (h *HTTPSFetcher) Fetch(pkg *ops.Header) error {
	cacheFile := h.cache.GetFile(pkg.FileName())

	// Download package if not in cache
	if !h.cache.Exists(pkg.FileName()) {
		dst, err := os.OpenFile(cacheFile, os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer dst.Close()

		// Uri
		fileUri := h.repo.Uri
		fileUri.Fragment = ""
		fileUri.Path = path.Join(fileUri.Path, pkg.Platform().String(), pkg.FileName())

		// Download
		resp, err := h.client.R().
			SetOutputFileName(cacheFile).
			Get(fileUri.String())

		if err != nil {
			return errors.New(fmt.Sprintf("error connecting to: %s", h.repo.Uri.Host))
		}

		if resp.IsError() {
			os.Remove(cacheFile)

			switch resp.StatusCode() {
			case 404:
				return errors.New(fmt.Sprintf("not found: %s", fileUri.String()))
			case 403:
				return errors.New(fmt.Sprintf("access denied: %s", fileUri.String()))
			default:
				return errors.New(fmt.Sprintf("server error %d: %s", resp.StatusCode(), fileUri.String()))
			}
		}
	}

	if h.sec.Mode() != security.None {
		validator := opkg.NewValidator(h.Logger, h.sec, true)

		err := validator.Validate(cacheFile)
		if err != nil {
			os.Remove(cacheFile)

			return errors.New(fmt.Sprintf("failed to validate signature and contents: %s", pkg.FileName()))
		}
	}

	return nil
}

func (h *HTTPSFetcher) Refresh() error {
	for _, pltfrm := range platform.SupportedPlatforms {
		// Uri
		metaUri := h.repo.Uri
		metaUri.Fragment = ""
		metaUri.Path = path.Join(metaUri.Path, pltfrm.String(), "metadata.db")

		resp, err := h.client.R().
			SetOutputFileName(h.cache.GetMeta(pltfrm.String(), h.repo.SafeUri())).
			Get(metaUri.String())

		if err != nil {
			return errors.New(fmt.Sprintf("error connecting to: %s", h.repo.Uri.Host))
		}

		if resp.IsError() {
			os.Remove(h.cache.GetMeta(pltfrm.String(), h.repo.SafeUri()))
			switch resp.StatusCode() {
			case 404:
				continue
			case 403:
				continue
			default:
				return errors.New(fmt.Sprintf("server error %d: %s", resp.StatusCode(), metaUri.String()))
			}
		}

		if h.sec.Mode() != security.None {
			metaData := metadata.New(h.cache.GetMeta(pltfrm.String(), h.repo.SafeUri()), h.repo.Prune)
			defer metaData.Close()

			_, err := h.sec.VerifyMetadata(metaData, *h.repo.Publisher())
			if err != nil {
				os.Remove(h.cache.GetMeta(pltfrm.String(), h.repo.SafeUri()))

				return errors.New(fmt.Sprintf("failed to validate signature and contents: %s %s", pltfrm.String(), h.repo.SafeUri()))
			}
		}

	}

	return nil
}
