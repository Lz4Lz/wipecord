package tui

import (
	"fmt"
	"time"

	"purge/internal/discord"
	"purge/internal/purge"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PurgeModel struct {
	err           error
	width, height int
	dmid          string
	Client        *discord.Client
	msgChan       chan tea.Msg

	Filters      []string
	SearchDelay  time.Duration
	DeleteDelay  time.Duration
	deletedCount int
	failedCount  int
	lastDeleted  string
	timeout      time.Duration
	status       string
	done         bool
}

func NewPurgeModel(DMID string, client *discord.Client) *PurgeModel {
	return &PurgeModel{
		dmid:         DMID,
		Client:       client,
		lastDeleted:  "null",
		deletedCount: 0,
		failedCount:  0,
		status:       "Idle",
		done:         false,
	}
}

func (m *PurgeModel) Init() tea.Cmd {
	return nil
}

func (m *PurgeModel) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		if m.msgChan == nil {
			return tea.Quit()
		}
		if msg, ok := <-m.msgChan; ok {
			return msg
		}
		return tea.Quit()
	}
}

func (m *PurgeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.msgChan != nil || m.done {
				return m, nil
			}

			m.msgChan = make(chan tea.Msg)
			m.status = "Starting purge..."

			go func() {

				purger, _ := purge.NewPurger(m.Client)

				if len(m.Filters) > 0 {
					purger.SetFilters(m.Filters)
				}

				if m.SearchDelay > 0 {
					purger.SetSearchDelay(m.SearchDelay)
				}

				if m.DeleteDelay > 0 {
					purger.SetDeleteDelay(m.DeleteDelay)
				}

				err := purger.Purge(m.dmid, func(u purge.Update) {
					m.msgChan <- u
				})

				if err != nil {
					m.msgChan <- errMsg(err)
				}
				close(m.msgChan)
			}()

			return m, m.waitForMsg()
		}

	case errMsg:
		m.err = msg
		m.done = true
		return m, tea.Quit

	case purge.Update:
		switch u := msg.(type) {

		case purge.UpdateDeleted:
			m.lastDeleted = truncate(u.Content, 50)
			m.deletedCount++

		case purge.UpdateFailed:
			m.failedCount++
			m.status = truncate(u.Message, 60)

		case purge.UpdateRateLimited:
			m.timeout = u.Timeout
			m.status = fmt.Sprintf("Rate limited. Waiting %s", u.Timeout)

		case purge.UpdateDone:
			m.done = true
			m.deletedCount = u.Deleted
			m.failedCount = u.Failed
			m.status = fmt.Sprintf(
				"Purge completed. Deleted: %d, Failed: %d, Throttled: %d",
				u.Deleted, u.Failed, u.Throttled)
		}

		return m, m.waitForMsg()
	}

	return m, nil
}

// Very ugly view, will make the TUI look better in future.
func (m *PurgeModel) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0AFF")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BA55D3"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)

	lines := []string{
		fmt.Sprintf("%s %s", labelStyle.Render("Deleted:"), valueStyle.Render(fmt.Sprintf("%d", m.deletedCount))),
		fmt.Sprintf("%s %s", labelStyle.Render("Failed:"), valueStyle.Render(fmt.Sprintf("%d", m.failedCount))),
		fmt.Sprintf("%s %s", labelStyle.Render("Last Msg:"), valueStyle.Render(truncate(m.lastDeleted, 40))),
		fmt.Sprintf("%s %s", labelStyle.Render("Delete Delay:"), valueStyle.Render(m.DeleteDelay.String())),
		fmt.Sprintf("%s %s", labelStyle.Render("Search Delay:"), valueStyle.Render(m.SearchDelay.String())),
		fmt.Sprintf("%s %s", labelStyle.Render("Timeout:"), valueStyle.Render(m.timeout.String())),
		fmt.Sprintf("%s %s", labelStyle.Render("Status:"), valueStyle.Render(truncate(m.status, 60))),
	}

	if m.done {
		lines = append(lines, labelStyle.Render("[Enter] to quit"))
	}

	if m.err != nil {
		lines = append(lines, errStyle.Render("Error: "+m.err.Error()))
	}

	statusBlock := lipgloss.JoinVertical(lipgloss.Left, lines...)

	container := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("129")).
		Render(statusBlock)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		container,
	)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
