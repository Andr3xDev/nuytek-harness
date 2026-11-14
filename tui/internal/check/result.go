package check

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
	// StatusSkipped means a prerequisite is absent or the check does not apply; Message says why.
	StatusSkipped Status = "skipped"
)

type Result struct {
	CheckID, Subject, Message string
	Status                    Status
	Suggestion                string // "" = no known fix
}
