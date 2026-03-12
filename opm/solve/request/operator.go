package request

import (
	"encoding"
	"fmt"
)

type Operator string

const (
	ANY Operator = "ANY"
	GTE Operator = "GTE"
	LTE Operator = "LTE"
	EQ  Operator = "EQ"
	EXQ Operator = "EXQ"
)

func (rcv Operator) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Operator)

func (rcv *Operator) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "ANY":
		*rcv = ANY
	case "GTE":
		*rcv = GTE
	case "LTE":
		*rcv = LTE
	case "EQ":
		*rcv = EQ
	case "EXQ":
		*rcv = EXQ
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid req.Operation`, str)
	}
	return nil
}
