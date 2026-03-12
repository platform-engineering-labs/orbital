package publisher

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/gofrs/flock"
	"github.com/platform-engineering-labs/orbital/opm/metadata"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/schema/names"
	"github.com/platform-engineering-labs/orbital/x/collections"
)

const Fs Type = "fs"

type FsPublisher struct {
	*Pub
}

func NewFsPublisher(pub *Pub) *FsPublisher {
	return &FsPublisher{pub}
}

func (f *FsPublisher) Init() error {
	_ = os.MkdirAll(f.repo.UriPublish.Path, os.FileMode(0750))
	lock := flock.New(filepath.Join(f.repo.UriPublish.Path, ".lock"))
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()
	defer os.Remove(lock.Path())

	for _, p := range platform.SupportedPlatforms {
		_ = os.RemoveAll(filepath.Join(f.repo.UriPublish.Path, p.String()))
	}

	return nil
}

func (f *FsPublisher) Channel(id *ops.Id, channels []string) error {
	for _, pltfrm := range platform.SupportedPlatforms {
		path := filepath.Join(f.repo.UriPublish.Path, pltfrm.String())

		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		lock := flock.New(filepath.Join(path, ".lock"))
		if err := lock.Lock(); err != nil {
			return err
		}
		defer lock.Unlock()
		defer os.Remove(lock.Path())

		metaData := metadata.New(filepath.Join(path, names.MetaDataDb), f.repo.Prune)
		defer metaData.Close()

		for _, channel := range channels {
			err := metaData.Channels.Add(id.String(), channel)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *FsPublisher) Publish(pkgs []string, channels []string) (published []string, pruned []string, err error) {
	pkgMap, err := HeadersFromFileList(pkgs, f.opts.WorkPath)
	if err != nil {
		return nil, nil, err
	}

	kp, err := f.sec.KeyPair(*f.repo.Publisher())
	if err != nil {
		return nil, nil, err
	}

	if kp == nil {
		f.Warn(fmt.Sprintf("no keypair found for publisher %s, not signing.", *f.repo.Publisher()))
	}

	for _, pltfrm := range platform.SupportedPlatforms {
		current := maps.Collect(collections.FilterMap(pkgMap, func(k string, v *ops.Header) bool {
			return pltfrm.Equal(v.Platform())
		}))

		if len(current) > 0 {
			_ = os.Mkdir(filepath.Join(f.repo.UriPublish.Path, pltfrm.String()), 0750)

			lock := flock.New(filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), ".lock"))
			if err := lock.Lock(); err != nil {
				return nil, nil, err
			}
			defer lock.Unlock()
			defer os.Remove(lock.Path())

			metaData := metadata.New(filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), names.MetaDataDb), f.repo.Prune)
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

				for file, header := range current {
					if slices.ContainsFunc(entries, func(e *metadata.Entry) bool {
						return e.Version.EXQ(header.Version)
					}) {
						err = f.upload(file, filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), filepath.Base(file)))
						if err != nil {
							return nil, nil, err
						}
						f.Info(fmt.Sprintf("uploaded %s", file))

						published = append(published, file)
					}
				}

				for _, remove := range pEntries {
					_ = os.Remove(filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), remove.FileName()))
					pruned = append(pruned, filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), remove.FileName()))
					f.Info(fmt.Sprintf("Removed %s", remove.FileName()))
				}
			} else {
				_ = os.RemoveAll(filepath.Join(f.repo.UriPublish.Path, pltfrm.String()))
			}
		}
	}

	return published, pruned, nil
}

func (f *FsPublisher) Yank(pkg string) error {
	kp, err := f.sec.KeyPair(*f.repo.Publisher())
	if err != nil {
		return err
	}

	if kp == nil {
		f.Warn(fmt.Sprintf("no keypair found for publisher %s, not signing.", *f.repo.Publisher()))
	}

	for _, pltfrm := range platform.SupportedPlatforms {
		path := filepath.Join(f.repo.UriPublish.Path, pltfrm.String())

		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		lock := flock.New(filepath.Join(path, ".lock"))
		if err := lock.Lock(); err != nil {
			return err
		}
		defer lock.Unlock()
		defer os.Remove(lock.Path())

		metaData := metadata.New(filepath.Join(path, names.MetaDataDb), f.repo.Prune)
		defer metaData.Close()

		entry, err := metaData.Packages.DelR(pkg)
		if err != nil {
			return err
		}

		if entry != nil && metaData.Packages.Count("") > 0 {
			f.Info(fmt.Sprintf("yanked: %s", entry.FileName()))
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

			os.Remove(filepath.Join(f.repo.UriPublish.Path, pltfrm.String(), entry.FileName()))
		}
	}

	return nil
}

func (f *FsPublisher) upload(file string, dest string) error {
	s, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	return d.Close()
}
