package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("39")  // 蓝
	colorAccent  = lipgloss.Color("213") // 粉
	colorMuted   = lipgloss.Color("245")
	colorOK      = lipgloss.Color("42")
	colorWarn    = lipgloss.Color("214")
	colorErr     = lipgloss.Color("196")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(colorPrimary).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().Foreground(colorMuted)

	itemStyle       = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle   = lipgloss.NewStyle().PaddingLeft(0).Foreground(colorAccent).Bold(true)
	checkedStyle    = lipgloss.NewStyle().Foreground(colorOK)
	helpStyle       = lipgloss.NewStyle().Foreground(colorMuted).MarginTop(1)
	errStyle        = lipgloss.NewStyle().Foreground(colorErr)
	okStyle         = lipgloss.NewStyle().Foreground(colorOK)
	warnStyle       = lipgloss.NewStyle().Foreground(colorWarn)
	boxStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorPrimary).Padding(0, 1)
	fieldLabelStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
)
