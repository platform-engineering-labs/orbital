package config

import (
	"encoding"
	"fmt"
)

type Mode string

const (
	DynamicMode  Mode = "dynamic"
	EmbeddedMode Mode = "embedded"
	RootMode     Mode = "root"
)

func (rcv Mode) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Mode)

func (rcv *Mode) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "dynamic":
		*rcv = DynamicMode
	case "embedded":
		*rcv = EmbeddedMode
	case "root":
		*rcv = RootMode
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid config.Mode`, str)
	}
	return nil
}
