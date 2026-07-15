package tui

// FocusTarget represents which input field should have focus
type FocusTarget string

const (
	// FocusTargetFooter indicates the footer input should have focus
	FocusTargetFooter FocusTarget = "footer"

	// FocusTargetMessage indicates the planning tab message input should have focus
	FocusTargetMessage FocusTarget = "message"
)

// String returns the string representation of the focus target
func (ft FocusTarget) String() string {
	return string(ft)
}

// IsValid checks if the focus target is a valid value
func (ft FocusTarget) IsValid() bool {
	switch ft {
	case FocusTargetFooter, FocusTargetMessage:
		return true
	default:
		return false
	}
}
