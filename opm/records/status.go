package records

import (
	"sort"

	"github.com/platform-engineering-labs/orbital/opm/candidate"
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
