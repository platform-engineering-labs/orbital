package ops

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/platform-engineering-labs/orbital/platform"
)

type Repository struct {
	Uri        url.URL `pkl:"uri"`
	UriPublish url.URL `pkl:"uriPublish"`

	Priority int  `pkl:"priority"`
	Enabled  bool `pkl:"enabled"`

	Prune int `pkl:"prune"`

	Channels map[*platform.Platform][]*Channel
	Packages map[*platform.Platform][]*Header
}

type Repos []*Repository

func (slice Repos) Len() int {
	return len(slice)
}

func (slice Repos) Less(i, j int) bool {
	return slice[i].Priority < slice[j].Priority
}

func (slice Repos) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func NewRepo(uri url.URL, enabled bool, priority int) *Repository {
	return &Repository{
		Uri:        uri,
		UriPublish: uri,
		Enabled:    enabled,
		Priority:   priority,

		Channels: make(map[*platform.Platform][]*Channel),
		Packages: make(map[*platform.Platform][]*Header),
	}
}

func (repo *Repository) Inventory() Inventory {
	var inventory Inventory

	for pltfrm, channels := range repo.Channels {
		for _, channel := range channels {
			inv := &InventoryRecord{
				Platform: pltfrm,
				Channel:  channel,
			}

			for _, pkg := range repo.Packages[pltfrm] {
				if slices.ContainsFunc(inv.Channel.EntryIds, func(id *Id) bool {
					return id.String() == pkg.Id().String()
				}) {
					inv.Packages = append(inv.Packages, pkg)
				}
			}

			inventory = append(inventory, inv)
		}

		all := &InventoryRecord{
			Platform: pltfrm,
			Channel:  &Channel{Name: "all"},
		}

		ids := make(map[string]bool)
		for _, channel := range repo.Channels[pltfrm] {
			for _, entryId := range channel.EntryIds {
				ids[entryId.String()] = true
			}
		}

		for _, pkg := range repo.Packages[pltfrm] {
			if _, exists := ids[pkg.Id().String()]; !exists {
				all.Packages = append(all.Packages, pkg)
			}
		}

		if len(all.Packages) > 0 {
			inventory = append(inventory, all)
		}
	}

	sort.Sort(inventory)
	return inventory
}

func (repo *Repository) Load(pltfrm *platform.Platform, channels []*Channel, headers Headers) error {
	if repo.Channels == nil {
		repo.Channels = make(map[*platform.Platform][]*Channel)
	}
	if repo.Packages == nil {
		repo.Packages = make(map[*platform.Platform][]*Header)
	}

	for _, channel := range channels {
		repo.Channels[pltfrm] = append(repo.Channels[pltfrm], channel)
	}

	sort.Sort(headers)
	for _, header := range headers {
		repo.Packages[pltfrm] = append(repo.Packages[pltfrm], header)
	}

	return nil
}

func (repo *Repository) Name() *string {
	parts := strings.Split(repo.Uri.Path, "/")

	if len(parts) < 2 {
		return nil
	} else {
		return &parts[len(parts)-1]
	}
}

func (repo *Repository) Platforms() []*platform.Platform {
	platforms := slices.Collect(maps.Keys(repo.Packages))

	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].String() < platforms[j].String()
	})

	return platforms
}

func (repo *Repository) Publisher() *string {
	parts := strings.Split(repo.Uri.Path, "/")

	if len(parts) < 2 {
		return nil
	} else {
		return &parts[len(parts)-2]
	}
}

func (repo *Repository) SafeUri() string {
	return fmt.Sprintf("%s://%s%s", repo.Uri.Scheme, repo.Uri.Host, repo.Uri.Path)
}

func (repo *Repository) SafePublishUri() string {
	return fmt.Sprintf("%s://%s%s", repo.UriPublish.Scheme, repo.UriPublish.Host, repo.UriPublish.Path)
}
