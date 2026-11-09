package check

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

type Result struct {
	CheckID, Subject, Message string
	Status                    Status
}
