// Package tui — базовая модель для bubbletea v2.
// Модель показывает список элементов с курсором и возвращает выбранный.
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Model — минимальная модель: список элементов с курсором.
// Расширяй под свою задачу (добавляй поля, секции, ввод и т.п.).
type Model struct {
	Title  string
	Items  []string
	cursor int
	quit   bool
}

// NewModel создаёт модель с заголовком и элементами.
func NewModel(title string, items []string) Model {
	return Model{
		Title: title,
		Items: items,
	}
}

// Init — обязательный метод tea.Model. Возвращает команду запуска.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения и возвращает обновлённую модель.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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

// View формирует строку для отрисовки.
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

// Selected возвращает текущий выбранный элемент.
// Пустая строка — если список пуст.
func (m Model) Selected() string {
	if m.cursor < 0 || m.cursor >= len(m.Items) {
		return ""
	}
	return m.Items[m.cursor]
}

// Cursor возвращает индекс текущего элемента.
func (m Model) Cursor() int {
	return m.cursor
}

// Quit сообщает, завершилась ли модель по запросу пользователя
// (а не по ошибке или внешнему сигналу).
func (m Model) Quit() bool {
	return m.quit
}

// Run запускает модель в отдельной программе bubbletea
// и возвращает финальную модель.
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
