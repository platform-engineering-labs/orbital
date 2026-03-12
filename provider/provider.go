package provider

import "context"

type Provider interface {
	Realize(ctx context.Context) error
}

type Options struct {
	OutputPath string
	TargetPath string
	WorkPath   string
	CachePath  string

	Secure   bool
	Restrict bool

	Owner string
	Group string

	Debug   bool
	Verbose bool
}
