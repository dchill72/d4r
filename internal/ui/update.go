package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"d4r/internal/config"
	"d4r/internal/docker"

	"github.com/charmbracelet/bubbles/textinput"
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

type msgVolumeContainers struct {
	containers []docker.Container
	err        error
}
type msgTarSummary struct {
	summary string
	err     error
}
type msgContainersStopped struct {
	stoppedIDs []string
	err        error
}
type msgBackupDone struct{ err error }
type msgRestoreDone struct{ err error }
type msgContainersRestarted struct{ err error }

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

func fetchContainersForVolume(client *docker.Client, volumeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		containers, err := client.GetContainersForVolume(ctx, volumeName)
		return msgVolumeContainers{containers: containers, err: err}
	}
}

func summarizeTarCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if _, err := os.Stat(path); err != nil {
			return msgTarSummary{err: fmt.Errorf("file not found: %s", path)}
		}
		out, err := exec.Command("tar", "-tzf", path).Output()
		if err != nil {
			return msgTarSummary{err: fmt.Errorf("cannot read archive: %w", err)}
		}
		lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		total := len(lines)
		shown := lines
		if total > 30 {
			shown = lines[:30]
		}
		summary := strings.Join(shown, "\n")
		if total > 30 {
			summary += fmt.Sprintf("\n... and %d more", total-30)
		}
		summary += fmt.Sprintf("\n\nTotal: %d entries", total)
		return msgTarSummary{summary: summary}
	}
}

func doStopContainersCmd(client *docker.Client, containers []docker.Container) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var stopped []string
		var lastErr error
		for _, ct := range containers {
			if err := client.StopContainer(ctx, ct.FullID); err != nil {
				lastErr = err
			} else {
				stopped = append(stopped, ct.FullID)
			}
		}
		return msgContainersStopped{stoppedIDs: stopped, err: lastErr}
	}
}

func doBackupVolumeCmd(client *docker.Client, volumeName, destPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		return msgBackupDone{client.BackupVolume(ctx, volumeName, destPath)}
	}
}

func doRestoreVolumeCmd(client *docker.Client, volumeName, sourcePath string, replace bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		return msgRestoreDone{client.RestoreVolume(ctx, volumeName, sourcePath, replace)}
	}
}

func doStartContainersCmd(client *docker.Client, ids []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var lastErr error
		for _, id := range ids {
			if err := client.StartContainer(ctx, id); err != nil {
				lastErr = err
			}
		}
		return msgContainersRestarted{err: lastErr}
	}
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

	case msgVolumeContainers:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.wizard = volumeWizard{}
			return m, nil
		}
		var running []docker.Container
		for _, ct := range msg.containers {
			if ct.State == "running" {
				running = append(running, ct)
			}
		}
		if len(running) > 0 {
			m.wizard.runningContainers = running
			m.wizard.step = wizardStepStopConfirm
			return m, nil
		}
		return m.proceedWithOperation()

	case msgTarSummary:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.wizard = volumeWizard{}
			return m, nil
		}
		m.wizard.tarSummary = msg.summary
		m.wizard.step = wizardStepTarSummary
		return m, nil

	case msgContainersStopped:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.wizard = volumeWizard{}
			return m, nil
		}
		m.wizard.stoppedIDs = msg.stoppedIDs
		return m.proceedWithOperation()

	case msgBackupDone:
		m.loading = false
		stoppedIDs := m.wizard.stoppedIDs
		m.wizard = volumeWizard{}
		if msg.err != nil {
			m.err = msg.err
		}
		if len(stoppedIDs) > 0 {
			m.loading = true
			return m, doStartContainersCmd(m.docker, stoppedIDs)
		}
		return m, fetchAll(m.docker, m.showAll)

	case msgRestoreDone:
		m.loading = false
		stoppedIDs := m.wizard.stoppedIDs
		m.wizard = volumeWizard{}
		if msg.err != nil {
			m.err = msg.err
		}
		if len(stoppedIDs) > 0 {
			m.loading = true
			return m, doStartContainersCmd(m.docker, stoppedIDs)
		}
		return m, fetchAll(m.docker, m.showAll)

	case msgContainersRestarted:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		}
		return m, fetchAll(m.docker, m.showAll)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward messages to textinput when wizard is in input mode (e.g. blink ticks).
	if m.wizard.op != wizardOpNone && m.wizard.step == wizardStepInput {
		var cmd tea.Cmd
		m.wizard.input, cmd = m.wizard.input.Update(msg)
		return m, cmd
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

	// Volume wizard intercepts all keys while active
	if m.wizard.op != wizardOpNone {
		return m.handleWizardKey(msg)
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
	case "f5":
		m.loading = true
		return m, fetchAll(m.docker, m.showAll)

	// Volume backup / restore
	case "b":
		if m.tab == tabVolumes {
			if v := m.selectedVolume(); v != nil {
				return m.startBackupWizard(v.Name)
			}
		}
	case "r":
		if m.tab == tabVolumes {
			if v := m.selectedVolume(); v != nil {
				return m.startRestoreWizard(v.Name)
			}
		}

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

func (m Model) startBackupWizard(volumeName string) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Width = 52
	ti.SetValue(volumeName + "-" + time.Now().Format("20060102-150405") + ".tar.gz")
	focusCmd := ti.Focus()
	m.wizard = volumeWizard{
		op:         wizardOpBackup,
		step:       wizardStepInput,
		volumeName: volumeName,
		input:      ti,
	}
	return m, focusCmd
}

func (m Model) startRestoreWizard(volumeName string) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Width = 52
	ti.Placeholder = "path/to/backup.tar.gz"
	focusCmd := ti.Focus()
	m.wizard = volumeWizard{
		op:         wizardOpRestore,
		step:       wizardStepInput,
		volumeName: volumeName,
		input:      ti,
	}
	return m, focusCmd
}

