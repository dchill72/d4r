package ui

// footerHints returns context-appropriate key hints for the footer.
// Each entry is a [key, description] pair.
func (m Model) footerHints() [][2]string {
	if m.confirm.active {
		return [][2]string{
			{"y", "confirm"},
			{"n/esc", "cancel"},
		}
	}

	switch m.screen {
	case screenDetail:
		return [][2]string{
			{"↑↓/j/k", "scroll"},
			{"pgup/pgdn", "page"},
			{"esc/q", "back"},
		}
	case screenLogs:
		follow := "follow: off"
		if m.following {
			follow = "follow: on"
		}
		return [][2]string{
			{"↑↓/j/k", "scroll"},
			{"pgup/pgdn", "page"},
			{"f", follow},
			{"esc/q", "back"},
		}
	}

	// List screen — tab-specific
	base := [][2]string{
		{"tab/1-4", "switch"},
		{"j/k/↑↓", "navigate"},
		{"r", "refresh"},
		{"t", "theme"},
		{"q", "quit"},
	}

	switch m.tab {
	case tabContainers:
		ct := m.selectedContainer()
		hints := [][2]string{
			{"tab/1-4", "switch"},
			{"j/k/↑↓", "navigate"},
			{"a", func() string {
			if m.showAll {
				return "running only"
			}
			return "show all"
		}()},
			{"r", "refresh"},
		}
		if ct != nil {
			hints = append(hints, [2]string{"enter/d", "details"})
			hints = append(hints, [2]string{"l", "logs"})
			if ct.State == "running" {
				hints = append(hints, [2]string{"s", "shell"})
				hints = append(hints, [2]string{"x", "stop"})
			} else {
				hints = append(hints, [2]string{"u", "start"})
			}
			hints = append(hints, [2]string{"D", "delete"})
		}
		hints = append(hints, [2]string{"t", "theme"})
		hints = append(hints, [2]string{"q", "quit"})
		return hints
	case tabVolumes, tabNetworks, tabImages:
		hasItem := m.selectedIndex() >= 0 && m.selectedIndex() < m.listLen()
		hints := base
		if hasItem {
			hints = [][2]string{
				{"tab/1-4", "switch"},
				{"j/k/↑↓", "navigate"},
				{"r", "refresh"},
				{"enter/d", "details"},
				{"D", "delete"},
				{"t", "theme"},
				{"q", "quit"},
			}
		}
		return hints
	}

	return base
}
