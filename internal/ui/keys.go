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

	// Shared fragments reused across all list-screen branches.
	nav := [][2]string{
		{"tab/1-4", "switch"},
		{"j/k/↑↓", "navigate"},
		{"F5", "refresh"},
	}
	tail := [][2]string{
		{"t", "theme"},
		{"q", "quit"},
	}

	switch m.tab {
	case tabContainers:
		aLabel := "show all"
		if m.showAll {
			aLabel = "running only"
		}
		hints := append(nav, [2]string{"a", aLabel})
		if ct := m.selectedContainer(); ct != nil {
			hints = append(hints, [2]string{"enter/d", "details"}, [2]string{"l", "logs"})
			if ct.State == "running" {
				hints = append(hints, [2]string{"s", "shell"}, [2]string{"x", "stop"})
			} else {
				hints = append(hints, [2]string{"u", "start"})
			}
			hints = append(hints, [2]string{"D", "delete"})
		}
		return append(hints, tail...)

	case tabVolumes:
		hints := nav
		if m.selectedIndex() < m.listLen() {
			hints = append(hints,
				[2]string{"enter/d", "details"},
				[2]string{"b", "backup"},
				[2]string{"r", "restore"},
				[2]string{"D", "delete"},
			)
		}
		return append(hints, tail...)

	case tabNetworks, tabImages:
		hints := nav
		if m.selectedIndex() < m.listLen() {
			hints = append(hints, [2]string{"enter/d", "details"}, [2]string{"D", "delete"})
		}
		return append(hints, tail...)
	}

	return append(nav, tail...)
}
