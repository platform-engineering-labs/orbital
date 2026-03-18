/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

/*
 * Copyright 2018 Zachary Schneider
 */

package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"

	"github.com/naegelejd/go-acl/os/group"
	"github.com/platform-engineering-labs/orbital/action"
	opayload "github.com/platform-engineering-labs/orbital/opkg/payload"
	pltfrm "github.com/platform-engineering-labs/orbital/platform"
)

type FileUnix struct {
	*slog.Logger
	file *action.File

	phaseMap map[string]Call
}

func NewFileUnix(file action.Action, phaseMap map[string]Call, log *slog.Logger) Provider {
	return &FileUnix{log, file.(*action.File), phaseMap}
}

func (f *FileUnix) Realize(ctx context.Context) error {
	switch f.phaseMap[Phase(ctx)] {
	case Install:
		return f.install(ctx)
	case Package:
		f.Info(fmt.Sprintf("%s %s", f.file.Type(), f.file.Key()))
		return f.pkg(ctx)
	case Remove:
		return f.remove(ctx)
	case Validate:
		return f.validate(ctx)
	default:
		return nil
	}
}

func (f *FileUnix) install(ctx context.Context) error {
	options := Opts(ctx)
	platform := Platform(ctx)
	payload := ctx.Value("payload").(*opayload.Reader)

	target := path.Join(options.TargetPath, f.file.Path)

	mode, err := strconv.ParseUint(f.file.Mode, 0, 0)
	if err != nil {
		return err
	}

	if f.file.Size != 0 {
		var digest string
		var err error

		os.Remove(target)

		digest, err = payload.Get(target, int64(f.file.Offset), int64(f.file.Size))
		if err != nil {
			return err
		}

		if digest != f.file.Digest {
			return errors.New(fmt.Sprint("digest does not match manifest for: ", target))
		}
	} else {
		file, err := os.Create(target)
		if err != nil {
			return err
		}

		file.Close()
	}

	// Silent failures are fine, only a super user can chown to another user
	// Also a given user may not exist on a system though we should catch
	// that elsewhere
	if platform.OS == "all" && pltfrm.Current().OS == "darwin" && f.file.Group == "root" {
		f.file.Group = "admin"
	}

	owner, _ := user.Lookup(f.file.Owner)
	grp, _ := user.LookupGroup(f.file.Group)
	var uid int64
	var gid int64

	if owner != nil && grp != nil {
		uid, _ = strconv.ParseInt(owner.Uid, 0, 0)
		gid, _ = strconv.ParseInt(grp.Gid, 0, 0)
	}

	os.Chown(target, int(uid), int(gid))
	os.Chmod(target, os.FileMode(mode))

	return nil
}

func (f *FileUnix) pkg(ctx context.Context) error {
	options := Opts(ctx)
	platform := Platform(ctx)
	payload := ctx.Value("payload").(*opayload.Writer)

	target := path.Join(options.TargetPath, f.file.Path)

	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if f.file.Mode == "" {
		f.file.Mode = fmt.Sprintf("%#o", info.Mode().Perm())
	}

	if f.file.Owner == "" {
		if options.Secure {
			f.file.Owner = "root"
		} else if options.Owner != "" {
			f.file.Owner = options.Owner
		} else {
			usr, err := user.LookupId(fmt.Sprint(info.Sys().(*syscall.Stat_t).Uid))
			if err != nil {
				return err
			}
			f.file.Owner = usr.Username
		}
	}

	if f.file.Group == "" {
		if options.Secure {
			if platform.OS == "darwin" {
				f.file.Group = "admin"
			} else {
				f.file.Group = "root"
			}
		} else if options.Group != "" {
			f.file.Group = options.Group
		} else {
			grp, err := group.LookupId(fmt.Sprint(info.Sys().(*syscall.Stat_t).Gid))
			if err != nil {
				return err
			}
			f.file.Group = grp.Name
		}
	}

	f.file.Size = int(info.Size())

	// Add to payload
	if f.file.Size != 0 {
		f.file.Offset, f.file.Csize, f.file.Digest, err = payload.Put(target)
	} else {
		f.file.Offset = 0
		f.file.Csize = 0
		f.file.Digest = ""
	}

	return err
}

func (f *FileUnix) validate(ctx context.Context) error {
	if f.file.Size == 0 {
		return nil
	}

	options := Opts(ctx)
	payload := ctx.Value("payload").(*opayload.Reader)

	digest, err := payload.Verify(int64(f.file.Offset), int64(f.file.Size))
	if err != nil {
		return err
	}

	if digest != f.file.Digest {
		return errors.New(fmt.Sprint("digest does not match manifest for: ", f.file.Path))
	}

	if options.Verbose {
		f.Info(fmt.Sprintf("Verified %s %s", f.file.Type(), f.file.Key()))
	}

	return err
}

func (f *FileUnix) remove(ctx context.Context) error {
	options := Opts(ctx)
	target := path.Join(options.TargetPath, f.file.Path)

	err := os.Remove(target)
	if os.IsNotExist(err) {
		return nil
	}

	return err
}
