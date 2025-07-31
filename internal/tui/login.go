package tui

import (
	"purge/internal/discord"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	errMsg error
)

type model struct {
	textInput textinput.Model
	err       error
	width     int
	height    int
}

func LoginModel() *model {
	ti := textinput.New()
	ti.Placeholder = "Token"
	ti.Focus()
	ti.CharLimit = 156
	//ti.Width = 50

	return &model{
		textInput: ti,
		err:       nil,
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			//TokenCheck
			value := m.textInput.Value()
			c := discord.NewClient(value)
			err := c.TokenCheck()
			if err != nil {
				m.err = err
				return m, nil
			}
			MainMenu := NewDMSelector(c)
			return MainMenu, nil

		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *model) View() string {

	titlestyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6600CC")).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6600CC")).MarginTop(1).Align(lipgloss.Center)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).MarginTop(1).Align(lipgloss.Center)

	components := []string{
		titlestyle.Render("Enter Discord Token:"),
		m.textInput.View(),
		infoStyle.Render("Press Enter to continue"),
	}

	if m.err != nil {
		components = append(components, errorStyle.Render(m.err.Error()))
	}

	ui := lipgloss.JoinVertical(lipgloss.Center, components...)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center, lipgloss.Center,
		ui,
	)
}
