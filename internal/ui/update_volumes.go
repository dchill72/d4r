package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"d4r/internal/docker"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Commands

func doRemoveVolume(client *docker.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveVolume(ctx, name, false)}
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

// Handlers

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
