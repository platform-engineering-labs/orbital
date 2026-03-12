package request

import (
	"encoding"
	"fmt"
)

type Method string

const (
	Conflicts Method = "conflicts"
	Depends   Method = "depends"
	Provides  Method = "provides"
)

func (rcv Method) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Method)

func (rcv *Method) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "conflicts":
		*rcv = Conflicts
	case "depends":
		*rcv = Depends
	case "provides":
		*rcv = Provides
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid req.Method`, str)
	}
	return nil
}
