package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platform-engineering-labs/orbital/action/actions"
)

type Dir struct {
	Path  string `json:"path" pkl:"path"`
	Owner string `json:"owner" pkl:"owner"`
	Group string `json:"group" pkl:"group"`
	Mode  string `json:"mode" pkl:"mode"`
}

func NewDir() *Dir {
	return &Dir{}
}

func (d *Dir) Key() string {
	return d.Path
}

func (d *Dir) Type() actions.Type {
	return actions.Dir
}

func (d *Dir) Columns() []string {
	return []string{
		strings.ToUpper(d.Type().String()),
		d.Mode,
		d.Owner + ":" + d.Group,
		d.Path,
	}
}

func (d *Dir) Id() string {
	return fmt.Sprint(d.Type(), ".", d.Key())
}

func (d *Dir) IsValid() bool {
	if d.Path != "" && d.Owner != "" && d.Group != "" && d.Mode != "" {
		return true
	}

	return false
}

func (d *Dir) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Dir
	}{
		Type: d.Type().String(),
		Dir:  *d,
	})
}
