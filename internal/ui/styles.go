package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Color definitions - Emerald Green Theme
	ColorPrimary   = lipgloss.Color("48")  // Emerald green
	ColorSecondary = lipgloss.Color("8")   // Bright black
	ColorSuccess   = lipgloss.Color("10")  // Green
	ColorWarning   = lipgloss.Color("11")  // Yellow
	ColorError     = lipgloss.Color("9")   // Red
	ColorInfo      = lipgloss.Color("6")   // Cyan
	ColorBg        = lipgloss.Color("235") // Dark gray background

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(2).
		PaddingRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true)

	// Command list styles
	CommandListStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		MarginTop(1).
		MarginBottom(1)

	CommandItemStyle = lipgloss.NewStyle().
		Padding(0, 2).
		MarginBottom(0)

	CommandItemSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 2).
		PaddingLeft(1).
		Background(ColorSecondary)

	// Search/input styles
	SearchInputStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		MarginBottom(1)

	// Footer/help styles
	HelpStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true).
		MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	// Status info styles
	StatusStyle = lipgloss.NewStyle().
		Foreground(ColorInfo).
		MarginBottom(1)

	BadgeStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(ColorInfo).
		Foreground(lipgloss.Color("0"))
)
