package publisher

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	s3x "github.com/platform-engineering-labs/orbital/x/aws/s3"
	"github.com/platform-engineering-labs/orbital/x/collections"
)

const S3 Type = "s3"

type S3Publisher struct {
	*Pub
	s3 *s3.Client
}

func NewS3Publisher(pub *Pub) (*S3Publisher, error) {
	client, err := s3x.GetS3Client()
	if err != nil {
		return nil, err
	}

	return &S3Publisher{Pub: pub, s3: client}, nil
}

func (s *S3Publisher) Init() error {
	lock, err := s3x.NewLock(s.repo.SafePublishUri(), s.s3)
	if err != nil {
		return err
	}

	_, err = lock.LockWithError(context.Background())
	if err != nil {
		return err
	}
	defer lock.Unlock()

	result, err := s.s3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:  &s.repo.UriPublish.Host,
		MaxKeys: aws.Int32(200),
		Prefix:  aws.String(strings.TrimPrefix(s.repo.UriPublish.Path, "/") + "/"),
	})
	if err != nil {
		return err
	}

	if len(result.Contents) == 0 {
		return nil
	}

	_, err = s.s3.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &s.repo.UriPublish.Host,
		Delete: &types.Delete{
			Objects: collections.Map(result.Contents, func(item types.Object) types.ObjectIdentifier {
				return types.ObjectIdentifier{
					Key: item.Key,
				}
			}),
			Quiet: aws.Bool(true),
		},
	})
	if err != nil {
		return err
	}

	_, err = s.s3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.repo.UriPublish.Host),
		Key:    aws.String(strings.TrimPrefix(s.repo.UriPublish.Path, "/") + "/"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Publisher) Channel(id *ops.Id, channels []string) error {
	for _, pltfrm := range platform.SupportedPlatforms {
		s3Path := filepath.Join(strings.TrimPrefix(s.repo.UriPublish.Path, "/"), pltfrm.String()) + "/"
		lockUri, err := url.JoinPath(s.repo.SafePublishUri(), pltfrm.String())
		if err != nil {
			return err
		}

		tmpDir, err := os.MkdirTemp(s.opts.WorkPath, "publish")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		metaPath := filepath.Join(tmpDir, "metadata.db")
		metadataTmp, err := os.Create(metaPath)
		if err != nil {
			return err
		}
		defer metadataTmp.Close()

		transfer := transfermanager.New(s.s3)

		_, err = transfer.DownloadObject(context.Background(), &transfermanager.DownloadObjectInput{
			Bucket:   aws.String(s.repo.UriPublish.Host),
			Key:      aws.String(path.Join(s3Path, "metadata.db")),
			WriterAt: metadataTmp,
		})
		if err != nil {
			var noSuchKey *types.NoSuchKey
			if errors.As(err, &noSuchKey) {
				continue
			} else {
				return err
			}
		}

		lock, err := s3x.NewLock(lockUri, s.s3)
		if err != nil {
			return err
		}

		_, err = lock.LockWithError(context.Background())
		if err != nil {
			return err
		}
		defer lock.Unlock()

		metaData := metadata.New(metaPath, s.repo.Prune)
		defer metaData.Close()

		for _, channel := range channels {
			err := metaData.Channels.Add(id.String(), channel)
			if err != nil {
				return err
			}
		}

		_, err = transfer.UploadObject(context.Background(), &transfermanager.UploadObjectInput{
			Bucket: aws.String(s.repo.UriPublish.Host),
			Key:    aws.String(path.Join(s3Path, "metadata.db")),
			Body:   metadataTmp,
		})
		if err != nil {
			return fmt.Errorf("unable to upload metadata file: %s, err: %s", path.Join(s3Path, "metadata.db"), err.Error())
		}
	}

	s.Info(fmt.Sprintf("added: %s to %s", id.String(), strings.Join(channels, ",")))

	return nil
}

