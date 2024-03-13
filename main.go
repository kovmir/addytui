package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	addy "github.com/kovmir/addyapi"
)

var (
	addyClient   = addy.NewClient(os.Getenv("ADDYTUI_TOKEN"))
	styleEnabled = lipgloss.NewStyle().Foreground(lipgloss.Color("#0f0"))
)

const (
	columnKeyAliasData = "alias_data"

	columnKeyEmail   = "email"
	columnKeyDesc    = "description"
	columnKeyFwdBlk  = "forwards_blocks"
	columnKeyReplSnd = "replies_sends"
)

type model struct {
	table   table.Model
	aliases []addy.Alias
}

func newModel() model {
	return model{
		table: table.
			New(generateColumns()).
			WithBaseStyle(lipgloss.NewStyle().Align(lipgloss.Left)).
			Focused(true).
			Filtered(true).
			WithRows([]table.Row{
				table.NewRow(table.RowData{
					columnKeyDesc:      "Fetching...",
					columnKeyAliasData: addy.Alias{Email: "..."},
				}),
			}),
	}
}

func (m model) Init() tea.Cmd {
	return fetchAliases
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "d":
			m.table = m.table.PageDown()
		case "u":
			m.table = m.table.PageUp()
		case "t":
			selRow := m.table.HighlightedRow()
			if len(selRow.Data) != 0 {
				selAlias := selRow.Data[columnKeyAliasData].(addy.Alias)
				cmds = append(cmds, m.toggleAlias(selAlias))
			}
		case "r":
			cmds = append(cmds, fetchAliases)
		}
	case []addy.Alias:
		m.aliases = msg
		m.table = m.table.
			WithColumns(generateColumns()).
			WithRows(generateRowsFromAliases(m.aliases))
	case addy.Alias:
		m.table = m.table.
			WithColumns(generateColumns()).
			WithRows(generateRowsFromAliases(m.aliases))
	case tea.WindowSizeMsg:
		m.table = m.table.WithPageSize(msg.Height - 6).
			WithTargetWidth(msg.Width)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.table.View()
}

// Toggles the "active" state of a given alias.
func (m *model) toggleAlias(alias addy.Alias) tea.Cmd {
	return func() tea.Msg {
		// Find the affected alias within the array.
		id := 0
		for i, v := range m.aliases {
			if v.ID == alias.ID {
				id = i
			}
		}
		// Toggle its state in API and local table.
		if alias.Active {
			err := addyClient.AliasDisable(alias.ID)
			if err != nil {
				panic(err)
			}
			m.aliases[id].Active = false
		} else {
			_, err := addyClient.AliasEnable(alias.ID)
			if err != nil {
				panic(err)
			}
			m.aliases[id].Active = true
		}
		// Return empty alias when done to re-draw the table.
		return addy.Alias{}
	}
}

// Fetches all aliases.
func fetchAliases() tea.Msg {
	res, err := addyClient.AliasesGet(&addy.AliasesGetArgs{})
	if err != nil {
		panic(err)
	}
	return res.Data
}

func generateColumns() []table.Column {
	return []table.Column{
		table.NewFlexColumn(columnKeyEmail, "E-Mail", 1).WithFiltered(true),
		table.NewFlexColumn(columnKeyDesc, "Description", 1).WithFiltered(true),
		table.NewColumn(columnKeyFwdBlk, "Forwaded/Blocked", 16),
		table.NewColumn(columnKeyReplSnd, "Replied/Sent", 16),
	}
}

func generateRowsFromAliases(aliases []addy.Alias) []table.Row {
	rows := []table.Row{}
	for _, alias := range aliases {
		row := table.NewRow(table.RowData{
			columnKeyAliasData: alias, // Invisible.

			columnKeyEmail:   alias.Email,
			columnKeyDesc:    alias.Description,
			columnKeyFwdBlk:  fmt.Sprintf("%d/%d", alias.EmailsBlocked, alias.EmailsForwarded),
			columnKeyReplSnd: fmt.Sprintf("%d/%d", alias.EmailsReplied, alias.EmailsSent),
		})
		if alias.Active {
			row = row.WithStyle(styleEnabled)
		}
		rows = append(rows, row)
	}
	return rows
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
