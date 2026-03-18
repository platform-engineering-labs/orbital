package orbital

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/platform-engineering-labs/orbital/action"
	"github.com/platform-engineering-labs/orbital/config"
	"github.com/platform-engineering-labs/orbital/opkg"
	"github.com/platform-engineering-labs/orbital/opm"
	"github.com/platform-engineering-labs/orbital/opm/candidate"
	"github.com/platform-engineering-labs/orbital/opm/fetcher"
	"github.com/platform-engineering-labs/orbital/opm/phase"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/opm/publisher"
	"github.com/platform-engineering-labs/orbital/opm/records"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/opm/solve"
	"github.com/platform-engineering-labs/orbital/opm/state"
	"github.com/platform-engineering-labs/orbital/opm/tree"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/provider"
	"github.com/platform-engineering-labs/orbital/schema/paths"
)

type Orbital struct {
	*slog.Logger

	config *config.Config
	tree   tree.Tree

	Cache       *Cache
	Opkg        *Opkg
	Pki         *Pki
	Publish     *Publish
	Repo        *Repo
	Transaction *Transaction
	Tree        *Tree
}

type Cache struct {
	*slog.Logger
	orb *Orbital
}

type Opkg struct {
	*slog.Logger
	orb *Orbital
}

type Pki struct {
	*slog.Logger
	orb *Orbital
}
type Publish struct {
	*slog.Logger
	orb *Orbital
}

type Repo struct {
	*slog.Logger
	orb *Orbital
}

type Transaction struct {
	*slog.Logger
	orb *Orbital
}
type Tree struct {
	*slog.Logger
	orb *Orbital
}

func New(logger *slog.Logger, cfg *config.Config, tr tree.Tree) (*Orbital, error) {
	orb := &Orbital{
		Logger: logger,
		config: cfg,
		tree:   tr,
	}

	orb.Cache = &Cache{logger, orb}
	orb.Opkg = &Opkg{logger, orb}
	orb.Pki = &Pki{logger, orb}
	orb.Publish = &Publish{logger, orb}
	orb.Repo = &Repo{logger, orb}
	orb.Transaction = &Transaction{logger, orb}
	orb.Tree = &Tree{logger, orb}

	err := orb.Init()
	if err != nil {
		return nil, err
	}

	return orb, nil
}

func (o *Orbital) Init() error {
	if o.config.Mode == config.DynamicMode {
		_ = os.MkdirAll(paths.ConfigDefault(), 0750)
		_ = os.MkdirAll(paths.DataDefault(), 0750)

		err := tree.CreateDefault(o.config.TreeRoot)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Orbital) Contents(pkg string) (action.Actions, error) {
	err := o.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer o.tree.Unlock()

	manifest, err := o.tree.State().Packages.Get(pkg)
	if err != nil {
		return nil, err
	}

	if manifest == nil {
		return nil, errors.New(fmt.Sprintf("not installed: %s", pkg))
	}

	return manifest.Contents(), nil
}

func (o *Orbital) Freeze(packages ...string) error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		req, err := ops.NewRequirementFromSimpleString(pkg)
		if err != nil {
			return err
		}

		target := pool.Installed(req)
		if target == nil {
			return fmt.Errorf("freeze candidate: %s not installed", pkg)
		} else {
			err := o.tree.State().Frozen.Put(target.Id().String())
			if err != nil {
				return err
			}
			o.Logger.Info(fmt.Sprintf("froze: %s", target.Id()))
		}
	}

	return nil
}

func (o *Orbital) Info(pkg string) (*ops.Header, error) {
	err := o.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer o.tree.Unlock()

	manifest, err := o.tree.State().Packages.Get(pkg)
	if err != nil {
		return nil, err
	}

	if manifest == nil {
		return nil, errors.New(fmt.Sprintf("not installed: %s", pkg))
	}

	return manifest.Header, nil
}

