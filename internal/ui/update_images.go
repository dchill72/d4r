package ui

import (
	"context"
	"time"

	"d4r/internal/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func doRemoveImage(client *docker.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return msgActionDone{client.RemoveImage(ctx, id, false)}
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
