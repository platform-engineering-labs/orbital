package metadata

import "github.com/platform-engineering-labs/orbital/ops"

type Entry struct {
	Id string `storm:"id"`

	*ops.Header `storm:"inline"`
}
