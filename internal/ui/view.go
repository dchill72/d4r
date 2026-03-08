package ui

import (
	"fmt"
	"strings"

	"d4r/internal/docker"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	base := lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), m.renderBody(), m.renderFooter())

	if m.themePickerActive {
		return overlayCenter(base, m.renderThemePicker(), m.width, m.height)
	}

	return base
}

// overlayCenter composites fg centred over bg using ANSI-aware slicing,
// so the background content remains visible around the picker box.
func overlayCenter(bg, fg string, totalW, totalH int) string {
	fgW := lipgloss.Width(fg)
	fgH := lipgloss.Height(fg)

	x := (totalW - fgW) / 2
	y := (totalH - fgH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		idx := y + i
		for len(bgLines) <= idx {
			bgLines = append(bgLines, "")
		}

		bgLine := bgLines[idx]
		bgW := lipgloss.Width(bgLine)
		fgLineW := lipgloss.Width(fgLine)

		// Left slice of background up to x, padded if the line is short.
		left := ansi.Truncate(bgLine, x, "")
		if lw := lipgloss.Width(left); lw < x {
			left += strings.Repeat(" ", x-lw)
		}

		// Right slice of background after the picker box.
		right := ""
		if x+fgLineW < bgW {
			right = ansi.TruncateLeft(bgLine, x+fgLineW, "")
		}

		bgLines[idx] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}

func (m Model) renderThemePicker() string {
	title := stylePickerTitle.Render("Select Theme")

	var rows []string
	for i, name := range ThemeNames {
		display := ThemeDisplayNames[name]
		if i == m.themePickerCursor {
			rows = append(rows, stylePickerSelected.Render("▶  "+display))
		} else {
			rows = append(rows, stylePickerOption.Render("   "+display))
		}
	}

	hints := lipgloss.JoinVertical(lipgloss.Left,
		stylePickerHint.Render(""),
		stylePickerHint.Render("↑↓ / j k   navigate"),
		stylePickerHint.Render("enter      select"),
		stylePickerHint.Render("esc        cancel"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
		hints,
	)

	return stylePickerBorder.Render(content)
}

// Header: title row + tab row + divider

func (m Model) renderHeader() string {
	// Title row
	title := styleTitle.Render("d4r")
	subtitle := styleVersion.Render(" · Docker TUI")
	loading := ""
	if m.loading {
		loading = styleVersion.Render("  loading...")
	}
	titleRow := title + subtitle + loading

	// Tab row
	var tabs []string
	for i, name := range tabNames {
		if tab(i) == m.tab {
			tabs = append(tabs, styleTabActive.Render(name))
		} else {
			tabs = append(tabs, styleTabInactive.Render(name))
		}
	}
	tabRow := strings.Join(tabs, " ")

	divider := styleDivider.Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, titleRow, tabRow, divider)
}

// Body

func (m Model) renderBody() string {
	switch m.screen {
	case screenDetail:
		return m.detailViewport.View()
	case screenLogs:
		return m.logViewport.View()
	}

	if m.err != nil {
		errMsg := styleError.Render("Error: " + m.err.Error())
		padded := lipgloss.NewStyle().
			Width(m.width).
			Height(m.bodyHeight()).
			Render(errMsg)
		return padded
	}

	switch m.tab {
	case tabContainers:
		return m.renderContainerList()
	case tabVolumes:
		return m.renderVolumeList()
	case tabNetworks:
		return m.renderNetworkList()
	case tabImages:
		return m.renderImageList()
	}
	return ""
}

// Container list

func (m Model) renderContainerList() string {
	if len(m.containers) == 0 {
		empty := styleVersion.Render("No containers found.")
		return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(empty)
	}

	// Column widths
	nameW := 20
	imageW := 30
	statusW := 18
	portsW := m.width - nameW - imageW - statusW - 4 // remainder

	// Header row
	header := styleColHeader.Render(
		padRight("NAME", nameW) + "  " +
			padRight("IMAGE", imageW) + "  " +
			padRight("STATUS", statusW) + "  " +
			"PORTS",
	)

	rows := []string{header}
	for i, ct := range m.containers {
		row := m.renderContainerRow(ct, i == m.containerSelected, nameW, imageW, statusW, portsW)
		rows = append(rows, row)
	}

	body := strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(body)
}

func (m Model) renderContainerRow(ct docker.Container, selected bool, nameW, imageW, statusW, portsW int) string {
	stateText := padRight(ct.State, 9)
	var stateDisplay string
	if selected {
		// No embedded colour — let the row selection style own the whole line uniformly.
		stateDisplay = stateText
	} else {
		stateDisplay = stateStyle(ct.State).Render(stateText)
	}
	statusStr := stateDisplay + " " + truncate(ct.Status, statusW-10)

	line := padRight(ct.Name, nameW) + "  " +
		padRight(truncate(ct.Image, imageW), imageW) + "  " +
		padRight(statusStr, statusW) + "  " +
		truncate(ct.Ports, portsW)

	if selected {
		return styleRowSelected.Render(line)
	}
	return styleRowNormal.Render(line)
}

// Volume list

func (m Model) renderVolumeList() string {
	if len(m.volumes) == 0 {
		empty := styleVersion.Render("No volumes found.")
		return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(empty)
	}

	nameW := 30
	driverW := 12
	sizeW := 10
	refW := 4

	header := styleColHeader.Render(
		padRight("NAME", nameW) + "  " +
			padRight("DRIVER", driverW) + "  " +
			padRight("SIZE", sizeW) + "  " +
			padRight("REFS", refW),
	)

	rows := []string{header}
	for i, v := range m.volumes {
		rows = append(rows, m.renderVolumeRow(v, i == m.volumeSelected, nameW, driverW, sizeW, refW))
	}

	body := strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(body)
}

func (m Model) renderVolumeRow(v docker.Volume, selected bool, nameW, driverW, sizeW, refW int) string {
	size := "unknown"
	if v.Size >= 0 {
		size = formatBytes(v.Size)
	}
	refs := "-"
	if v.RefCount >= 0 {
		refs = fmt.Sprintf("%d", v.RefCount)
	}

	line := padRight(truncate(v.Name, nameW), nameW) + "  " +
		padRight(v.Driver, driverW) + "  " +
		padRight(size, sizeW) + "  " +
		padRight(refs, refW)

	if selected {
		return styleRowSelected.Render(line)
	}
	return styleRowNormal.Render(line)
}

// Network list

func (m Model) renderNetworkList() string {
	if len(m.networks) == 0 {
		empty := styleVersion.Render("No networks found.")
		return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(empty)
	}

	nameW := 22
	driverW := 10
	scopeW := 8
	subnetW := 20

	header := styleColHeader.Render(
		padRight("NAME", nameW) + "  " +
			padRight("DRIVER", driverW) + "  " +
			padRight("SCOPE", scopeW) + "  " +
			padRight("SUBNET", subnetW),
	)

	rows := []string{header}
	for i, n := range m.networks {
		rows = append(rows, m.renderNetworkRow(n, i == m.networkSelected, nameW, driverW, scopeW, subnetW))
	}

	body := strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(body)
}

func (m Model) renderNetworkRow(n docker.Network, selected bool, nameW, driverW, scopeW, subnetW int) string {
	line := padRight(truncate(n.Name, nameW), nameW) + "  " +
		padRight(n.Driver, driverW) + "  " +
		padRight(n.Scope, scopeW) + "  " +
		padRight(n.Subnet, subnetW)

	if selected {
		return styleRowSelected.Render(line)
	}
	return styleRowNormal.Render(line)
}

// Image list

func (m Model) renderImageList() string {
	if len(m.images) == 0 {
		empty := styleVersion.Render("No images found.")
		return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(empty)
	}

	idW := 14
	tagW := 40
	sizeW := 10

	header := styleColHeader.Render(
		padRight("ID", idW) + "  " +
			padRight("TAG", tagW) + "  " +
			padRight("SIZE", sizeW),
	)

	rows := []string{header}
	for i, img := range m.images {
		rows = append(rows, m.renderImageRow(img, i == m.imageSelected, idW, tagW, sizeW))
	}

	body := strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight()).Render(body)
}

