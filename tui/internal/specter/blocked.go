package specter

// ErrBlocked is returned by planners when a precondition fails; Reason is shown as "Cannot run: <reason>".
type ErrBlocked struct{ Reason string }

func (e ErrBlocked) Error() string { return e.Reason }
