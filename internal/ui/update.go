package ui

import (
	"context"
	"os/exec"
	"time"

	"d4r/internal/config"
	"d4r/internal/docker"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages

type msgContainersLoaded struct{ containers []docker.Container }
type msgVolumesLoaded struct{ volumes []docker.Volume }
type msgNetworksLoaded struct{ networks []docker.Network }
type msgImagesLoaded struct{ images []docker.Image }
type msgAllLoaded struct {
	containers []docker.Container
	volumes    []docker.Volume
	networks   []docker.Network
	images     []docker.Image
}
type msgActionDone struct{ err error }
type msgDetailLoaded struct{ content string }
type msgLogsLoaded struct{ content string }
type msgLogTick struct{}
type msgErr struct{ err error }

// Commands

func fetchAll(client *docker.Client, all bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		containers, err := client.ListContainers(ctx, all)
		if err != nil {
			return msgErr{err}
		}
		volumes, err := client.ListVolumes(ctx)
		if err != nil {
			return msgErr{err}
		}
		networks, err := client.ListNetworks(ctx)
		if err != nil {
			return msgErr{err}
		}
		images, err := client.ListImages(ctx)
		if err != nil {
			return msgErr{err}
		}
		return msgAllLoaded{containers, volumes, networks, images}
	}
}

func fetchContainers(client *docker.Client, all bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		containers, err := client.ListContainers(ctx, all)
		if err != nil {
			return msgErr{err}
		}
		return msgContainersLoaded{containers}
	}
}

func doStopContainer(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.StopContainer(ctx, id)}
	}
}

func doStartContainer(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.StartContainer(ctx, id)}
	}
}

func doRemoveContainer(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveContainer(ctx, id, true)}
	}
}

func doRemoveVolume(client *docker.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveVolume(ctx, name, false)}
	}
}

func doRemoveNetwork(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveNetwork(ctx, id)}
	}
}

func doRemoveImage(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveImage(ctx, id, false)}
	}
}

func fetchContainerDetail(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.InspectContainer(ctx, id)
		if err != nil {
			return msgErr{err}
		}
		return msgDetailLoaded{content}
	}
}

func fetchVolumeDetail(client *docker.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.InspectVolume(ctx, name)
		if err != nil {
			return msgErr{err}
		}
		return msgDetailLoaded{content}
	}
}

func fetchNetworkDetail(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.InspectNetwork(ctx, id)
		if err != nil {
			return msgErr{err}
		}
		return msgDetailLoaded{content}
	}
}

func fetchImageDetail(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.InspectImage(ctx, id)
		if err != nil {
			return msgErr{err}
		}
		return msgDetailLoaded{content}
	}
}

func fetchLogs(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.FetchLogs(ctx, id, "500")
		if err != nil {
			return msgErr{err}
		}
		return msgLogsLoaded{content}
	}
}

func logTickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return msgLogTick{}
	})
}

// Update

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vp := viewport.New(m.width, m.bodyHeight())
		if m.screen == screenDetail {
			m.detailViewport.Width = vp.Width
			m.detailViewport.Height = vp.Height
		} else if m.screen == screenLogs {
			m.logViewport.Width = vp.Width
			m.logViewport.Height = vp.Height
		}
		return m, nil

	case msgAllLoaded:
		m.loading = false
		m.containers = msg.containers
		m.volumes = msg.volumes
		m.networks = msg.networks
		m.images = msg.images
		m.clampSelected()
		return m, nil

	case msgContainersLoaded:
		m.loading = false
		m.containers = msg.containers
		m.clampSelected()
		return m, nil

	case msgActionDone:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m, fetchAll(m.docker, m.showAll)

	case msgDetailLoaded:
		m.loading = false
		m.detailViewport = viewport.New(m.width, m.bodyHeight())
		m.detailViewport.SetContent(msg.content)
		m.screen = screenDetail
		return m, nil

	case msgLogsLoaded:
		m.loading = false
		m.logViewport = viewport.New(m.width, m.bodyHeight())
		m.logViewport.SetContent(msg.content)
		m.logViewport.GotoBottom()
		m.screen = screenLogs
		var cmd tea.Cmd
		if m.following {
			cmd = logTickCmd()
		}
		return m, cmd

	case msgLogTick:
		if m.screen != screenLogs || !m.following {
			return m, nil
		}
		ct := m.selectedContainer()
		if ct == nil {
			return m, nil
		}
		return m, fetchLogs(m.docker, ct.FullID)

	case msgErr:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward viewport messages
	if m.screen == screenDetail {
		var cmd tea.Cmd
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd
	}
	if m.screen == screenLogs {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Theme picker intercepts all keys while active
	if m.themePickerActive {
		return m.handlePickerKey(msg)
	}

	// Confirm dialog
	if m.confirm.active {
		return m.handleConfirmKey(msg)
	}

	// Detail / log screens
	if m.screen == screenDetail {
		return m.handleDetailKey(msg)
	}
	if m.screen == screenLogs {
		return m.handleLogsKey(msg)
	}

	// Clear error on any key
	m.err = nil

	return m.handleListKey(msg)
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := m.confirm.action
		target := m.confirm.target
		m.confirm = confirmState{}
		m.loading = true
		switch action {
		case "stop":
			return m, doStopContainer(m.docker, target)
		case "delete-container":
			return m, doRemoveContainer(m.docker, target)
		case "delete-volume":
			return m, doRemoveVolume(m.docker, target)
		case "delete-network":
			return m, doRemoveNetwork(m.docker, target)
		case "delete-image":
			return m, doRemoveImage(m.docker, target)
		}
	case "n", "N", "esc":
		m.confirm = confirmState{}
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenList
		return m, nil
	}
	var cmd tea.Cmd
	m.detailViewport, cmd = m.detailViewport.Update(msg)
	return m, cmd
}

