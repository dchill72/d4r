package ui

import (
	"d4r/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.themePickerCursor < len(ThemeNames)-1 {
			m.themePickerCursor++
			ApplyTheme(ThemeNames[m.themePickerCursor])
		}
	case "k", "up":
		if m.themePickerCursor > 0 {
			m.themePickerCursor--
			ApplyTheme(ThemeNames[m.themePickerCursor])
		}
	case "enter":
		m.currentTheme = ThemeNames[m.themePickerCursor]
		m.themePickerActive = false
		return m, saveThemeCmd(m.currentTheme)
	case "esc", "q":
		ApplyTheme(m.currentTheme)
		m.themePickerActive = false
	}
	return m, nil
}

func saveThemeCmd(theme string) tea.Cmd {
	return func() tea.Msg {
		_ = config.Save(config.Config{Theme: theme})
		return nil
	}
}
