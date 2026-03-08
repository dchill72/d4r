package main

import (
	"fmt"
	"os"

	"d4r/internal/config"
	"d4r/internal/docker"
	"d4r/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = config.Default()
	}

	ui.ApplyTheme(cfg.Theme)

	client, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	p := tea.NewProgram(
		ui.NewModel(client, cfg.Theme),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
