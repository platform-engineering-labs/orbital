package actions

type Type string

const (
	Dir       Type = "Dir"
	File      Type = "File"
	SymLink   Type = "SymLink"
	Signature Type = "Signature"
)

func (t Type) String() string {
	return string(t)
}