func (m Model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenList
		m.following = false
		return m, nil
	case "f":
		m.following = !m.following
		if m.following {
			return m, logTickCmd()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.logViewport, cmd = m.logViewport.Update(msg)
	return m, cmd
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	// Tab switching
	case "tab":
		m.tab = tab((int(m.tab) + 1) % 4)
		return m, nil
	case "shift+tab":
		m.tab = tab((int(m.tab) + 3) % 4)
		return m, nil
	case "1":
		m.tab = tabContainers
	case "2":
		m.tab = tabVolumes
	case "3":
		m.tab = tabNetworks
	case "4":
		m.tab = tabImages

	// Navigation
	case "j", "down":
		if idx := m.selectedIndex(); idx < m.listLen()-1 {
			m.setSelectedIndex(idx + 1)
		}
	case "k", "up":
		if idx := m.selectedIndex(); idx > 0 {
			m.setSelectedIndex(idx - 1)
		}

	// Refresh
	case "r":
		m.loading = true
		return m, fetchAll(m.docker, m.showAll)

	// Container-specific
	case "a":
		if m.tab == tabContainers {
			m.showAll = !m.showAll
			m.loading = true
			return m, fetchContainers(m.docker, m.showAll)
		}

	case "enter", "d":
		return m.openDetail()

	case "l":
		if m.tab == tabContainers {
			ct := m.selectedContainer()
			if ct != nil {
				m.loading = true
				return m, fetchLogs(m.docker, ct.FullID)
			}
		}

	case "s":
		if m.tab == tabContainers {
			ct := m.selectedContainer()
			if ct != nil && ct.State == "running" {
				return m.shellIntoContainer(ct.FullID)
			}
		}

	case "x":
		if m.tab == tabContainers {
			ct := m.selectedContainer()
			if ct != nil && ct.State == "running" {
				m.confirm = confirmState{true, "stop", ct.FullID}
			}
		}

	case "u":
		if m.tab == tabContainers {
			ct := m.selectedContainer()
			if ct != nil && ct.State != "running" {
				m.loading = true
				return m, doStartContainer(m.docker, ct.FullID)
			}
		}

	case "t":
		m.themePickerActive = true
		m.themePickerCursor = ThemeIndex(m.currentTheme)
		return m, nil

	case "D":
		switch m.tab {
		case tabContainers:
			if ct := m.selectedContainer(); ct != nil {
				m.confirm = confirmState{true, "delete-container", ct.FullID}
			}
		case tabVolumes:
			if v := m.selectedVolume(); v != nil {
				m.confirm = confirmState{true, "delete-volume", v.Name}
			}
		case tabNetworks:
			if n := m.selectedNetwork(); n != nil {
				m.confirm = confirmState{true, "delete-network", n.ID}
			}
		case tabImages:
			if img := m.selectedImage(); img != nil {
				m.confirm = confirmState{true, "delete-image", img.FullID}
			}
		}
	}

	return m, nil
}

func (m Model) openDetail() (tea.Model, tea.Cmd) {
	m.loading = true
	switch m.tab {
	case tabContainers:
		if ct := m.selectedContainer(); ct != nil {
			return m, fetchContainerDetail(m.docker, ct.FullID)
		}
	case tabVolumes:
		if v := m.selectedVolume(); v != nil {
			return m, fetchVolumeDetail(m.docker, v.Name)
		}
	case tabNetworks:
		if n := m.selectedNetwork(); n != nil {
			return m, fetchNetworkDetail(m.docker, n.ID)
		}
	case tabImages:
		if img := m.selectedImage(); img != nil {
			return m, fetchImageDetail(m.docker, img.FullID)
		}
	}
	m.loading = false
	return m, nil
}

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
		// Revert to the previously confirmed theme
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

func (m Model) shellIntoContainer(id string) (tea.Model, tea.Cmd) {
	cmd := exec.Command("docker", "exec", "-it", id, "/bin/sh")
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return msgActionDone{err}
	})
}

func (m *Model) clampSelected() {
	clamp := func(idx, length int) int {
		if length == 0 {
			return 0
		}
		if idx >= length {
			return length - 1
		}
		if idx < 0 {
			return 0
		}
		return idx
	}
	m.containerSelected = clamp(m.containerSelected, len(m.containers))
	m.volumeSelected = clamp(m.volumeSelected, len(m.volumes))
	m.networkSelected = clamp(m.networkSelected, len(m.networks))
	m.imageSelected = clamp(m.imageSelected, len(m.images))
}
