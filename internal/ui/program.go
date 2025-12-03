package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"lloader/internal/app"
)

type Program struct {
	program *tea.Program
	logger  *zap.Logger
	config  *app.Config
}

func NewProgram(models []string, config *app.Config, logger *zap.Logger) *Program {
	m := NewModel(models, config, logger)
	p := tea.NewProgram(m, tea.WithAltScreen())

	return &Program{
		program: p,
		logger:  logger,
		config:  config,
	}
}

func (p *Program) Run() (tea.Model, error) {
	return p.program.Run()
}
