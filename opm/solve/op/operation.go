package op

type Operation string

const (
	Install Operation = "install"
	Update  Operation = "update"
	Remove  Operation = "remove"
	Upgrade Operation = "upgrade"
)
