package ui

import (
	"d4r/internal/docker"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type tab int

const (
	tabContainers tab = iota
	tabVolumes
	tabNetworks
	tabImages
)

var tabNames = []string{"Containers", "Volumes", "Networks", "Images"}

type screen int

const (
	screenList   screen = iota
	screenDetail        // detail viewport for any entity
	screenLogs          // log viewport for containers
)

type confirmState struct {
	active bool
	action string // "stop", "delete", "start"
	target string // full entity ID/name
}

type Model struct {
	docker *docker.Client

	// Navigation
	tab    tab
	screen screen

	// Containers
	containers        []docker.Container
	containerSelected int
	showAll           bool

	// Volumes
	volumes       []docker.Volume
	volumeSelected int

	// Networks
	networks        []docker.Network
	networkSelected int

	// Images
	images        []docker.Image
	imageSelected int

	// Shared state
	loading bool
	err     error
	confirm confirmState

	// Theme picker
	themePickerActive  bool
	themePickerCursor  int
	currentTheme       string

	// Detail / log viewports
	detailViewport viewport.Model
	logViewport    viewport.Model
	following      bool

	// Terminal size
	width  int
	height int
}

func NewModel(client *docker.Client, theme string) Model {
	return Model{
		docker:       client,
		tab:          tabContainers,
		screen:       screenList,
		width:        80,
		height:       24,
		currentTheme: theme,
		showAll:      true,
	}
}

func (m Model) Init() tea.Cmd {
	return fetchAll(m.docker, m.showAll)
}

// Convenience helpers

func (m *Model) selectedIndex() int {
	switch m.tab {
	case tabContainers:
		return m.containerSelected
	case tabVolumes:
		return m.volumeSelected
	case tabNetworks:
		return m.networkSelected
	case tabImages:
		return m.imageSelected
	}
	return 0
}

func (m *Model) setSelectedIndex(i int) {
	switch m.tab {
	case tabContainers:
		m.containerSelected = i
	case tabVolumes:
		m.volumeSelected = i
	case tabNetworks:
		m.networkSelected = i
	case tabImages:
		m.imageSelected = i
	}
}

func (m *Model) listLen() int {
	switch m.tab {
	case tabContainers:
		return len(m.containers)
	case tabVolumes:
		return len(m.volumes)
	case tabNetworks:
		return len(m.networks)
	case tabImages:
		return len(m.images)
	}
	return 0
}

func (m *Model) selectedContainer() *docker.Container {
	if len(m.containers) == 0 || m.containerSelected < 0 || m.containerSelected >= len(m.containers) {
		return nil
	}
	return &m.containers[m.containerSelected]
}

func (m *Model) selectedVolume() *docker.Volume {
	if len(m.volumes) == 0 || m.volumeSelected < 0 || m.volumeSelected >= len(m.volumes) {
		return nil
	}
	return &m.volumes[m.volumeSelected]
}

func (m *Model) selectedNetwork() *docker.Network {
	if len(m.networks) == 0 || m.networkSelected < 0 || m.networkSelected >= len(m.networks) {
		return nil
	}
	return &m.networks[m.networkSelected]
}

func (m *Model) selectedImage() *docker.Image {
	if len(m.images) == 0 || m.imageSelected < 0 || m.imageSelected >= len(m.images) {
		return nil
	}
	return &m.images[m.imageSelected]
}

// Layout constants
const (
	headerHeight = 3 // title+tabs line + divider
	footerHeight = 3 // divider + status + hints
)

func (m Model) bodyHeight() int {
	h := m.height - headerHeight - footerHeight
	if h < 1 {
		h = 1
	}
	return h
}
