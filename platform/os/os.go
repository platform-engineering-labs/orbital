package os

import (
	"encoding"
	"fmt"
)

type OS string

const (
	Linux  OS = "linux"
	Darwin OS = "darwin"
	All    OS = "all"
)

func (rcv OS) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(OS)

func (rcv *OS) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "linux":
		*rcv = Linux
	case "darwin":
		*rcv = Darwin
	case "all":
		*rcv = All
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid platform.OS`, str)
	}
	return nil
}
