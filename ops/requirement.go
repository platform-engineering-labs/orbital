package ops

import (
	"strings"

	"github.com/platform-engineering-labs/orbital/opm/solve/request"
)

type Requirement struct {
	Name     string           `pkl:"name" json:"name"`
	Method   request.Method   `pkl:"method" json:"method"`
	Operator request.Operator `pkl:"operator" json:"operator"`
	Version  *Version         `pkl:"version" json:"version"`
}

func NewRequirement(name string, version *Version) *Requirement {
	return &Requirement{Name: name, Version: version}
}

func NewRequirementFromSimpleString(id string) (*Requirement, error) {
	requirement := &Requirement{}
	requirement.Method = "depends"

	split := strings.Split(id, "@")

	if len(split) < 2 {
		requirement.Name = id
		return requirement.ANY(), nil
	}

	requirement.Name = split[0]

	version := &Version{}
	err := version.Parse(split[1])
	if err != nil {
		return nil, err
	}

	requirement.Version = version

	if requirement.Version.Timestamp.IsZero() {
		return requirement.EQ(), nil
	} else {
		return requirement.EXQ(), nil
	}
}

func (r *Requirement) Depends() *Requirement {
	r.Method = request.Depends
	return r
}

func (r *Requirement) Provides() *Requirement {
	r.Method = request.Provides
	return r
}

func (r *Requirement) Conflicts() *Requirement {
	r.Method = request.Conflicts
	return r
}

func (r *Requirement) ANY() *Requirement {
	r.Operator = request.ANY
	return r
}

func (r *Requirement) GTE() *Requirement {
	r.Operator = request.GTE
	return r
}

func (r *Requirement) LTE() *Requirement {
	r.Operator = request.LTE
	return r
}

func (r *Requirement) EQ() *Requirement {
	r.Operator = request.EQ
	return r
}

func (r *Requirement) EXQ() *Requirement {
	r.Operator = request.EXQ
	return r
}