func (m Model) proceedWithOperation() (tea.Model, tea.Cmd) {
	m.loading = true
	switch m.wizard.op {
	case wizardOpBackup:
		return m, doBackupVolumeCmd(m.docker, m.wizard.volumeName, m.wizard.destPath)
	case wizardOpRestore:
		return m, doRestoreVolumeCmd(m.docker, m.wizard.volumeName, m.wizard.sourcePath, m.wizard.replaceMode)
	}
	return m, nil
}

func (m Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Block input while an async operation is in flight.
	if m.loading {
		return m, nil
	}

	switch m.wizard.step {
	case wizardStepInput:
		switch msg.String() {
		case "esc":
			m.wizard = volumeWizard{}
			return m, nil
		case "enter":
			path := strings.TrimSpace(m.wizard.input.Value())
			if path == "" {
				return m, nil
			}
			if m.wizard.op == wizardOpBackup {
				m.wizard.destPath = path
				m.loading = true
				return m, fetchContainersForVolume(m.docker, m.wizard.volumeName)
			}
			// Restore: summarise archive first.
			m.wizard.sourcePath = path
			m.loading = true
			return m, summarizeTarCmd(path)
		default:
			var cmd tea.Cmd
			m.wizard.input, cmd = m.wizard.input.Update(msg)
			return m, cmd
		}

	case wizardStepTarSummary:
		switch msg.String() {
		case "y", "enter":
			m.wizard.step = wizardStepRestoreMode
		case "n", "esc":
			m.wizard = volumeWizard{}
		}
		return m, nil

	case wizardStepRestoreMode:
		switch msg.String() {
		case "m":
			m.wizard.replaceMode = false
			m.loading = true
			return m, fetchContainersForVolume(m.docker, m.wizard.volumeName)
		case "r":
			m.wizard.replaceMode = true
			m.loading = true
			return m, fetchContainersForVolume(m.docker, m.wizard.volumeName)
		case "esc":
			m.wizard = volumeWizard{}
		}
		return m, nil

	case wizardStepStopConfirm:
		switch msg.String() {
		case "y", "Y":
			m.loading = true
			return m, doStopContainersCmd(m.docker, m.wizard.runningContainers)
		case "n", "N", "esc":
			m.wizard = volumeWizard{}
		}
		return m, nil
	}

	return m, nil
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
