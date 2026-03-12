package ops

import (
	"fmt"

	"github.com/platform-engineering-labs/orbital/opm/solve/request"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
)

type Header struct {
	Name         string                       `pkl:"name" json:"name"`
	Version      *Version                     `pkl:"version" json:"version"`
	Publisher    string                       `pkl:"publisher" json:"publisher"`
	Originator   string                       `pkl:"originator" json:"originator,omitempty"`
	Arch         arch.Arch                    `pkl:"arch" json:"arch"`
	OS           os.OS                        `pkl:"os" json:"os"`
	Summary      string                       `pkl:"summary" json:"summary"`
	Description  string                       `pkl:"description" json:"description"`
	Requirements []*Requirement               `pkl:"requirements" json:"requirements"`
	Metadata     map[string]map[string]string `pkl:"metadata" json:"metadata"`

	Priority int
	Location int
}

func (h *Header) Id() *Id {
	return &Id{Name: h.Name, Version: h.Version}
}

func (h *Header) FileName() string {
	return fmt.Sprintf("%s@%s-%s-%s.opkg", h.Name, h.Version.String(), h.OS, h.Arch)
}

func (h *Header) Platform() *platform.Platform {
	return &platform.Platform{
		OS:   h.OS,
		Arch: h.Arch,
	}
}

func (h *Header) Satisfies(rq *Requirement) bool {
	switch rq.Operator {
	case request.ANY:
		return true
	case request.EXQ:
		return h.Version.EXQ(rq.Version)
	case request.GTE:
		return h.Version.GTE(rq.Version)
	case request.EQ:
		return h.Version.EQ(rq.Version)
	case request.LTE:
		return h.Version.LTE(rq.Version)
	}

	return false
}

type Headers []*Header

func (slice Headers) Len() int {
	return len(slice)
}

func (slice Headers) Less(i, j int) bool {
	if slice[i].Name < slice[j].Name {
		return true
	}
	if slice[i].Name > slice[j].Name {
		return false
	}

	if slice[i].Priority < slice[j].Priority {
		return true
	}
	if slice[i].Priority > slice[j].Priority {
		return false
	}

	return slice[i].Version.GT(slice[j].Version)
}

func (slice Headers) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
