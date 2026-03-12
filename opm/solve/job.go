package solve

import (
	"github.com/platform-engineering-labs/orbital/opm/solve/op"
	"github.com/platform-engineering-labs/orbital/ops"
)

type Job struct {
	op          op.Operation
	requirement *ops.Requirement
}

func NewJob(op op.Operation, requirement *ops.Requirement) *Job {
	return &Job{op, requirement}
}

func (j *Job) Op() op.Operation {
	return j.op
}

func (j *Job) Requirement() *ops.Requirement {
	return j.requirement
}
