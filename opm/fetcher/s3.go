package fetcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	s3x "github.com/platform-engineering-labs/orbital/x/aws/s3"
)

const S3 Type = "s3"

type S3Fetcher struct {
	*Ftch
	s3 *s3.Client
}

func NewS3Fetcher(ftch *Ftch) (*S3Fetcher, error) {
	client, err := s3x.GetS3Client()
	if err != nil {
		return nil, err
	}

	return &S3Fetcher{ftch, client}, nil
}

func (s *S3Fetcher) Fetch(pkg *ops.Header) error {
	cacheFile := s.cache.GetFile(pkg.FileName())

	// Download package if not in cache
	if !s.cache.Exists(cacheFile) {
		dst, err := os.OpenFile(cacheFile, os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer dst.Close()

		transfer := transfermanager.New(s.s3)

		_, err = transfer.DownloadObject(context.Background(), &transfermanager.DownloadObjectInput{
			Bucket:   aws.String(s.repo.Uri.Host),
			Key:      aws.String(path.Join(strings.TrimPrefix(s.repo.Uri.Path, "/"), pkg.Platform().String(), pkg.FileName())),
			WriterAt: dst,
		})
	}

	if s.sec.Mode() != security.None {
		validator := opkg.NewValidator(s.Logger, s.sec, true)

		err := validator.Validate(cacheFile)
		if err != nil {
			os.Remove(cacheFile)

			return errors.New(fmt.Sprintf("failed to validate signature and contents: %s", pkg.FileName()))
		}
	}

	return nil
}

func (s *S3Fetcher) Refresh() error {
	for _, pltfrm := range platform.SupportedPlatforms {
		dst, err := os.OpenFile(s.cache.GetMeta(pltfrm.String(), s.repo.SafeUri()), os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}

		transfer := transfermanager.New(s.s3)

		_, err = transfer.DownloadObject(context.Background(), &transfermanager.DownloadObjectInput{
			Bucket:   aws.String(s.repo.Uri.Host),
			Key:      aws.String(path.Join(strings.TrimPrefix(s.repo.Uri.Path, "/"), pltfrm.String(), "metadata.db")),
			WriterAt: dst,
		})
		if err != nil {
			var noSuchKey *types.NoSuchKey
			if errors.As(err, &noSuchKey) {
				dst.Close()
				os.Remove(s.cache.GetMeta(pltfrm.String(), s.repo.SafeUri()))
				continue
			} else {
				return err
			}
		}
		defer dst.Close()

		if s.sec.Mode() != security.None {
			metaData := metadata.New(s.cache.GetMeta(pltfrm.String(), s.repo.SafeUri()), s.repo.Prune)
			defer metaData.Close()

			_, err := s.sec.VerifyMetadata(metaData, *s.repo.Publisher())
			if err != nil {
				os.Remove(s.cache.GetMeta(pltfrm.String(), s.repo.SafeUri()))

				return errors.New(fmt.Sprintf("failed to validate signature and contents: %s %s", pltfrm.String(), s.repo.SafeUri()))
			}
		}

	}

	return nil
}
