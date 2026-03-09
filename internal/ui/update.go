package ui

import (
	"context"
	"time"

	"d4r/internal/docker"

	"github.com/charmbracelet/bubbles/spinner"
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
type msgContextLoaded struct{ content string }
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
		withTimeout := func(fn func(context.Context) error) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return fn(ctx)
		}

		var containers []docker.Container
		err := withTimeout(func(ctx context.Context) error {
			var listErr error
			containers, listErr = client.ListContainers(ctx, all)
			return listErr
		})
		if err != nil {
			return msgErr{err}
		}

		var volumes []docker.Volume
		err = withTimeout(func(ctx context.Context) error {
			var listErr error
			volumes, listErr = client.ListVolumes(ctx)
			return listErr
		})
		if err != nil {
			return msgErr{err}
		}

		var networks []docker.Network
		err = withTimeout(func(ctx context.Context) error {
			var listErr error
			networks, listErr = client.ListNetworks(ctx)
			return listErr
		})
		if err != nil {
			return msgErr{err}
		}

		var images []docker.Image
		err = withTimeout(func(ctx context.Context) error {
			var listErr error
			images, listErr = client.ListImages(ctx)
			return listErr
		})
		if err != nil {
			return msgErr{err}
		}
		return msgAllLoaded{containers, volumes, networks, images}
	}
}

func fetchContextInfo(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return msgContextLoaded{content: client.ContextReport(ctx)}
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
		} else if m.contextModalActive {
			w, h := contextModalSize(m.width, m.height)
			m.contextViewport.Width = w
			m.contextViewport.Height = h
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

	case msgContextLoaded:
		m.loading = false
		w, h := contextModalSize(m.width, m.height)
		m.contextViewport = viewport.New(w, h)
		m.contextViewport.SetContent(msg.content)
		m.contextModalActive = true
		return m, nil

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

	case spinner.TickMsg:
		if m.spinnerLabel != "" {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case msgBackupDone:
		m.loading = false
		m.spinnerLabel = ""
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
		m.spinnerLabel = ""
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

	// Forward viewport messages.
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
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.themePickerActive {
		return m.handlePickerKey(msg)
	}

	if m.contextModalActive {
		return m.handleContextKey(msg)
	}

	if m.wizard.op != wizardOpNone {
		return m.handleWizardKey(msg)
	}

	if m.confirm.active {
		return m.handleConfirmKey(msg)
	}

	if m.screen == screenDetail {
		return m.handleDetailKey(msg)
	}
	if m.screen == screenLogs {
		return m.handleLogsKey(msg)
	}

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

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

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

	case "j", "down":
		if idx := m.selectedIndex(); idx < m.listLen()-1 {
			m.setSelectedIndex(idx + 1)
		}
	case "k", "up":
		if idx := m.selectedIndex(); idx > 0 {
			m.setSelectedIndex(idx - 1)
		}

	case "f5":
		m.loading = true
		return m, fetchAll(m.docker, m.showAll)

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
	case "c":
		m.loading = true
		return m, fetchContextInfo(m.docker)

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

func (m Model) handleContextKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.contextModalActive = false
		return m, nil
	case "c", "f5":
		m.loading = true
		return m, fetchContextInfo(m.docker)
	}
	var cmd tea.Cmd
	m.contextViewport, cmd = m.contextViewport.Update(msg)
	return m, cmd
}
