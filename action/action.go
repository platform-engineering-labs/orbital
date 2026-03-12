package action

import (
	"encoding/json"
	"fmt"

	"github.com/platform-engineering-labs/orbital/action/actions"
)

type Action interface {
	Id() string
	Key() string
	Type() actions.Type
	Columns() []string
	IsValid() bool
}

type Actions []Action

func (a *Actions) UnmarshalJSON(data []byte) error {
	var rawMessages []json.RawMessage
	if err := json.Unmarshal(data, &rawMessages); err != nil {
		return err
	}

	for _, raw := range rawMessages {
		var check struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &check); err != nil {
			return err
		}
		var act Action
		switch check.Type {
		case actions.Dir.String():
			var dir Dir
			if err := json.Unmarshal(raw, &dir); err != nil {
				return err
			}
			act = &dir
		case actions.File.String():
			var file File
			if err := json.Unmarshal(raw, &file); err != nil {
				return err
			}
			act = &file
		case actions.Signature.String():
			var sig Signature
			if err := json.Unmarshal(raw, &sig); err != nil {
				return err
			}
			act = &sig
		case actions.SymLink.String():
			var sym SymLink
			if err := json.Unmarshal(raw, &sym); err != nil {
				return err
			}
			act = &sym
		default:
			// Handle unknown types or return an error
			return fmt.Errorf("unknown action type: %s", check.Type)
		}

		*a = append(*a, act)
	}

	return nil
}

func (a Actions) Len() int {
	return len(a)
}

func (a Actions) Less(i, j int) bool {
	return a[i].Key() < a[j].Key()
}

func (a Actions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