// TODO add support for local pkg files
func (o *Orbital) Install(packages ...string) error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return err
	}

	if pool.RepoCount() <= 1 {
		return errors.New("no repo metadata found: run refresh")
	}

	request := solve.NewRequest()
	for _, name := range packages {
		req, err := ops.NewRequirementFromSimpleString(name)
		if err != nil {
			return err
		}

		if len(pool.WhatProvides(req)) == 0 {
			return errors.New(fmt.Sprint("no isntall candidates found for: ", name))
		}

		request.Install(req)
	}

	// TODO: configure policy
	solver := solve.NewSolver(pool, solve.NewPolicy(solve.Updated))

	solution, err := solver.Solve(request)
	if err != nil {
		return err
	}

	operations, err := solution.Graph()
	if err != nil {
		return err
	}

	for _, op := range operations {
		switch op.Operation {
		case phase.INSTALL:
			fe, err := fetcher.New(o.Logger, o.tree.Cache(), o.tree.Security(), pool.Location(op.Package.Location))
			if err != nil {
				return err
			}
			o.Logger.Info(fmt.Sprint("fetching: ", op.Package.Id()))

			err = fe.Fetch(op.Package)
			if err != nil {
				o.Error(fmt.Sprint("failed: ", op.Package.Id()))
				return err
			}

			o.Logger.Info(fmt.Sprint("fetched: ", op.Package.Id()))
		case phase.NOOP:
			o.Logger.Info(fmt.Sprint("using: ", op.Package.Id()))
		}
	}

	if solution.Noop() {
		return nil
	}

	tr := opm.NewTransaction(o.Logger, o.tree.Current().Path, o.tree.Cache(), o.tree.State())

	return tr.Realize(solution)
}

func (o *Orbital) List() (packages []*records.Package, err error) {
	err = o.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pool.Tree() {
		record := &records.Package{Header: pkg, Frozen: pool.Frozen(pkg.Id().String())}

		packages = append(packages, record)
	}

	if len(packages) == 0 {
		o.Warn("no packages installed")
		return nil, nil
	}

	return packages, nil
}

func (o *Orbital) Plan(action string, packages ...string) ([]*solve.Operation, error) {
	err := o.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer o.tree.Unlock()

	if action != phase.INSTALL && action != phase.REMOVE {
		return nil, errors.New(fmt.Sprintf("invalid action: %s (install/remove supported)", action))
	}

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return nil, err
	}

	if pool.RepoCount() <= 1 {
		return nil, errors.New("no repo metadata found: run refresh")
	}

	request := solve.NewRequest()
	for _, pkg := range packages {
		req, err := ops.NewRequirementFromSimpleString(pkg)
		if err != nil {
			return nil, err
		}

		if len(pool.WhatProvides(req)) == 0 {
			return nil, errors.New(fmt.Sprintf("no candidates found for: %s", pkg))
		}

		switch action {
		case phase.INSTALL:
			request.Install(req)
		case phase.REMOVE:
			request.Remove(req)
		}
	}

	// TODO: configure policy
	solver := solve.NewSolver(pool, solve.NewPolicy(solve.Updated))

	solution, err := solver.Solve(request)
	if err != nil {
		return nil, err
	}

	return solution.Graph()
}

func (o *Orbital) Refresh() error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	for _, r := range o.tree.Config().Repositories {
		if r.Enabled == false {
			o.Warn(fmt.Sprintf("repo disabled: %s", r.SafeUri()))
			continue
		}

		ftchr, err := fetcher.New(o.Logger, o.tree.Cache(), o.tree.Security(), &r)
		if err != nil {
			o.Warn(fmt.Sprintf("failed to refresh: %s error: %s", r.SafeUri(), err))
			continue
		}

		o.Logger.Info(fmt.Sprintf("refreshing: %s", r.SafeUri()))

		err = ftchr.Refresh()
		if err == nil {
			o.Logger.Info(fmt.Sprintf("refreshed: %s", r.SafeUri()))
		} else if strings.Contains(err.Error(), "no trusted certificates") {
			o.Error(fmt.Sprintf("metadata validation failed: %s", r.SafeUri()))
		} else if strings.Contains(err.Error(), "refresh failed") {
			o.Error(fmt.Sprintf("refresh failed: %s", r.SafeUri()))
		} else {
			o.Warn(fmt.Sprintf("no metadata: %s", r.SafeUri()))
		}
	}

	return nil
}

func (o *Orbital) Ready() bool {
	return o.tree.Ready()
}

