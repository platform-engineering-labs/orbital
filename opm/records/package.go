package records

import "github.com/platform-engineering-labs/orbital/ops"

type Package struct {
	*ops.Header
	Frozen    bool
	Installed bool
}
