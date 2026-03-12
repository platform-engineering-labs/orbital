package ops

import (
	"github.com/platform-engineering-labs/orbital/platform"
)

type InventoryRecord struct {
	Platform *platform.Platform
	Channel  *Channel
	Packages []*Header
}

type Inventory []*InventoryRecord

func (slice Inventory) Len() int {
	return len(slice)
}

func (slice Inventory) Less(i, j int) bool {
	if slice[i].Platform.String() < slice[j].Platform.String() {
		return true
	}
	if slice[i].Platform.String() > slice[j].Platform.String() {
		return false
	}

	return slice[i].Channel.Name < slice[j].Channel.Name
}

func (slice Inventory) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
