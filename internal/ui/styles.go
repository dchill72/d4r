package ui

import "github.com/charmbracelet/lipgloss"

// All style variables are declared here and initialised by ApplyTheme in theme.go.
// Do not set initial values here — theme.go's init() call handles that.
var (
	styleTitle         lipgloss.Style
	styleVersion       lipgloss.Style
	styleTabActive     lipgloss.Style
	styleTabInactive   lipgloss.Style
	styleDivider       lipgloss.Style
	styleColHeader     lipgloss.Style
	styleRowSelected   lipgloss.Style
	styleRowNormal     lipgloss.Style
	styleStateRunning  lipgloss.Style
	styleStateExited   lipgloss.Style
	styleStatePaused   lipgloss.Style
	styleStateOther    lipgloss.Style
	styleFooterKey     lipgloss.Style
	styleFooterDesc    lipgloss.Style
	styleFooterSep     lipgloss.Style
	styleStatusBar     lipgloss.Style
	styleConfirmPrompt lipgloss.Style
	styleError         lipgloss.Style

	// Theme picker overlay
	stylePickerBorder   lipgloss.Style
	stylePickerTitle    lipgloss.Style
	stylePickerSelected lipgloss.Style
	stylePickerOption   lipgloss.Style
	stylePickerHint     lipgloss.Style

	// Current dim colour used for Place whitespace background
	currentDimMuted lipgloss.Color
)

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return styleStateRunning
	case "exited":
		return styleStateExited
	case "paused":
		return styleStatePaused
	default:
		return styleStateOther
	}
}
