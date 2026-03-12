package solve

import (
	"sort"

	"github.com/platform-engineering-labs/orbital/ops"
)

type PolicyMethod string

const (
	Updated   PolicyMethod = "updated"
	Installed PolicyMethod = "installed"
)

type Policy interface {
	PruneProvides(packages ops.Headers) ops.Headers
	SelectRequest(packages ops.Headers) *ops.Header
	SelectSolution(solutions Solutions) *Solution
}

type UpdatedPolicy struct{}
type InstalledPolicy struct{}

func NewPolicy(method PolicyMethod) Policy {
	switch method {
	case Updated:
		return &UpdatedPolicy{}
	case Installed:
		return &InstalledPolicy{}
	}

	return nil
}

func (u *UpdatedPolicy) PruneProvides(packages ops.Headers) ops.Headers {
	if len(packages) == 0 {
		return packages
	}

	sort.Sort(packages)

	if packages[0].Priority == -2 {
		return ops.Headers{packages[0]}
	}

	return packages
}

func (u *UpdatedPolicy) SelectRequest(packages ops.Headers) *ops.Header {
	sort.Sort(packages)

	for _, pkg := range packages {
		if len(packages) > 1 && pkg.Priority == -1 {
			if packages[1].Version.GT(packages[0].Version) {
				return packages[1]
			}
		}

		return pkg
	}

	return nil
}

func (u *UpdatedPolicy) SelectSolution(solutions Solutions) *Solution {

	solution := solutions[0]
	return &solution
}

func (i *InstalledPolicy) PruneProvides(packages ops.Headers) ops.Headers {
	return packages
}

func (i *InstalledPolicy) SelectRequest(packages ops.Headers) *ops.Header {
	sort.Sort(packages)

	for _, pkg := range packages {
		if pkg.Priority <= -1 {
			return pkg
		}
	}

	for _, pkg := range packages {
		return pkg
	}

	return nil
}

func (i *InstalledPolicy) SelectSolution(solutions Solutions) *Solution {
	return nil
}
