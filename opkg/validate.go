package opkg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital/action/actions"
	"github.com/platform-engineering-labs/orbital/opm/phase"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/provider"
)

type Validator struct {
	*slog.Logger
	sec   security.Security
	quiet bool
}

func NewValidator(logger *slog.Logger, sec security.Security, quiet bool) *Validator {
	return &Validator{logger, sec, quiet}
}

func (v *Validator) Validate(path string) error {
	reader := NewReader(path, "")

	err := reader.Read()
	if err != nil {
		return err
	}

	verified, err := v.sec.VerifyManifest(reader.Manifest)
	if err != nil {
		return err
	}

	if !v.quiet {
		for _, result := range verified {
			v.Info(fmt.Sprintf("verified signature by signer: %s %s", result.Cert.Subject, result.Signature.SKI))
		}
	}

	options := &provider.Options{}

	ctx := context.WithValue(context.Background(), "phase", phase.VALIDATE)
	ctx = context.WithValue(ctx, "payload", reader.Payload)
	ctx = context.WithValue(ctx, "options", options)
	
	contents := reader.Manifest.Select(actions.File)

	factory := provider.DefaultFactory(v.Logger)

	if !v.quiet {
		v.Info("validating payload ...")
	}

	for _, fsObject := range contents {
		err = factory.Get(fsObject).Realize(ctx)
		if err != nil {
			return err
		}
	}

	if !v.quiet {
		v.Info(fmt.Sprintf("payload verified: %s", reader.Manifest.Id()))
	}

	return nil
}