func (o *Orbital) Remove(packages ...string) error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return err
	}

	request := solve.NewRequest()
	for _, name := range packages {
		req, err := ops.NewRequirementFromSimpleString(name)
		if err != nil {
			return err
		}

		if pool.Installed(req) == nil {
			return errors.New(fmt.Sprint("no removal candidates found for: ", name))
		}

		request.Remove(req)
	}

	// TODO: configure policy
	solver := solve.NewSolver(pool, solve.NewPolicy(solve.Updated))

	solution, err := solver.Solve(request)
	if err != nil {
		return err
	}

	tr := opm.NewTransaction(o.Logger, o.tree.Current().Path, o.tree.Cache(), o.tree.State())

	return tr.Realize(solution)
}

func (o *Orbital) Status(pkg string) (*records.Status, error) {
	err := o.tree.Lock()
	if err != nil {
		return &records.Status{Status: candidate.None}, err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return &records.Status{Status: candidate.None}, err
	}

	req, err := ops.NewRequirementFromSimpleString(pkg)
	if err != nil {
		return &records.Status{Status: candidate.None}, err
	}

	result := &records.Status{
		Status: candidate.Available,
	}

	for _, pkg := range pool.WhatProvides(req) {
		record := &records.Package{Header: pkg, Frozen: pool.Frozen(pkg.Id().String()), Installed: pkg.Priority == -1}
		record.Locations = append(record.Locations, pkg.Location)

		if record.Frozen {
			result.Status = candidate.Frozen
		}
		if record.Installed {
			result.Status = candidate.Installed
		}

		if found := slices.IndexFunc(result.Available, func(r *records.Package) bool {
			return pkg.Version.EXQ(r.Version)
		}); found != -1 {
			result.Available[found].Locations = append(result.Available[found].Locations, pkg.Location)
		} else {
			result.Available = append(result.Available, record)
		}

	}

	if len(result.Available) == 0 {
		result.Status = candidate.NotFound
	}

	result.Sort()

	return result, nil
}

func (o *Orbital) Thaw(packages ...string) error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		req, err := ops.NewRequirementFromSimpleString(pkg)
		if err != nil {
			return err
		}

		target := pool.Installed(req)
		if target == nil {
			return fmt.Errorf("thaw candidate: %s not installed", pkg)
		} else {
			err := o.tree.State().Frozen.Del(target.Id().String())
			if err != nil {
				return err
			}
			o.Logger.Info(fmt.Sprintf("thawed: %s", target.Id()))
		}
	}

	return nil
}

func (o *Orbital) Update(packages ...string) error {
	err := o.tree.Lock()
	if err != nil {
		return err
	}
	defer o.tree.Unlock()

	pool, err := o.tree.Pool(platform.Expanded(o.tree.Config().Platform()), false)
	if err != nil {
		return err
	}

	if pool.RepoCount() <= 1 {
		return errors.New("no repo metadata found: run refresh")
	}
	if len(pool.Tree()) == 0 {
		o.Warn("no packages installed to update")
		return nil
	}

	// Update everything if no requested packages
	if len(packages) == 0 {
		for _, pkg := range pool.Tree() {
			packages = append(packages, pkg.Name)
		}
	}

	request := solve.NewRequest()
	for _, name := range packages {
		req, err := ops.NewRequirementFromSimpleString(name)
		if err != nil {
			return err
		}

		if len(pool.WhatProvides(req)) == 0 {
			return errors.New(fmt.Sprint("no update candidates found for: ", name))
		}

		request.Install(req)
	}

	solver := solve.NewSolver(pool, solve.NewPolicy(solve.Updated))

	solution, err := solver.Solve(request)
	if err != nil {
		return err
	}

	operations, err := solution.Graph()
	if err != nil {
		return err
	}

	for _, op := range operations {
		switch op.Operation {
		case phase.INSTALL:
			fe, err := fetcher.New(o.Logger, o.tree.Cache(), o.tree.Security(), pool.Location(op.Package.Location))
			if err != nil {
				return err
			}
			o.Logger.Info(fmt.Sprint("fetching: ", op.Package.Id()))

			err = fe.Fetch(op.Package)
			if err != nil {
				o.Error(fmt.Sprint("failed: ", op.Package.Id()))
				return err
			}

			o.Logger.Info(fmt.Sprint("fetched: ", op.Package.Id()))
		case phase.NOOP:
			o.Logger.Info(fmt.Sprint("using: ", op.Package.Id()))
		}
	}

	if solution.Noop() {
		return nil
	}

	tr := opm.NewTransaction(o.Logger, o.tree.Current().Path, o.tree.Cache(), o.tree.State())

	return tr.Realize(solution)
}

