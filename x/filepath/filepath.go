package filepath

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !errors.Is(err, os.ErrNotExist)
}

func MustAbs(path string) string {
	var abs string

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		path = strings.Replace(path, "~", home, 1)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	return abs
}
