package opm

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/ops"
)

func ReqsReposFromNames(packages []string) (reqs []string, repos []*ops.Repository, err error) {
	index := make(map[string]*ops.Repository)

	for _, item := range packages {
		if filepath.Ext(item) == ".opkg" {
			if _, err := os.Stat(item); err == nil {
				reader := opkg.NewReader(item, "")
				err := reader.Read()
				if err != nil {
					return nil, nil, err
				}
				defer reader.Close()

				reqs = append(reqs, reader.Manifest.Id().String())

				itemPath, err := filepath.Abs(item)
				if err != nil {
					return nil, nil, err
				}

				path := filepath.Dir(itemPath)

				if index[path] == nil {
					uri := url.URL{
						Scheme: "file",
						Path:   path,
					}

					index[path] = ops.NewRepo(uri, true, 0)
					repos = append(repos, index[path])
				}

				index[path].Add(reader.Manifest.Header)
			}
		} else {
			reqs = append(reqs, item)
		}
	}

	return reqs, repos, err
}