func (o *Orbital) getContext(phase string, options *provider.Options) context.Context {
	ctx := context.WithValue(context.Background(), "phase", phase)
	ctx = context.WithValue(ctx, "options", options)

	return ctx
}

func Dynamic(logger *slog.Logger, cfgPath string) (*Orbital, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	cfg.Mode = config.DynamicMode

	tr, err := tree.New(logger, cfg.TreeRoot, tree.Dynamic, nil)
	if err != nil {
		return nil, err
	}

	orb, err := New(logger, cfg, tr)
	if err != nil {
		return nil, fmt.Errorf("error: %s", err)
	}

	return orb, nil
}

func Embedded(logger *slog.Logger, treePath string, treeConfig *tree.Config) (*Orbital, error) {
	cfg := &config.Config{
		Mode:     config.EmbeddedMode,
		TreeRoot: treePath,
	}

	tr, err := tree.New(logger, cfg.TreeRoot, tree.Embedded, treeConfig)
	if err != nil {
		return nil, err
	}

	orb, err := New(logger, cfg, tr)
	if err != nil {
		return nil, fmt.Errorf("error: %s", err)
	}

	return orb, nil
}

func (c *Cache) Clean() error {
	return c.orb.tree.Cache().Clean()
}

func (c *Cache) Clear() error {
	return c.orb.tree.Cache().Clear()
}

func (o *Opkg) Build(manifestPath string, pltfrm *platform.Platform, targetPath string, workPath string, outputPath string, restrict bool, secure bool) (*ops.Manifest, string, error) {
	builder := opkg.NewBuilder(o.Logger).
		Platform(pltfrm).
		TargetPath(targetPath).
		WorkPath(workPath).
		OutputPath(outputPath).
		Restrict(restrict).
		Secure(secure)

	manifest, pkgPath, err := builder.Build(manifestPath)
	if err != nil {
		return nil, "", err
	}

	kp, err := o.orb.tree.Security().KeyPair(manifest.Publisher)
	if err != nil {
		return nil, "", err
	}

	if kp == nil {
		o.Error(fmt.Sprintf("no keypair found for publisher or distributor %s, not signing", manifest.Publisher))
		return manifest, pkgPath, nil
	}

	signer := opkg.NewSigner(pkgPath, workPath)

	rsaKey, err := kp.RSAKey()
	if err != nil {
		return nil, "", err
	}

	err = signer.Sign(kp.SKI, kp.Fingerprint, rsaKey)
	if err == nil {
		o.Info(fmt.Sprintf("Signed with keypair: %s", kp.Subject))
	}

	return manifest, pkgPath, nil
}

