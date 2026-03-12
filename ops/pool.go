package ops

import (
	"errors"
	"sort"

	"github.com/platform-engineering-labs/orbital/opm/solve/request"
)

type Pool struct {
	index  map[string]Headers
	rindex map[string]Headers
	frozen map[string]bool

	Packages Headers

	repos Repos
}

func NewPool(tree *Repository, frozen map[string]bool, repos ...*Repository) (*Pool, error) {
	pool := &Pool{index: make(map[string]Headers), rindex: make(map[string]Headers), frozen: frozen}

	if pool.frozen == nil {
		pool.frozen = make(map[string]bool)
	}

	if tree == nil {
		return nil, errors.New("repo.Pool: tree must not be nil, can be empty repository")
	}

	// Force set this for now
	tree.Priority = -1
	pool.repos = append(pool.repos, tree)

	if len(repos) > 0 {
		for _, rp := range repos {
			pool.repos = append(pool.repos, rp)
		}

		// Sort by priority
		sort.Sort(pool.repos)
	}

	pool.populate()

	return pool, nil
}

func (p *Pool) Contains(pkg *Header) bool {
	if _, ok := p.index[pkg.Name]; ok {
		for _, candidate := range p.index[pkg.Name] {
			if candidate.Version.EXQ(pkg.Version) {
				return true
			}
		}
	}

	return false
}

func (p *Pool) Location(index int) *Repository {
	return p.repos[index]
}

func (p *Pool) Installed(req *Requirement) *Header {
	if _, ok := p.index[req.Name]; ok {
		for index, candidate := range p.index[req.Name] {
			if candidate.Satisfies(req) && candidate.Priority <= -1 {
				return p.index[req.Name][index]
			}
		}
	}

	return nil
}

func (p *Pool) Frozen(id string) bool {
	return p.frozen[id]
}

func (p *Pool) Tree() Headers {
	var tree Headers

	for index, pkg := range p.Packages {
		if pkg.Priority <= -1 {
			tree = append(tree, p.Packages[index])
		}
	}

	return tree
}

func (p *Pool) RepoCount() int {
	return len(p.repos)
}

func (p *Pool) WhatDepends(name string) Headers {

	if _, ok := p.rindex[name]; ok {
		return p.rindex[name]
	}

	return nil
}

func (p *Pool) WhatProvides(req *Requirement) Headers {
	var provides Headers

	if _, ok := p.index[req.Name]; ok {
		for _, candidate := range p.index[req.Name] {
			// Exact equality will never satisfy provides for frozen entries
			// this will insure exact install requests will fail
			if candidate.Satisfies(req) || p.frozen[candidate.Id().String()] {
				provides = append(provides, candidate)
			}
		}
	}

	return provides
}

func (p *Pool) populate() {
	for index, rp := range p.repos {
		if rp.Enabled == false {
			continue
		}

		for _, packages := range rp.Packages {
			for _, pkg := range packages {
				pkg.Priority = rp.Priority
				pkg.Location = index

				if p.frozen[pkg.Id().String()] && pkg.Priority == -1 {
					pkg.Priority = -2
				}

				p.Packages = append(p.Packages, pkg)
				p.index[pkg.Name] = append(p.index[pkg.Name], pkg)

				// install reverse index
				if rp.Priority == -1 {
					for _, rq := range pkg.Requirements {
						if rq.Method == request.Depends {
							p.rindex[rq.Name] = append(p.rindex[rq.Name], pkg)
						}
					}
				}

				// provides support
				for _, rq := range pkg.Requirements {
					if rq.Method == request.Provides {
						p.index[rq.Name] = append(p.index[rq.Name], pkg)
					}
				}
			}
		}
	}

	sort.Sort(p.Packages)
}
