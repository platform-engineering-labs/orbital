package solution

type Operation string

const (
	Install Operation = "install"
	Remove  Operation = "remove"
	Noop    Operation = "noop"
)