func (s *S3Publisher) Yank(pkg string) error {
	for _, pltfrm := range platform.SupportedPlatforms {
		s3Path := filepath.Join(strings.TrimPrefix(s.repo.UriPublish.Path, "/"), pltfrm.String()) + "/"
		lockUri, err := url.JoinPath(s.repo.SafePublishUri(), pltfrm.String())
		if err != nil {
			return err
		}

		kp, err := s.sec.KeyPair(*s.repo.Publisher())
		if err != nil {
			return err
		}

		if kp == nil {
			s.Warn(fmt.Sprintf("no keypair found for publisher %s, not signing.", *s.repo.Publisher()))
		}

		tmpDir, err := os.MkdirTemp(s.opts.WorkPath, "publish")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		metaPath := filepath.Join(tmpDir, "metadata.db")
		metadataTmp, err := os.Create(metaPath)
		if err != nil {
			return err
		}
		defer metadataTmp.Close()

		transfer := transfermanager.New(s.s3)

		_, err = transfer.DownloadObject(context.Background(), &transfermanager.DownloadObjectInput{
			Bucket:   aws.String(s.repo.UriPublish.Host),
			Key:      aws.String(path.Join(s3Path, "metadata.db")),
			WriterAt: metadataTmp,
		})
		if err != nil {
			var noSuchKey *types.NoSuchKey
			if errors.As(err, &noSuchKey) {
				continue
			} else {
				return err
			}
		}

		lock, err := s3x.NewLock(lockUri, s.s3)
		if err != nil {
			return err
		}

		_, err = lock.LockWithError(context.Background())
		if err != nil {
			return err
		}
		defer lock.Unlock()

		metaData := metadata.New(metaPath, s.repo.Prune)
		defer metaData.Close()

		entry, err := metaData.Packages.DelR(pkg)
		if err != nil {
			return err
		}

		if entry != nil && metaData.Packages.Count("") > 0 {
			if kp != nil {
				content, err := metaData.ToSigningJson()
				if err != nil {
					return err
				}

				rsaKey, err := kp.RSAKey()
				if err != nil {
					return err
				}

				sigAction, err := security.SignBytes(&content, kp.SKI, kp.Fingerprint, rsaKey, "sha256")
				if err != nil {
					return err
				}

				err = metaData.Signatures.Put(sigAction)
				if err != nil {
					return err
				}
			}

			_, err = s.s3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
				Bucket: aws.String(s.repo.UriPublish.Host),
				Key:    aws.String(path.Join(s3Path, entry.FileName())),
			})
			if err != nil {
				var noSuchKey *types.NoSuchKey
				if !errors.As(err, &noSuchKey) {
					return fmt.Errorf("unable to delete file: %s, err: %s", entry.FileName(), err.Error())
				}
			}

			_, err = transfer.UploadObject(context.Background(), &transfermanager.UploadObjectInput{
				Bucket: aws.String(s.repo.UriPublish.Host),
				Key:    aws.String(path.Join(s3Path, "metadata.db")),
				Body:   metadataTmp,
			})
			if err != nil {
				return fmt.Errorf("unable to upload metadata file: %s, err: %s", path.Join(s3Path, "metadata.db"), err.Error())
			}

			s.Info(fmt.Sprintf("yanked: %s", entry.FileName()))
		}
	}

	return nil
}

