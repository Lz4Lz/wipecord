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

	deletedCount int
	failedCount  int
	lastDeleted  string
	delay        time.Duration
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

func (m *PurgeModel) View() string {
	var status string

	if m.done {
		status = fmt.Sprintf(
			"Purge completed.\nDeleted: %d\nFailed: %d\nLast message: %s\nDelay: %s\nTimeout: %s\nStatus: %s\n\n[Esc]: quit",
			m.deletedCount,
			m.failedCount,
			m.lastDeleted,
			m.delay,
			m.timeout,
			m.status,
		)
	} else {
		status = fmt.Sprintf(
			"Purging...\nDeleted: %d\nFailed: %d\nLast message: %s\nDelay: %s\nTimeout: %s\nStatus: %s\n\n[Enter]: start | [Esc]: quit",
			m.deletedCount,
			m.failedCount,
			m.lastDeleted,
			m.delay,
			m.timeout,
			m.status,
		)
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFAA")).
		Bold(true).
		Padding(1, 2)

	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3333")).
			Bold(true)
		return style.Render(status) + "\n" + errStyle.Render("Error: "+m.err.Error())
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		style.Render(status),
	)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
