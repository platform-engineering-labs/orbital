package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platform-engineering-labs/orbital/action/actions"
)

type SymLink struct {
	Path   string `json:"path" pkl:"path"`
	Owner  string `json:"owner" pkl:"owner"`
	Group  string `json:"group" pkl:"group"`
	Target string `json:"target" pkl:"target"`
}

func NewSymLink() *SymLink {
	return &SymLink{}
}

func (s *SymLink) Key() string {
	return s.Path
}

func (s *SymLink) Type() actions.Type {
	return actions.SymLink
}

func (s *SymLink) Columns() []string {
	return []string{
		strings.ToUpper(s.Type().String()),
		"",
		s.Owner + ":" + s.Group,
		s.Path,
	}
}

func (s *SymLink) Id() string {
	return fmt.Sprint(s.Type(), ".", s.Key())
}

func (s *SymLink) IsValid() bool {
	if s.Path != "" && s.Owner != "" && s.Group != "" && s.Target != "" {
		return true
	}

	return false
}

func (s *SymLink) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		SymLink
	}{
		Type:    s.Type().String(),
		SymLink: *s,
	})
}
