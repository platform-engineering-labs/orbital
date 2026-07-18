package publisher

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/provider"
)

type Publisher interface {
	Init() error
	Channel(pkg *ops.Id, channels []string) error
	Yank(pkg string) error
	Publish(pkgs []string, channels []string) (published []string, pruned []string, err error)
}

type Type string

type Pub struct {
	*slog.Logger

	repo    *ops.Repository
	signing *pki.Signing

	opts *provider.Options
}

func New(log *slog.Logger, opts *provider.Options, signing *pki.Signing, repo *ops.Repository) (Publisher, error) {
	pub := &Pub{
		Logger:  log,
		repo:    repo,
		signing: signing,
		opts:    opts,
	}

	switch Type(repo.UriPublish.Scheme) {
	case Fs:
		return NewFsPublisher(pub), nil
	case S3:
		return NewS3Publisher(pub)
	default:
		return nil, fmt.Errorf("unsupported repository type: %s", repo.Uri.Scheme)
	}
}

func HeadersFromFileList(pkgs []string, workPath string) (map[string]*ops.Header, error) {
	headers := make(map[string]*ops.Header)

	for _, file := range pkgs {
		reader := opkg.NewReader(file, workPath)

		err := reader.Read()
		if err != nil {
			return nil, err
		}

		headers[file] = reader.Manifest.Header
	}

	return headers, nil
}
