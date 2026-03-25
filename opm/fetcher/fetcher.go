package fetcher

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/opm/cache"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Fetcher interface {
	Fetch(pkg *ops.Header) error
	Refresh() error
}

type Type string

type Ftch struct {
	*slog.Logger

	cache *cache.Cache
	repo  *ops.Repository
	sec   security.Security
}

func New(log *slog.Logger, cache *cache.Cache, sec security.Security, repo *ops.Repository) (Fetcher, error) {
	fetch := &Ftch{
		Logger: log,
		cache:  cache,
		repo:   repo,
		sec:    sec,
	}

	switch Type(repo.Uri.Scheme) {
	case File:
		return NewFileFetcher(fetch), nil
	case Fs:
		return NewFsFetcher(fetch), nil
	case HTTPS:
		return NewHTTPSFetcher(fetch)
	case S3:
		return NewS3Fetcher(fetch)
	default:

		return nil, fmt.Errorf("unknown repository type: %s", repo.Uri.Scheme)
	}
}
