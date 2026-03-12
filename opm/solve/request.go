package solve

import (
	"github.com/platform-engineering-labs/orbital/opm/solve/op"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Request struct {
	jobs []*Job
}

func NewRequest() *Request {
	request := &Request{}
	return request
}

func (r *Request) Jobs() []*Job {
	return r.jobs
}

func (r *Request) Install(requirement *ops.Requirement) {
	r.addJob(requirement, op.Install)
}

func (r *Request) Update(requirement *ops.Requirement) {
	r.addJob(requirement, op.Update)
}

func (r *Request) Remove(requirement *ops.Requirement) {
	r.addJob(requirement, op.Remove)
}

func (r *Request) Upgrade() {
	r.jobs = append(r.jobs, NewJob(op.Upgrade, nil))
}

func (r *Request) addJob(requirement *ops.Requirement, op op.Operation) {
	r.jobs = append(r.jobs, NewJob(op, requirement))
}
