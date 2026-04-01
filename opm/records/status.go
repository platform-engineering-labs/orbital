package records

import (
	"sort"

	"github.com/platform-engineering-labs/orbital/opm/candidate"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Status struct {
	Status    candidate.Status
	Available []*Package
}

func (s *Status) Sort() {
	sort.Slice(s.Available, func(i, j int) bool {
		return s.Available[i].Version.GT(s.Available[j].Version)
	})
}

func (s *Status) HasUpdate() (bool, *Package) {
	s.Sort()
	if s.Available[0].Priority != -1 && s.Available[0].Priority != -2 {
		return true, s.Available[0]
	} else {
		return false, nil
	}
}

func (s *Status) HasVersion(version *ops.Version) (bool, *Package) {
	s.Sort()

	var result *Package
	for _, pkg := range s.Available {
		if pkg.Version.EXQ(version) {
			result = pkg
			break
		}
		if pkg.Version.EQ(version) {
			result = pkg
			break
		}
	}

	return result != nil, result
}
