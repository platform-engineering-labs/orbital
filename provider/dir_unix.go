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
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"

	"github.com/naegelejd/go-acl/os/group"
	"github.com/platform-engineering-labs/orbital/action"
	pltfrm "github.com/platform-engineering-labs/orbital/platform"
)

type DirUnix struct {
	*slog.Logger
	dir *action.Dir

	phaseMap map[string]Call
}

func NewDirUnix(dir action.Action, phaseMap map[string]Call, log *slog.Logger) Provider {
	return &DirUnix{log, dir.(*action.Dir), phaseMap}
}

func (d *DirUnix) Realize(ctx context.Context) error {
	switch d.phaseMap[Phase(ctx)] {
	case Install:
		return d.install(ctx)
	case Package:
		d.Info(fmt.Sprintf("%s %s", d.dir.Type(), d.dir.Key()))
		return d.pkg(ctx)
	case Remove:
		return d.remove(ctx)
	default:
		return nil
	}
}

func (d *DirUnix) install(ctx context.Context) error {
	options := Opts(ctx)
	platform := Platform(ctx)
	target := path.Join(options.TargetPath, d.dir.Path)

	mode, err := strconv.ParseUint(d.dir.Mode, 0, 0)
	if err != nil {
		return err
	}

	// Allow chmod if exist for now ...
	err = os.Mkdir(target, os.FileMode(mode))
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Silent failures are fine, only a super user can chown to another user
	// Also a given user may not exist on a system though we should catch
	// that elsewhere
	if platform.OS == "all" && pltfrm.Current().OS == "darwin" && d.dir.Group == "root" {
		d.dir.Group = "admin"
	}

	owner, _ := user.Lookup(d.dir.Owner)
	grp, _ := user.LookupGroup(d.dir.Group)
	var uid int64
	var gid int64

	if owner != nil && grp != nil {
		uid, _ = strconv.ParseInt(owner.Uid, 0, 0)
		gid, _ = strconv.ParseInt(grp.Gid, 0, 0)
	}

	os.Chown(target, int(uid), int(gid))

	return nil
}

func (d *DirUnix) pkg(ctx context.Context) error {
	options := Opts(ctx)
	platform := Platform(ctx)
	target := path.Join(options.TargetPath, d.dir.Path)

	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if d.dir.Mode == "" {
		d.dir.Mode = fmt.Sprintf("%#o", info.Mode().Perm())
	}

	if d.dir.Owner == "" {
		if options.Secure {
			d.dir.Owner = "root"
		} else if options.Owner != "" {
			d.dir.Owner = options.Owner
		} else {
			usr, err := user.LookupId(fmt.Sprint(info.Sys().(*syscall.Stat_t).Uid))
			if err != nil {
				return err
			}
			d.dir.Owner = usr.Username
		}
	}

	if d.dir.Group == "" {
		if options.Secure {
			if platform.OS == "darwin" {
				d.dir.Group = "admin"
			} else {
				d.dir.Group = "root"
			}
		} else if options.Group != "" {
			d.dir.Group = options.Group
		} else {
			grp, err := group.LookupId(fmt.Sprint(info.Sys().(*syscall.Stat_t).Gid))
			if err != nil {
				return err
			}
			d.dir.Group = grp.Name
		}
	}

	return err
}

func (d *DirUnix) remove(ctx context.Context) error {
	options := Opts(ctx)
	target := path.Join(options.TargetPath, d.dir.Path)

	empty, err := d.isEmpty(target)
	if err != nil {
		return err
	}

	if empty == false {
		return nil
	}

	err = os.Remove(target)
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

func (d *DirUnix) isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
