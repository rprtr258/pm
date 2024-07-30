package cli

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	background = "#222433"
	selection  = "#1543AD"
	foreground = "#F2F0F2"
	comment    = "#111840"
	green      = "#50fa7b"
	purple     = "#3E2F5C"
	red        = "#ff5555"
	yellow     = "#f1fa8c"
)

func theme() *huh.Theme {
	t := huh.ThemeBase()

	var (
		background = lipgloss.AdaptiveColor{Dark: background}
		selection  = lipgloss.AdaptiveColor{Dark: selection}
		foreground = lipgloss.AdaptiveColor{Dark: foreground}
		comment    = lipgloss.AdaptiveColor{Dark: comment}
		green      = lipgloss.AdaptiveColor{Dark: green}
		purple     = lipgloss.AdaptiveColor{Dark: purple}
		red        = lipgloss.AdaptiveColor{Dark: red}
		yellow     = lipgloss.AdaptiveColor{Dark: yellow}
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(selection)
	t.Focused.Title = t.Focused.Title.Foreground(foreground)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(purple)
	t.Focused.Description = t.Focused.Description.Foreground(comment)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.Directory = t.Focused.Directory.Foreground(purple)
	t.Focused.File = t.Focused.File.Foreground(foreground)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(yellow)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(yellow)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(yellow)
	t.Focused.Option = t.Focused.Option.Foreground(foreground)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(yellow)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(foreground)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(comment)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(foreground).Background(selection).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(foreground).Background(background)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(comment)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return t
}
