package shared

// NewPhase returns a Phase object with the given name and message
func NewPhase(name string, message string, reason string) *Phase {
	return &Phase{
		Name:    name,
		Message: message,
		Reason:  reason,
	}
}
