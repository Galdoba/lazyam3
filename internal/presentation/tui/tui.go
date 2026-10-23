// Package tui provides a basic model for bubbletea v2.
// The model shows a list of items with a cursor and returns the selected one.
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Model is a minimal model: a list of items with a cursor.
// Extend it for your task (add fields, sections, input, etc.).
type Model struct {
	Title  string
	Items  []string
	cursor int
	quit   bool
}

// NewModel creates a model with a title and items.
func NewModel(title string, items []string) Model {
	return Model{
		Title: title,
		Items: items,
	}
}

// Init is a required tea.Model method. Returns the startup command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update processes messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.Items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// View formats the string for rendering.
func (m Model) View() tea.View {
	var b strings.Builder

	if m.Title != "" {
		b.WriteString(m.Title)
		b.WriteString("\n\n")
	}

	for i, item := range m.Items {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, item)
	}

	b.WriteString("\n(↑/↓ navigate, q/esc quit)\n")
	return tea.NewView(b.String())
}

// Selected returns the currently selected item.
// Empty string if the list is empty.
func (m Model) Selected() string {
	if m.cursor < 0 || m.cursor >= len(m.Items) {
		return ""
	}
	return m.Items[m.cursor]
}

// Cursor returns the index of the current item.
func (m Model) Cursor() int {
	return m.cursor
}

// Quit reports whether the model was terminated by user request
// (not by an error or external signal).
func (m Model) Quit() bool {
	return m.quit
}

// Run starts the model in a separate bubbletea program
// and returns the final model.
func Run(m Model) (Model, error) {
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return m, fmt.Errorf("tui run failed: %w", err)
	}
	result, ok := final.(Model)
	if !ok {
		return m, fmt.Errorf("unexpected model type: %T", final)
	}
	return result, nil
}
