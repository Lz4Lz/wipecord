package tui

import (
	"fmt"
	"log"
	"purge/internal/discord"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Todo: add winsize detection for the slicer

type DMSelector struct {
	options      []string
	filtered     []string
	cursor       int
	sliceIndex   int
	width        int
	height       int
	selectedDMID string
	searchInput  string
	Client       *discord.Client
}

func NewDMSelector(client *discord.Client) *DMSelector {

	err := client.FetchDMS()

	if err != nil {
		log.Fatalf("Failed to fetch DMs: %v", err)
	}

	var options []string

	for _, ch := range client.DMS {
		var name string

		switch ch.Type {
		case 1: // DM
			if len(ch.Recipients) > 0 {
				name = ch.Recipients[0].Username
			} else {
				name = "(unknown DM)"
			}
		case 3: // Group DM
			if ch.Name != "" {
				name = ch.Name
			} else {
				var names []string
				for _, r := range ch.Recipients {
					names = append(names, r.Username)
				}
				name = "(Group: " + strings.Join(names, ", ") + ")"
			}
		default:
			name = "(unknown channel type)"
		}

		option := fmt.Sprintf("%s: %s", name, ch.ID)
		options = append(options, option)
	}

	return &DMSelector{
		options:  options,
		filtered: options,
		Client:   client,
	}
}

func (m *DMSelector) Init() tea.Cmd {
	return nil
}

func (m *DMSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.sliceIndex > 0 {
				m.sliceIndex--
			} else {
				m.sliceIndex = max(0, len(m.filtered)-25)
				m.cursor = min(24, len(m.filtered)-1)
			}

		case "down", "j":
			if m.cursor < 24 && m.sliceIndex+m.cursor+1 < len(m.filtered) {
				m.cursor++
			} else if m.sliceIndex+25 < len(m.filtered) {
				m.sliceIndex++
			} else {
				m.sliceIndex = 0
				m.cursor = 0
			}

		case "backspace":
			if len(m.searchInput) > 0 {
				m.searchInput = m.searchInput[:len(m.searchInput)-1]
				m.updateFiltered()
			}

		case "enter":
			if m.cursor < len(m.filtered) {
				selected := m.filtered[m.cursor]
				parts := strings.Split(selected, ": ")
				if len(parts) == 2 {
					m.selectedDMID = parts[1]
				}

				settings := NewSettingsModel(m.Client)
				settings.SetChannelID(m.selectedDMID)

				return settings, nil

			}

		default:
			if len(msg.String()) == 1 {
				m.searchInput += msg.String()
				m.updateFiltered()
			}
		}
	}
	return m, nil
}

func (m *DMSelector) updateFiltered() {
	if m.searchInput == "" {
		m.filtered = m.options
	} else {
		var filtered []string
		for _, option := range m.options {
			if strings.Contains(strings.ToLower(option), strings.ToLower(m.searchInput)) {
				filtered = append(filtered, option)
			}
		}
		m.filtered = filtered
	}
	m.cursor = 0
	m.sliceIndex = 0
}

func sliceWindow(original []string, start, windowSize int) []string {
	if len(original) == 0 || windowSize <= 0 {
		return []string{}
	}

	if start < 0 {
		start = 0
	}
	if start > len(original) {
		start = len(original)
	}

	end := start + windowSize
	if end > len(original) {
		end = len(original)
		start = max(0, end-windowSize)
	}

	return original[start:end]
}

func (m *DMSelector) View() string {
	items := sliceWindow(m.filtered, m.sliceIndex, 25)

	menuStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("129")).
		Margin(1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	header := lipgloss.NewStyle().Bold(true).Render("Search: " + m.searchInput)

	var menuItems []string
	for i, item := range items {
		if i == m.cursor {
			menuItems = append(menuItems, selectedStyle.Render("> "+item+" <"))
		} else {
			menuItems = append(menuItems, unselectedStyle.Render("  "+item+"  "))
		}
	}

	menuContent := lipgloss.JoinVertical(
		lipgloss.Center,
		append([]string{header}, menuItems...)...,
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		menuStyle.Render(menuContent),
	)
}
