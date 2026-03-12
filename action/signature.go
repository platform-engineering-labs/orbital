package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platform-engineering-labs/orbital/action/actions"
)

type Signature struct {
	FingerPrint string `json:"fingerprint"`
	SKI         string `json:"ski"`
	Algo        string `json:"algo"`
	Value       string `json:"value"`
}

func NewSignature() *Signature {
	return &Signature{}
}

func (s *Signature) Key() string {
	return s.FingerPrint
}

func (s *Signature) Type() actions.Type {
	return actions.Signature
}

func (s *Signature) Columns() []string {
	return []string{
		strings.ToUpper(s.Type().String()),
		s.FingerPrint,
	}
}

func (s *Signature) Id() string {
	return fmt.Sprint(s.Type(), ".", s.Key())
}

func (s *Signature) IsValid() bool {
	if s.FingerPrint != "" && s.Algo != "" && s.Value != "" {
		return true
	}

	return false
}

func (s *Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Signature
	}{
		Type:      s.Type().String(),
		Signature: *s,
	})
}
