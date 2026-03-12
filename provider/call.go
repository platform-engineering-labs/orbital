package provider

type Call string

const (
	Install  Call = "install"
	Package  Call = "package"
	Remove   Call = "remove"
	Validate Call = "validate"
)
