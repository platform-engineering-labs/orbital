package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"
)

type Cache struct {
	path   string
	hasher hash.Hash
}

func New(path string) *Cache {
	_ = os.MkdirAll(path, 0750)
	return &Cache{path, sha256.New()}
}

func (c *Cache) Exists(name string) bool {
	if _, err := os.Stat(c.GetFile(name)); os.IsNotExist(err) {
		return false
	}

	return true
}

func (c *Cache) HasMeta(uri string) bool {
	files, err := filepath.Glob(filepath.Join(c.path, c.getId(uri)+"*.db"))
	if err != nil {
		return false
	}

	if len(files) > 0 {
		return true
	}

	return false
}

func (c *Cache) GetMeta(platform string, uri string) string {
	return filepath.Join(c.path, fmt.Sprint(c.getId(uri), "-", platform, ".metadata.db"))
}

func (c *Cache) GetFile(name string) string {
	return filepath.Join(c.path, name)
}

func (c *Cache) Clean() error {
	pkgs, _ := filepath.Glob(filepath.Join(c.path, "*.opkg"))

	for _, f := range pkgs {
		os.Remove(f)
	}

	return nil
}

func (c *Cache) Clear() error {
	err := c.Clean()
	if err != nil {
		return err
	}

	metafiles, _ := filepath.Glob(filepath.Join(c.path, "*.db"))

	for _, f := range metafiles {
		os.Remove(f)
	}

	return nil
}

func (c *Cache) getId(id string) string {
	c.hasher.Reset()
	c.hasher.Write([]byte(id))
	return hex.EncodeToString(c.hasher.Sum(nil))
}
