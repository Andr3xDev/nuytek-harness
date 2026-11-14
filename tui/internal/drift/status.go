package drift

type State string

const (
	StateInstalled    State = "installed"
	StateDrift        State = "drift"
	StateNotInstalled State = "not_installed"
	StateUnknown      State = "unknown"
)

type Status struct {
	Name   string
	State  State
	Reason string
}
