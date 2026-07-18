package paths

import (
	"path/filepath"

	"github.com/platform-engineering-labs/orbital/schema/names"
	filepathx "github.com/platform-engineering-labs/orbital/x/filepath"
)

const (
	Data   = "~/.pel/ops"
	Config = "~/.config/ops"
)

func ConfigFileDefault() string {
	return filepath.Join(filepathx.MustAbs(Config), names.ConfigFile)
}

func ConfigDefault() string {
	return filepathx.MustAbs(Config)
}

func DataDefault() string {
	return filepathx.MustAbs(Data)
}

func TreeStore() string {
	return filepathx.MustAbs(filepath.Join(Data, names.TreesDb))
}

func TreeCache(path string) string {
	return filepath.Join(path, names.TreeDataDir, names.Cache)
}
func TreeLock(path string) string {
	return filepath.Join(path, names.TreeDataDir, ".lock")
}
func TreeSigning(path string) string {
	return filepath.Join(path, names.TreeDataDir, names.PkiSigning)
}
func TreeTrust(path string) string {
	return filepath.Join(path, names.TreeDataDir, names.PkiTrust)
}
func TreeState(path string) string {
	return filepath.Join(path, names.TreeDataDir, names.StateDb)
}
func TreeRootDefault() string {
	return filepath.Join(filepathx.MustAbs(Data), names.TreeRoot)
}