func (o *Opkg) Extract(opkgPath, targetPath string) error {
	reader := opkg.NewReader(opkgPath, "")

	err := reader.Read()
	if err != nil {
		return err
	}

	contents := reader.Manifest.Contents()

	options := &provider.Options{TargetPath: targetPath}
	ctx := o.orb.getContext(phase.INSTALL, options)
	ctx = context.WithValue(ctx, "platform", reader.Manifest.Platform())
	ctx = context.WithValue(ctx, "payload", reader.Payload)

	factory := provider.DefaultFactory(o.Logger)

	for _, entry := range contents {
		o.Info(fmt.Sprintf("Extracted => %s %s", strings.ToUpper(entry.Type().String()), path.Join(targetPath, entry.Key())))

		err = factory.Get(entry).Realize(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Opkg) Manifest(opkgPath string) (*ops.Manifest, error) {
	reader := opkg.NewReader(opkgPath, "")

	err := reader.Read()
	if err != nil {
		return nil, err
	}

	return reader.Manifest, nil
}

func (o *Opkg) Validate(opkgPath string) error {
	validator := opkg.NewValidator(o.Logger, o.orb.tree.Security(), false)

	return validator.Validate(opkgPath)
}

func (p *Pki) KeyPairImport(mode string, cert, key string) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	var certPem, keyPem []byte

	switch mode {
	case "env":
		certEnv, exists := os.LookupEnv(cert)
		if !exists {
			return fmt.Errorf("missing certificate: %s", cert)
		}
		certPem = []byte(certEnv)

		keyEnv, exists := os.LookupEnv(cert)
		if !exists {
			return fmt.Errorf("missing key: %s", key)
		}
		keyPem = []byte(keyEnv)
	case "file":
		certPem, err = os.ReadFile(cert)
		if err != nil {
			return err
		}

		keyPem, err = os.ReadFile(key)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid import mode: %s", mode)
	}

	err = security.ValidateKeyPair(certPem, keyPem)
	if err != nil {
		return err
	}

	metadata, err := security.CertMetadataFromBytes(&certPem)
	if err != nil {
		return err
	}

	kps, err := p.orb.tree.Pki().KeyPairs.GetByPublisher(metadata.Publisher)
	if err != nil {
		return err
	}

	if len(kps) > 0 {
		for index := range kps {
			p.Warn(
				fmt.Sprintf("removing: %s due to matching publisher: %s",
					kps[index].Fingerprint,
					metadata.Publisher,
				),
			)

			err := p.orb.tree.Pki().KeyPairs.Del(kps[index].SKI)
			if err != nil {
				return err
			}
		}
	}

	err = p.orb.tree.Pki().KeyPairs.Put(metadata.SKI, metadata.Fingerprint, metadata.Subject, metadata.Publisher, certPem, keyPem)
	if err != nil {
		return err
	}

	p.Info(fmt.Sprintf("imported keypair for publisher %s", metadata.Publisher))

	return nil
}

func (p *Pki) KeyPairList() ([]*pki.KeyPairEntry, error) {
	err := p.orb.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer p.orb.tree.Unlock()

	return p.orb.tree.Pki().KeyPairs.All()
}

func (p *Pki) KeyPairRemove(ski string) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	err = p.orb.tree.Pki().KeyPairs.Del(ski)
	if err != nil {
		return err
	}

	p.Info(fmt.Sprintf("removed keypair: %s", ski))

	return nil
}

func (p *Pki) TrustImportDNS(ski, publisher string) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	_, err = p.orb.tree.Security().Resolve(ski, publisher)

	return nil
}

func (p *Pki) TrustImportFiles(cert ...string) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	for _, crt := range cert {
		content, err := os.ReadFile(crt)
		if err != nil {
			return err
		}

		entry, err := p.orb.tree.Security().Trust(&content)
		if err != nil {
			return err
		}

		p.Info(fmt.Sprintf("trusted cert: %s -> %s", entry.Subject, entry.SKI))
	}

	return nil
}

func (p *Pki) TrustList() ([]*pki.CertEntry, error) {
	err := p.orb.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer p.orb.tree.Unlock()

	return p.orb.tree.Pki().Certificates.All()
}

func (p *Pki) TrustRefresh() error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	return p.orb.tree.Security().Refresh()
}

func (p *Pki) TrustRemove(ski string) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	err = p.orb.tree.Pki().Certificates.Del(ski)
	if err != nil {
		return err
	}

	p.Info(fmt.Sprintf("removed trusted certificate: %s", ski))

	return nil
}

func (p *Publish) Channel(repo string, channels []string, id *ops.Id) error {
	rp, err := p.orb.tree.Config().Repository(repo)
	if err != nil {
		return err
	}

	pub, err := publisher.New(p.Logger, &provider.Options{}, p.orb.tree.Security(), rp)
	if err != nil {
		return err
	}

	return pub.Channel(id, channels)
}

