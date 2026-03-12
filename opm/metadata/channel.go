package metadata

import (
	"sort"

	"github.com/platform-engineering-labs/orbital/ops"
)

type Channel struct {
	Name string `storm:"id"`

	EntryIds []*ops.Id
}

func (c *Channel) IdMap() map[string][]*ops.Id {
	idMap := make(map[string][]*ops.Id)

	for _, entryId := range c.EntryIds {
		idMap[entryId.Name] = append(idMap[entryId.Name], entryId)
	}

	for name, _ := range idMap {
		sort.Slice(idMap[name], func(i, j int) bool {
			return idMap[name][i].Version.GT(idMap[name][j].Version)
		})
	}

	return idMap
}

func (c *Channel) Prune(count int) {
	idMap := c.IdMap()
	
	var result []*ops.Id

	for name := range idMap {
		for len(idMap[name]) > count {
			idMap[name] = idMap[name][:len(idMap[name])-1]
		}

		result = append(result, idMap[name]...)
	}

	c.EntryIds = result
}
