package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"purge/internal/discord"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingsModel struct {
	client *discord.Client

	channel  textinput.Model
	filters  textinput.Model
	searchMs textinput.Model
	deleteMs textinput.Model

	cursor        int
	width, height int
}

func NewSettingsModel(client *discord.Client) *SettingsModel {

	ch := textinput.New()
	ch.Placeholder = "Channel ID / DM ID"

	f := textinput.New()
	f.Placeholder = "word1,word2,word3"

	sd := textinput.New()
	sd.Placeholder = "3000"

	dd := textinput.New()
	dd.Placeholder = "2000"

	ch.Focus()

	return &SettingsModel{
		client:   client,
		channel:  ch,
		filters:  f,
		searchMs: sd,
		deleteMs: dd,
	}
}

func (m *SettingsModel) SetChannelID(id string) {
	m.channel.SetValue(id)
}

func (m *SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SettingsModel) updateFocus() {
	m.channel.Blur()
	m.filters.Blur()
	m.searchMs.Blur()
	m.deleteMs.Blur()

	switch m.cursor {
	case 0:
		m.channel.Focus()
	case 1:
		m.filters.Focus()
	case 2:
		m.searchMs.Focus()
	case 3:
		m.deleteMs.Focus()
	}
}

func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			m.updateFocus()

		case tea.KeyDown:
			if m.cursor < 3 {
				m.cursor++
			}
			m.updateFocus()

		case tea.KeyEnter:
			return m.buildPurgeModel()

		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	var cmd1, cmd2, cmd3, cmd4 tea.Cmd
	m.channel, cmd1 = m.channel.Update(msg)
	m.filters, cmd2 = m.filters.Update(msg)
	m.searchMs, cmd3 = m.searchMs.Update(msg)
	m.deleteMs, cmd4 = m.deleteMs.Update(msg)

	return m, tea.Batch(cmd1, cmd2, cmd3, cmd4)
}

func (m *SettingsModel) buildPurgeModel() (tea.Model, tea.Cmd) {
	dmid := strings.TrimSpace(m.channel.Value())
	if dmid == "" {
		dmid = "0"
	}

	filters := []string{}

	filtersValue := strings.TrimSpace(m.filters.Value())
	if filtersValue != "" {
		filters = strings.Split(filtersValue, ",")
		for i, f := range filters {
			filters[i] = strings.TrimSpace(f)
		}
	}
	pm := NewPurgeModel(dmid, m.client)

	searchMsValue := m.searchMs.Value()
	searchMsInt, _ := strconv.Atoi(searchMsValue)

	deleteMsValue := m.deleteMs.Value()
	deleteMsInt, _ := strconv.Atoi(deleteMsValue)

	pm.Filters = filters
	pm.SearchDelay = time.Millisecond * time.Duration(searchMsInt)
	pm.DeleteDelay = time.Millisecond * time.Duration(deleteMsInt)

	return pm, func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
}

func (m *SettingsModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("129")).
		Padding(1, 2)

	pinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0AFF")).Bold(true)

	content := fmt.Sprintf(
		"Purge Settings\n\n"+
			"Channel ID:\n%s\n\n"+
			"Filters (comma-separated):\n%s\n\n"+
			"Search Delay (ms):\n%s\n\n"+
			"Delete Delay (ms):\n%s\n\n"+
			"%s Start Purge   %s Quit",
		m.channel.View(),
		m.filters.View(),
		m.searchMs.View(),
		m.deleteMs.View(),
		pinkStyle.Render("[Enter]"),
		pinkStyle.Render("[Esc]"),
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}
