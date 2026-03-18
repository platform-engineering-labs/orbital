package provider

import (
	"context"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/action/actions"
	"github.com/platform-engineering-labs/orbital/opm/phase"
	"github.com/platform-engineering-labs/orbital/platform"
)

type Factory struct {
	phaseMap    map[actions.Type]map[string]Call
	providerMap map[actions.Type]func(action.Action, map[string]Call, *slog.Logger) Provider
	log         *slog.Logger
}

func New(log *slog.Logger) *Factory {
	return &Factory{
		make(map[actions.Type]map[string]Call),
		make(map[actions.Type]func(action.Action, map[string]Call, *slog.Logger) Provider),
		log,
	}
}

// Need to add provider switching
// for now defaults will work on all OSs we care about
func (f *Factory) Get(ac action.Action) Provider {
	return f.providerMap[ac.Type()](ac, f.phaseMap[ac.Type()], f.log)
}

// Build phase map
func (f *Factory) On(action actions.Type, phase string, call Call) *Factory {
	if f.phaseMap[action] == nil {
		f.phaseMap[action] = make(map[string]Call)
	}
	f.phaseMap[action][phase] = call

	return f
}

// Register Provider
func (f *Factory) Register(provider actions.Type, newFunc func(action.Action, map[string]Call, *slog.Logger) Provider) *Factory {
	f.providerMap[provider] = newFunc

	return f
}

func Phase(ctx context.Context) string {
	return ctx.Value("phase").(string)
}

func Opts(ctx context.Context) *Options {
	return ctx.Value("options").(*Options)
}

func Platform(ctx context.Context) *platform.Platform {
	return ctx.Value("platform").(*platform.Platform)
}

func DefaultFactory(log *slog.Logger) *Factory {
	factory := New(log)

	factory.
		Register(actions.Dir, NewDirUnix).
		Register(actions.File, NewFileUnix).
		Register(actions.SymLink, NewSymLinkUnix)

	factory.
		On(actions.Dir, phase.INSTALL, Install).
		On(actions.Dir, phase.PACKAGE, Package).
		On(actions.Dir, phase.REMOVE, Remove).
		On(actions.File, phase.INSTALL, Install).
		On(actions.File, phase.PACKAGE, Package).
		On(actions.File, phase.REMOVE, Remove).
		On(actions.File, phase.VALIDATE, Validate).
		On(actions.SymLink, phase.INSTALL, Install).
		On(actions.SymLink, phase.PACKAGE, Package).
		On(actions.SymLink, phase.REMOVE, Remove)

	return factory
}
