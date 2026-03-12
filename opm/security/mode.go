package security

import (
	"encoding"
	"fmt"
)

type Mode string

const (
	Empty Mode = "empty"
)

func (rcv Mode) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Mode)

func (rcv *Mode) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "default":
		*rcv = Default
	case "":
		*rcv = Empty
	case "none":
		*rcv = None
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid security.Mode`, str)
	}
	return nil
}
