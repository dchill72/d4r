package ui

import "github.com/charmbracelet/lipgloss"

// ThemeNames is the ordered list of available themes, matching huh's theme set.
var ThemeNames = []string{"charm", "dracula", "tokyo-night", "base16", "catppuccin"}

// ThemeDisplayNames maps theme IDs to human-readable labels.
var ThemeDisplayNames = map[string]string{
	"charm":       "Charm",
	"dracula":     "Dracula",
	"tokyo-night": "Tokyo Night",
	"base16":      "Base16",
	"catppuccin":  "Catppuccin",
}

type palette struct {
	primary    lipgloss.Color // main accent (tabs, titles, key hints)
	success    lipgloss.Color // running state
	warning    lipgloss.Color // paused / confirm prompt
	errCol     lipgloss.Color // exited / error state
	muted      lipgloss.Color // secondary text
	dimMuted   lipgloss.Color // dividers, whitespace bg
	text       lipgloss.Color // primary text
	bgSelected lipgloss.Color // selected row background
	border     lipgloss.Color // divider lines
	headerText lipgloss.Color // column header labels
}

// Color palettes modelled after the corresponding huh themes.
var palettes = map[string]palette{
	// huh.ThemeCharm — pink/purple
	"charm": {
		primary:    "#F780E2",
		success:    "#02BA84",
		warning:    "#ECFD65",
		errCol:     "#FF4672",
		muted:      "#a49fa5",
		dimMuted:   "#3D3346",
		text:       "#FFFFFF",
		bgSelected: "#2B2534",
		border:     "#585189",
		headerText: "#7571F9",
	},
	// huh.ThemeDracula
	"dracula": {
		primary:    "#BD93F9",
		success:    "#50FA7B",
		warning:    "#F1FA8C",
		errCol:     "#FF5555",
		muted:      "#6272A4",
		dimMuted:   "#44475A",
		text:       "#F8F8F2",
		bgSelected: "#44475A",
		border:     "#6272A4",
		headerText: "#8BE9FD",
	},
	// huh.ThemeTokyoNight
	"tokyo-night": {
		primary:    "#7AA2F7",
		success:    "#9ECE6A",
		warning:    "#E0AF68",
		errCol:     "#F7768E",
		muted:      "#565F89",
		dimMuted:   "#3B4261",
		text:       "#C0CAF5",
		bgSelected: "#283457",
		border:     "#3B4261",
		headerText: "#7DCFFF",
	},
	// huh.ThemeBase16 — terminal-native
	"base16": {
		primary:    "#6FB3D2",
		success:    "#4EBF71",
		warning:    "#E9B143",
		errCol:     "#E06C75",
		muted:      "#767676",
		dimMuted:   "#444444",
		text:       "#D0D0D0",
		bgSelected: "#1C1C1C",
		border:     "#444444",
		headerText: "#6FB3D2",
	},
	// huh.ThemeCatppuccin — Mocha variant
	"catppuccin": {
		primary:    "#CBA6F7",
		success:    "#A6E3A1",
		warning:    "#F9E2AF",
		errCol:     "#F38BA8",
		muted:      "#6C7086",
		dimMuted:   "#45475A",
		text:       "#CDD6F4",
		bgSelected: "#313244",
		border:     "#45475A",
		headerText: "#89DCEB",
	},
}

// ApplyTheme rebuilds all UI style variables to match the named theme.
// Calling this at any point causes subsequent renders to use the new theme.
func ApplyTheme(name string) {
	p, ok := palettes[name]
	if !ok {
		p = palettes["charm"]
	}
	currentDimMuted = p.dimMuted

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(p.primary)
	styleVersion = lipgloss.NewStyle().Foreground(p.muted)
	styleTabActive = lipgloss.NewStyle().Bold(true).Foreground(p.text).Background(p.primary).Padding(0, 2)
	styleTabInactive = lipgloss.NewStyle().Foreground(p.muted).Padding(0, 2)
	styleDivider = lipgloss.NewStyle().Foreground(p.border)

	styleColHeader = lipgloss.NewStyle().Bold(true).Foreground(p.headerText)
	styleRowSelected = lipgloss.NewStyle().Background(p.bgSelected).Foreground(p.text)
	styleRowNormal = lipgloss.NewStyle()

	styleStateRunning = lipgloss.NewStyle().Foreground(p.success).Bold(true)
	styleStateExited = lipgloss.NewStyle().Foreground(p.errCol)
	styleStatePaused = lipgloss.NewStyle().Foreground(p.warning)
	styleStateOther = lipgloss.NewStyle().Foreground(p.muted)

	styleFooterKey = lipgloss.NewStyle().Foreground(p.primary).Bold(true)
	styleFooterDesc = lipgloss.NewStyle().Foreground(p.muted)
	styleFooterSep = lipgloss.NewStyle().Foreground(p.dimMuted)
	styleStatusBar = lipgloss.NewStyle().Foreground(p.muted)
	styleConfirmPrompt = lipgloss.NewStyle().Foreground(p.warning).Bold(true)
	styleError = lipgloss.NewStyle().Foreground(p.errCol)

	// Theme picker overlay
	stylePickerBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.primary).
		Padding(1, 3)
	stylePickerTitle = lipgloss.NewStyle().Bold(true).Foreground(p.primary)
	stylePickerSelected = lipgloss.NewStyle().Bold(true).Foreground(p.primary)
	stylePickerOption = lipgloss.NewStyle().Foreground(p.muted)
	stylePickerHint = lipgloss.NewStyle().Foreground(p.dimMuted)
}

// ThemeIndex returns the list position of a theme name, defaulting to 0.
func ThemeIndex(name string) int {
	for i, n := range ThemeNames {
		if n == name {
			return i
		}
	}
	return 0
}

func init() {
	ApplyTheme("charm")
}