func (m Model) renderImageRow(img docker.Image, selected bool, idW, tagW, sizeW int) string {
	tag := "<none>"
	if len(img.Tags) > 0 {
		tag = img.Tags[0]
	}

	line := padRight(img.ID, idW) + "  " +
		padRight(truncate(tag, tagW), tagW) + "  " +
		padRight(formatBytes(img.Size), sizeW)

	if selected {
		return styleRowSelected.Render(line)
	}
	return styleRowNormal.Render(line)
}

// Footer: divider + status bar + key hints

func (m Model) renderFooter() string {
	divider := styleDivider.Render(strings.Repeat("─", m.width))

	// Status bar
	status := m.renderStatus()

	// Key hints
	hints := m.renderHints()

	return lipgloss.JoinVertical(lipgloss.Left, divider, status, hints)
}

func (m Model) renderStatus() string {
	if m.confirm.active {
		msg := styleConfirmPrompt.Render(
			fmt.Sprintf("Confirm %s? (y/n)", m.confirm.action),
		)
		return msg
	}

	switch m.screen {
	case screenDetail:
		return styleStatusBar.Render("Detail view")
	case screenLogs:
		follow := ""
		if m.following {
			follow = "  [following]"
		}
		return styleStatusBar.Render("Logs" + follow)
	}

	count := m.listLen()
	sel := m.selectedIndex() + 1
	if count == 0 {
		sel = 0
	}
	info := fmt.Sprintf("%d/%d", sel, count)

	extra := ""
	if m.tab == tabContainers {
		if m.showAll {
			extra = "  all"
		} else {
			extra = "  running"
		}
	}

	themeName := ThemeDisplayNames[m.currentTheme]
	themeInfo := styleStatusBar.Render("  theme:" + themeName)

	return styleStatusBar.Render(info+extra) + themeInfo
}

func (m Model) renderHints() string {
	hints := m.footerHints()
	var parts []string
	for _, h := range hints {
		key := styleFooterKey.Render(h[0])
		desc := styleFooterDesc.Render(":" + h[1])
		parts = append(parts, key+desc)
	}
	sep := styleFooterSep.Render("  ")
	return strings.Join(parts, sep)
}

// Helpers

// padRight pads or truncates s to exactly n visible characters.
// It uses lipgloss.Width so embedded ANSI escape codes don't distort alignment.
func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w > n {
		// s has no ANSI codes in normal usage; truncate by rune count.
		return truncate(s, n)
	}
	if w < n {
		s += strings.Repeat(" ", n-w)
	}
	return s
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func formatBytes(b int64) string {
	if b < 0 {
		return "unknown"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
