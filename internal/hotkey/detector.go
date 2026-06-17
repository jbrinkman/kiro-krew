package hotkey

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

// HotkeyTriggeredMsg represents a hotkey event for Bubble Tea
type HotkeyTriggeredMsg struct{}

// HotkeyErrorMsg represents a hotkey error event
type HotkeyErrorMsg struct {
	Err error
}

// IsKiroKrewContext validates that the current process is running in a kiro-krew terminal
func IsKiroKrewContext() bool {
	return os.Getenv("KIRO_KREW_WATCHER_PID") != ""
}

// IsCtrlOptionP checks if the key sequence matches Ctrl+Option+P
func IsCtrlOptionP(msg tea.KeyPressMsg) bool {
	return msg.String() == "ctrl+alt+p"
}

// HandleKeyMsg processes key messages and returns hotkey events when appropriate
func HandleKeyMsg(msg tea.KeyPressMsg) tea.Cmd {
	if IsCtrlOptionP(msg) {
		if !IsKiroKrewContext() {
			return func() tea.Msg {
				return HotkeyErrorMsg{
					Err: fmt.Errorf("hotkey toggle not available outside kiro-krew context"),
				}
			}
		}

		return func() tea.Msg {
			return HotkeyTriggeredMsg{}
		}
	}
	return nil
}
