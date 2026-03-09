package ui

import (
	"context"
	"time"

	"d4r/internal/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func doRemoveNetwork(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveNetwork(ctx, id)}
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
