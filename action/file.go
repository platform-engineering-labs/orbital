package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platform-engineering-labs/orbital/action/actions"
)

type File struct {
	Path  string `json:"path" pkl:"path"`
	Owner string `json:"owner" pkl:"owner"`
	Group string `json:"group" pkl:"group"`
	Mode  string `json:"mode" pkl:"mode"`

	Digest string `json:"digest"`
	Offset int    `json:"offset"`
	Csize  int    `json:"csize"`
	Size   int    `json:"size"`
}

func NewFile() *File {
	return &File{}
}

func (f *File) Key() string {
	return f.Path
}

func (f *File) Type() actions.Type {
	return actions.File
}

func (f *File) Columns() []string {
	return []string{
		strings.ToUpper(f.Type().String()),
		f.Mode,
		f.Owner + ":" + f.Group,
		f.Path,
	}
}

func (f *File) Id() string {
	return fmt.Sprint(f.Type(), ".", f.Key())
}

func (f *File) IsValid() bool {
	if f.Path != "" && f.Owner != "" && f.Group != "" && f.Mode != "" {
		return true
	}

	return false
}

func (f *File) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		File
	}{
		Type: f.Type().String(),
		File: *f,
	})
}
