package os

import "os"

func LookupEnv(key ...string) bool {
	for _, k := range key {
		if _, exists := os.LookupEnv(k); !exists {
			return false
		}
	}

	return true
}
