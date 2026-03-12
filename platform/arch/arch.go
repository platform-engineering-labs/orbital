package arch

import (
	"encoding"
	"fmt"
)

type Arch string

const (
	X8664 Arch = "x8664"
	Arm64 Arch = "arm64"
	All   Arch = "all"
)

func (rcv Arch) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Arch)

func (rcv *Arch) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "x8664":
		*rcv = X8664
	case "arm64":
		*rcv = Arm64
	case "all":
		*rcv = All
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid platform.Arch`, str)
	}
	return nil
}
