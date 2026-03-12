package metadata

import "github.com/platform-engineering-labs/orbital/action"

type Signature struct {
	Id string `storm:"id"`

	*action.Signature `storm:"inline"`
}
