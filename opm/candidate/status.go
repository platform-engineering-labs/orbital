package candidate

type Status string

const (
	Frozen    Status = "Frozen"
	Installed Status = "Installed"
	Available Status = "Available"
	NotFound  Status = "NotFound"
	None      Status = ""
)