func (p *Publish) Fetch(names []string, pltfrm *platform.Platform) error {
	err := p.orb.tree.Lock()
	if err != nil {
		return err
	}
	defer p.orb.tree.Unlock()

	pool, err := p.orb.tree.Pool(platform.Expanded(pltfrm), true)
	if err != nil {
		return err
	}

	if pool.RepoCount() <= 1 {
		return errors.New("no repo metadata found: run ops refresh")
	}

	request := solve.NewRequest()
	for _, arg := range names {
		req, err := ops.NewRequirementFromSimpleString(arg)
		if err != nil {
			return err
		}

		if len(pool.WhatProvides(req)) == 0 {
			return errors.New(fmt.Sprintf("no candidates found for: %s", arg))
		}

		request.Install(req)
	}

	// TODO: configure policy
	policy := solve.NewPolicy(solve.Updated)

	for _, job := range request.Jobs() {
		pkg := policy.SelectRequest(pool.WhatProvides(job.Requirement()))

		fe, err := fetcher.New(p.orb.Logger, p.orb.tree.Cache(), p.orb.tree.Security(), pool.Location(pkg.Location))
		if err != nil {
			return err
		}

		err = fe.Fetch(pkg)
		if err != nil {
			return err
		}

		p.Info(fmt.Sprint("fetching: ", pkg.Id()))

		// Copy from cache to working directory
		wd, err := os.Getwd()
		if err != nil {
			return errors.New("could not get current directory")
		}

		src, err := os.Open(p.orb.tree.Cache().GetFile(pkg.FileName()))
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.OpenFile(filepath.Join(wd, pkg.FileName()), os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	return nil
}

func (p *Publish) Publish(repo string, workPath string, opkgFiles []string, channels []string) (published, pruned []string, err error) {
	rp, err := p.orb.tree.Config().Repository(repo)
	if err != nil {
		return nil, nil, err
	}

	if len(channels) > 0 && rp.Uri.Fragment != "" {
		channels = append(channels, rp.Uri.Fragment)
	}

	pub, err := publisher.New(p.Logger, &provider.Options{WorkPath: workPath}, p.orb.tree.Security(), rp)
	if err != nil {
		return nil, nil, err
	}

	return pub.Publish(opkgFiles, channels)
}

func (p *Publish) Yank(repo string, pkg string, workPath string) error {
	rp, err := p.orb.tree.Config().Repository(repo)
	if err != nil {
		return err
	}

	pub, err := publisher.New(p.Logger, &provider.Options{WorkPath: workPath}, p.orb.tree.Security(), rp)
	if err != nil {
		return err
	}

	return pub.Yank(pkg)
}

func (r *Repo) Contents(repoName string, all bool) (*ops.Repository, error) {
	err := r.orb.tree.Lock()
	if err != nil {
		return nil, err
	}
	defer r.orb.tree.Unlock()

	platforms := platform.SupportedPlatforms

	if !all {
		platforms = platform.Expanded(r.orb.tree.Config().Platform())
	}

	for _, repo := range r.orb.tree.Config().Repositories {
		if repoName == *repo.Name() {
			err := r.orb.tree.RepoLoad(platforms, &repo, all)
			if err != nil {
				return nil, err
			}

			return &repo, nil
		}
	}

	return nil, nil
}

func (r *Repo) Init(repo, workPath string) (uri *url.URL, err error) {
	rp, err := r.orb.tree.Config().Repository(repo)
	if err != nil {
		return nil, err
	}

	pub, err := publisher.New(r.Logger, &provider.Options{WorkPath: workPath}, r.orb.tree.Security(), rp)
	if err != nil {
		return nil, err
	}

	return &rp.Uri, pub.Init()
}

func (r *Repo) List() []ops.Repository {
	return r.orb.tree.Config().Repositories
}

func (t *Transaction) List() (map[string][]*state.TransactionEntry, error) {
	transactions, err := t.orb.tree.State().Transactions.All()
	if err != nil {
		return nil, err
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Id > transactions[j].Id
	})

	txmap := make(map[string][]*state.TransactionEntry)
	for _, tx := range transactions {
		txmap[tx.Id] = append(txmap[tx.Id], tx)
	}

	return txmap, nil
}

func (t *Tree) Destroy(name string) (*tree.Entry, error) {
	return t.orb.tree.Destroy(name)
}

func (t *Tree) Current() *tree.Entry {
	return t.orb.tree.Current()
}

func (t *Tree) Get(name string) (*tree.Entry, error) {
	return t.orb.tree.Get(name)
}

func (t *Tree) Init(name string, pltfrm *platform.Platform, force bool) (*tree.Entry, error) {
	return t.orb.tree.Init(name, pltfrm, force)
}

func (t *Tree) Pool(platforms []*platform.Platform, empty bool) (*ops.Pool, error) {
	return t.orb.tree.Pool(platforms, empty)
}

func (t *Tree) List() ([]*tree.Entry, error) {
	return t.orb.tree.List()
}

func (t *Tree) Switch(name string) error {
	return t.orb.tree.Switch(name)
}