func (s *S3Publisher) Publish(pkgs []string, channels []string) (published []string, pruned []string, err error) {
	pkgMap, err := HeadersFromFileList(pkgs, s.opts.WorkPath)
	if err != nil {
		return nil, nil, err
	}

	kp, err := s.sec.KeyPair(*s.repo.Publisher())
	if err != nil {
		return nil, nil, err
	}

	if kp == nil {
		s.Warn(fmt.Sprintf("no keypair found for publisher %s, not signing.", *s.repo.Publisher()))
	}

	for _, pltfrm := range platform.SupportedPlatforms {
		current := maps.Collect(collections.FilterMap(pkgMap, func(k string, v *ops.Header) bool {
			return pltfrm.Equal(v.Platform())
		}))

		if len(current) > 0 {
			s3Path := filepath.Join(strings.TrimPrefix(s.repo.UriPublish.Path, "/"), pltfrm.String()) + "/"
			lockUri, err := url.JoinPath(s.repo.SafePublishUri(), pltfrm.String())
			if err != nil {
				return nil, nil, err
			}

			tmpDir, err := os.MkdirTemp(s.opts.WorkPath, "publish")
			if err != nil {
				return nil, nil, err
			}
			defer os.RemoveAll(tmpDir)

			metaPath := filepath.Join(tmpDir, "metadata.db")

			_, err = s.s3.PutObject(context.Background(), &s3.PutObjectInput{
				Bucket: aws.String(s.repo.UriPublish.Host),
				Key:    aws.String(s3Path),
			})
			if err != nil {
				return nil, nil, err
			}

			lock, err := s3x.NewLock(lockUri, s.s3)
			if err != nil {
				return nil, nil, err
			}

			_, err = lock.LockWithError(context.Background())
			if err != nil {
				return nil, nil, err
			}
			defer lock.Unlock()

			metadataTmp, err := os.Create(metaPath)
			if err != nil {
				return nil, nil, err
			}
			defer metadataTmp.Close()

			transfer := transfermanager.New(s.s3)

			_, err = transfer.DownloadObject(context.Background(), &transfermanager.DownloadObjectInput{
				Bucket:   aws.String(s.repo.UriPublish.Host),
				Key:      aws.String(path.Join(s3Path, "metadata.db")),
				WriterAt: metadataTmp,
			})
			if err != nil {
				var noSuchKey *types.NoSuchKey
				if !errors.As(err, &noSuchKey) {
					return nil, nil, fmt.Errorf("unable to download metadata file: %s, err: %s", path.Join(s3Path, "metadata.db"), err.Error())
				}
			}

			metaData := metadata.New(metaPath, s.repo.Prune)
			defer metaData.Close()

			entries, pEntries, err := metaData.Packages.PutAll(slices.Collect(maps.Values(current)), channels)
			if err != nil {
				return nil, nil, err
			}

			if metaData.Packages.Count("") > 0 {
				if kp != nil {
					content, err := metaData.ToSigningJson()
					if err != nil {
						return nil, nil, err
					}

					rsaKey, err := kp.RSAKey()
					if err != nil {
						return nil, nil, err
					}

					sigAction, err := security.SignBytes(&content, kp.SKI, kp.Fingerprint, rsaKey, "sha256")
					if err != nil {
						return nil, nil, err
					}

					err = metaData.Signatures.Put(sigAction)
					if err != nil {
						return nil, nil, err
					}
				}

				_, err := transfer.UploadObject(context.Background(), &transfermanager.UploadObjectInput{
					Bucket: aws.String(s.repo.UriPublish.Host),
					Key:    aws.String(path.Join(s3Path, "metadata.db")),
					Body:   metadataTmp,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("unable to upload metadata file: %s, err: %s", path.Join(s3Path, "metadata.db"), err.Error())
				}

				for file, header := range current {
					if slices.ContainsFunc(entries, func(e *metadata.Entry) bool {
						return e.Version.EXQ(header.Version)
					}) {
						ufile, err := os.Open(file)
						if err != nil {
							return nil, nil, err
						}

						_, err = transfer.UploadObject(context.Background(), &transfermanager.UploadObjectInput{
							Bucket: aws.String(s.repo.UriPublish.Host),
							Key:    aws.String(path.Join(s3Path, header.FileName())),
							Body:   ufile,
						})
						if err != nil {
							return nil, nil, fmt.Errorf("unable to upload file: %s, err: %s", file, err.Error())
							ufile.Close()
						}
						ufile.Close()

						s.Info(fmt.Sprintf("uploaded: %s", file))

						published = append(published, file)
					}
				}

				for _, remove := range pEntries {
					_, err = s.s3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
						Bucket: aws.String(s.repo.UriPublish.Host),
						Key:    aws.String(path.Join(s3Path, remove.FileName())),
					})
					if err != nil {
						var noSuchKey *types.NoSuchKey
						if !errors.As(err, &noSuchKey) {
							return nil, nil, fmt.Errorf("unable to delete file: %s, err: %s", remove, err.Error())
						}
					}
					pruned = append(pruned, path.Join(s3Path, remove.FileName()))

					s.Info(fmt.Sprintf("removed: %s", path.Join(s3Path, remove.FileName())))
				}
			} else {
				result, err := s.s3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
					Bucket:  &s.repo.UriPublish.Host,
					MaxKeys: aws.Int32(200),
					Prefix:  aws.String(filepath.Join(strings.TrimPrefix(s.repo.UriPublish.Path, "/"), pltfrm.String()) + "/"),
				})
				if err != nil {
					return nil, nil, err
				}

				if len(result.Contents) == 0 {
					return nil, nil, err
				}

				_, err = s.s3.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
					Bucket: &s.repo.UriPublish.Host,
					Delete: &types.Delete{
						Objects: collections.Map(result.Contents, func(item types.Object) types.ObjectIdentifier {
							return types.ObjectIdentifier{
								Key: item.Key,
							}
						}),
						Quiet: aws.Bool(true),
					},
				})
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	return published, pruned, nil
}
