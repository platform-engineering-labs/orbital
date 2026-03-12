package ops

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/action/actions"
)

type Manifest struct {
	*Header

	Actions action.Actions `pkl:"actions" json:"actions"`
}

func (m *Manifest) Load(manifest []byte) error {
	err := json.Unmarshal(manifest, m)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manifest) Add(act action.Action) {
	if m.Exists(act) {
		m.Actions[m.Index(act)] = act
	} else {
		m.Actions = append(m.Actions, act)
	}
}

func (m *Manifest) Exists(act action.Action) bool {
	return slices.ContainsFunc(m.Actions, func(a action.Action) bool {
		return a.Id() == act.Id()
	})
}

func (m *Manifest) Contents() action.Actions {
	var fs action.Actions

	for i, act := range m.Actions {
		if act.Type() == actions.File || act.Type() == actions.Dir || act.Type() == actions.SymLink {
			fs = append(fs, m.Actions[i])
		}
	}
	sort.Sort(fs)

	return fs
}

func (m *Manifest) Select(atype actions.Type) action.Actions {
	var acts action.Actions

	for i, act := range m.Actions {
		if act.Type() == atype {
			acts = append(acts, m.Actions[i])
		}
	}

	sort.Sort(acts)
	return acts
}

func (m *Manifest) Signatures() []*action.Signature {
	var sigs []*action.Signature

	for i, act := range m.Actions {
		if act.Type() == actions.Signature {
			sigs = append(sigs, m.Actions[i].(*action.Signature))
		}
	}

	return sigs
}

func (m *Manifest) Index(act action.Action) int {
	return slices.IndexFunc(m.Actions, func(a action.Action) bool {
		return act.Id() == a.Id()
	})
}

func (m *Manifest) Validate() error {
	sort.Sort(m.Actions)
	for index, act := range m.Actions {
		prev := index - 1
		if prev != -1 {
			if act.Key() == m.Actions[prev].Key() {
				return errors.New(fmt.Sprint(
					"Action Conflicts:\n",
					strings.ToUpper(m.Actions[prev].Type().String()), " => ", m.Actions[prev].Key(), "\n",
					strings.ToUpper(act.Type().String()), " => ", act.Key()))
			}
		}
	}

	return nil
}

func (m *Manifest) ToJson() []byte {
	sort.Sort(m.Actions)

	out, _ := json.Marshal(m)

	return out
}

func (m *Manifest) ToSigningJson() []byte {
	sort.Sort(m.Actions)

	s := &Manifest{Header: m.Header}

	for _, act := range m.Actions {
		if act.Type() != actions.Signature {
			s.Actions = append(s.Actions, act)
		}
	}

	out, _ := json.Marshal(s)

	return out
}
