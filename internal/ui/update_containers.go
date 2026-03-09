package ui

import (
	"context"
	"os/exec"
	"time"

	"d4r/internal/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// Commands

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

// Handlers

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

func (m Model) shellIntoContainer(id string) (tea.Model, tea.Cmd) {
	cmd := exec.Command("docker", "exec", "-it", id, "/bin/sh")
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return msgActionDone{err}
	})
}
