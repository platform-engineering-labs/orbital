package opm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/platform-engineering-labs/orbital/action/actions"
	"github.com/platform-engineering-labs/orbital/opm/solve/solution"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/segmentio/ksuid"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm/cache"
	"github.com/platform-engineering-labs/orbital/opm/phase"
	"github.com/platform-engineering-labs/orbital/opm/solve"
	"github.com/platform-engineering-labs/orbital/opm/state"
	"github.com/platform-engineering-labs/orbital/provider"
)

type Transaction struct {
	*slog.Logger

	targetPath string
	cache      *cache.Cache
	state      *state.State

	solution *solve.Solution
	readers  map[string]*opkg.Reader

	id   ksuid.KSUID
	date time.Time
}

func NewTransaction(log *slog.Logger, targetPath string, cache *cache.Cache, state *state.State) *Transaction {
	return &Transaction{log, targetPath, cache, state, nil, nil, ksuid.New(), time.Now()}
}

func (t *Transaction) Realize(sol *solve.Solution) error {
	t.solution = sol
	t.readers = make(map[string]*opkg.Reader)

	err := t.loadReaders()
	if err != nil {
		return err
	}

	err = t.solutionConflicts()
	if err != nil {
		return err
	}

	err = t.treeConflicts()
	if err != nil {
		return err
	}

	operations, err := t.solution.Graph()
	if err != nil {
		return err
	}

	for _, operation := range operations {
		switch operation.Operation {
		case solution.Remove:
			t.Info(fmt.Sprint("removing ", operation.Package.Id().String()))
			err = t.remove(operation.Package)
			if err != nil {
				return err
			}
		case solution.Install:
			// check if another version is installed and remove
			lookup, err := t.state.Packages.Get(operation.Package.Name)
			if err != nil {
				return err
			}

			if lookup != nil {
				t.Info(fmt.Sprint("removing ", lookup.Id().String()))
				err = t.remove(operation.Package)
				if err != nil {
					return err
				}

				err = t.state.Transactions.Put(t.id.String(), lookup.Id().String(), "remove", &t.date)
				if err != nil {
					return err
				}
			}

			t.Info(fmt.Sprint("installing ", operation.Package.Id()))
			err = t.install(operation.Package)
			if err != nil {
				return err
			}
		}

		if operation.Operation != solution.Noop {
			err = t.state.Transactions.Put(t.id.String(), operation.Package.Id().String(), string(operation.Operation), &t.date)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Transaction) loadReaders() error {
	var err error

	// Read Manifests
	for _, operation := range t.solution.Operations() {
		if operation.Operation == solution.Install {
			reader := opkg.NewReader(t.cache.GetFile(operation.Package.FileName()), "")

			err = reader.Read()
			if err != nil {
				return err
			}

			t.readers[reader.Manifest.Name] = reader
		}
	}

	return err
}

func (t *Transaction) solutionConflicts() error {
	var err error
	var fsActions action.Actions
	lookup := make(map[action.Action]*ops.Manifest)

	for _, reader := range t.readers {
		manifest := reader.Manifest

		actionz := manifest.Contents()

		// build lookup index, TODO revisit this
		for _, act := range actionz {
			lookup[act] = manifest
		}

		fsActions = append(fsActions, actionz...)
	}

	sort.Sort(fsActions)
	for index, act := range fsActions {
		prev := index - 1
		if prev != -1 {
			if act.Key() == fsActions[prev].Key() && act.Type() != actions.Dir && fsActions[prev].Type() != actions.Dir {
				return errors.New(fmt.Sprint(
					"Package Conflicts:\n",
					lookup[fsActions[prev]].Name, " ", strings.ToUpper(fsActions[prev].Type().String()), " => ", fsActions[prev].Key(), "\n",
					lookup[act].Name, " ", strings.ToUpper(act.Type().String()), " => ", act.Key()))
			}
		}
	}

	return err
}

func (t *Transaction) treeConflicts() error {
	var err error

	for _, reader := range t.readers {
		manifest := reader.Manifest

		for _, actn := range manifest.Contents() {
			fsEntries, err := t.state.Objects.Get(actn.Key())

			if err != nil {
				return err
			}

			for _, entry := range fsEntries {
				if entry.Pkg != manifest.Name && entry.Type != actions.Dir && actn.Type() != actions.Dir {
					return errors.New(fmt.Sprint(
						entry.Type,
						" ",
						entry.Path,
						" from installed pkg ",
						entry.Pkg,
						" conflicts with candidate ",
						manifest.Name))
				}
			}
		}
	}

	return err
}

func (t *Transaction) install(pkg *ops.Header) error {
	reader := t.readers[pkg.Name]

	// Setup context
	ctx := context.WithValue(context.Background(), "options", &provider.Options{TargetPath: t.targetPath})
	ctx = context.WithValue(ctx, "phase", phase.INSTALL)
	ctx = context.WithValue(ctx, "payload", reader.Payload)

	// Provider Factory
	factory := provider.DefaultFactory(t.Logger)

	manifest := reader.Manifest

	var contents action.Actions
	contents = manifest.Contents()

	sort.Sort(contents)

	for _, fsObject := range contents {
		err := factory.Get(fsObject).Realize(ctx)
		if err != nil {
			return err
		}
	}

	// Add this to the package db
	err := t.state.Packages.Put(pkg.Name, reader.Manifest)
	if err != nil {
		return err
	}

	// Add all the fs object to the fs db
	for _, fsObject := range contents {
		err = t.state.Objects.Put(fsObject.Key(), pkg.Name, fsObject.Type())
		if err != nil {
			return err
		}
	}

	// TODO Add templates to the tpl db

	return err
}

func (t *Transaction) remove(pkg *ops.Header) error {
	manifest, err := t.state.Packages.Get(pkg.Name)
	if err != nil {
		return err
	}

	if manifest != nil {
		// Setup context
		ctx := context.WithValue(context.Background(), "options", &provider.Options{TargetPath: t.targetPath})
		ctx = context.WithValue(ctx, "phase", phase.REMOVE)

		// Provider Factory
		factory := provider.DefaultFactory(t.Logger)

		var contents action.Actions
		contents = manifest.Contents()

		// Reverse the actionlist
		sort.Sort(sort.Reverse(contents))

		for _, fsObject := range contents {
			err = factory.Get(fsObject).Realize(ctx)
			if err != nil {
				return err
			}
		}
		
		// Remove from the package db
		err = t.state.Packages.Del(pkg.Name)
		if err != nil {
			return err
		}

		// Remove fs objects from fs db
		err = t.state.Objects.Del(pkg.Name)
		if err != nil {
			return err
		}

		// Remove an existing frozen entry
		err = t.state.Frozen.Del(pkg.Id().String())
		if err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "not found") {
				return err
			}
		}

		// TODO Remove templates from tpl db
	}

	return nil
}
