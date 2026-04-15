package ui

import (
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

// This file defines all terminal UI styling constants for gemtracker.
// It uses a dark slate/blue ANSI 256-color palette suitable for long terminal sessions,
// with specialized colors for status indicators (success green, warning yellow, danger red).
// All styles are built using lipgloss for composable, terminal-safe formatting.

// Color palette - dark slate/blue theme (256-color ANSI)
const (
	ColorBg           = "235" // #262626 - base background
	ColorSurface      = "237" // #3a3a3a - cards/panels
	ColorBorder       = "240" // #585858 - default borders
	ColorBorderActive = "74"  // #5fafd7 - focused border (slate blue)
	ColorText         = "252" // #d0d0d0 - primary text
	ColorTextMuted    = "244" // #808080 - secondary text
	ColorTextSubtle   = "240" // #585858 - hints, tree connectors
	ColorPrimary      = "74"  // #5fafd7 - app accent
	ColorSuccess      = "71"  // #5faf5f - latest/up to date
	ColorWarning      = "178" // #d7af00 - outdated
	ColorDanger       = "160" // #d70000 - vulnerable (CRITICAL)
	ColorHigh         = "208" // #ff8700 - high severity (orange)
	ColorSelected     = "24"  // #005f87 - selected row background
	ColorTabActive    = "74"  // same as Primary
	ColorTabInactive  = "244" // same as TextMuted
)

// Layout constants
const (
	FixedChrome     = 3 // header (1) + tabbar (1) + statusbar (1)
	HeaderHeight    = 3
	TabBarHeight    = 1
	StatusBarHeight = 1
)

// ============================================================================
// Base Container Style
// ============================================================================

// AppBackgroundStyle ensures consistent dark background regardless of terminal theme
var AppBackgroundStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#262626"))

// ============================================================================
// App Chrome Styles
// ============================================================================

var AppHeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorPrimary)).
	Bold(true).
	Background(lipgloss.Color("#262626")).
	Padding(0, 2)

var ProjectPathStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted)).
	Padding(0, 2)

var AppVersionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

// ============================================================================
// Tab Bar Styles
// ============================================================================

var TabStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTabInactive)).
	Padding(0, 2).
	Background(lipgloss.Color("#3a3a3a"))

var TabActiveStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTabActive)).
	Bold(true).
	Padding(0, 2).
	Background(lipgloss.Color("#3a3a3a")).
	Underline(true)

// ============================================================================
// Status Bar Styles
// ============================================================================

var StatusBarStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#3a3a3a")).
	Foreground(lipgloss.Color(ColorTextMuted)).
	Padding(0, 2)

var KeyHintKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorPrimary)).
	Bold(true)

var KeyHintDescStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

// ============================================================================
// Table / List Row Styles
// ============================================================================

var TableHeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted)).
	Background(lipgloss.Color("#262626")).
	Bold(true).
	BorderBottom(true).
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(ColorBorder))

var RowNormalStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorText)).
	Background(lipgloss.Color("#262626"))

var RowSelectedStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorSelected)).
	Foreground(lipgloss.Color(ColorText)).
	Bold(true)

var RowMutedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

// ============================================================================
// Badge/Status Indicator Styles
// ============================================================================

var BadgeOKStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorSuccess))

var BadgeOutdatedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorWarning))

var BadgeVulnerableStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorDanger))

var BadgeLoadingStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

var BadgeErrorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorDanger))

var BadgeHealthyDotStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorSuccess))

var BadgeWarningDotStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorWarning))

var BadgeCriticalDotStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorDanger))

var BadgeHighDotStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorHigh))

// ============================================================================
// Panel/Container Styles
// ============================================================================

var PanelTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorPrimary)).
	Bold(true)

var PanelBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorBorder)).
	Background(lipgloss.Color("#262626")).
	Padding(0, 1)

var PanelBorderActiveStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorBorderActive)).
	Background(lipgloss.Color("#262626")).
	Padding(0, 1)

// ============================================================================
// Input Styles
// ============================================================================

var SearchPromptStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorPrimary)).
	Bold(true)

var SearchBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorBorderActive)).
	Background(lipgloss.Color("#262626")).
	Padding(0, 1)

var InputBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorBorder)).
	Foreground(lipgloss.Color(ColorText)).
	Background(lipgloss.Color("#262626")).
	Padding(0, 1)

// ============================================================================
// Tree Styles
// ============================================================================

var TreeConnectorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextSubtle))

var TreeGemNameStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

var TreeVersionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

// ============================================================================
// Loading/Spinner Styles
// ============================================================================

var SpinnerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorPrimary)).
	Bold(true)

var LoadingMessageStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextMuted))

// ============================================================================
// Error Styles
// ============================================================================

var ErrorBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorDanger)).
	Foreground(lipgloss.Color(ColorDanger)).
	Bold(true).
	Background(lipgloss.Color("#262626")).
	Padding(1, 2)

var ErrorTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorDanger)).
	Bold(true)

var ErrorMessageStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorText))

// ============================================================================
// Update Notification Styles
// ============================================================================

var UpdateBarStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#3a3a3a")).
	Foreground(lipgloss.Color(ColorWarning)).
	Padding(0, 2)

// ============================================================================
// Markdown Rendering (Glamour)
// ============================================================================

// NewMarkdownRenderer creates a glamour renderer with our dark color scheme
// Uses WithStandardStyle("dark") to enforce consistent dark background regardless of terminal settings
func NewMarkdownRenderer(width int) *glamour.TermRenderer {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStandardStyle("dark"),
	)
	return renderer
}
