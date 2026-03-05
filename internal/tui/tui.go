package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	internalgit "github.com/frankcruz/tasklin/internal/git"
	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
)

// view modes
type viewMode int

const (
	viewBoard  viewMode = iota
	viewDetail          // ticket detail overlay
	viewMove            // pick target status
	viewNew             // type new ticket title
	viewEdit            // edit ticket title
	viewHelp            // help overlay
)

// Model is the Bubble Tea model.
type Model struct {
	store      *store.Store
	cfg        model.Config
	tickets    []model.Ticket // runtime (branch overrides applied)
	statuses   []model.Status // sorted
	colIdx        int      // focused column
	rowIdx        int      // focused row within column
	boardRowIdx   int      // rowIdx saved before entering move mode
	mode          viewMode
	inputBuf   string
	err        error
	branch     string
	projectDir string
	width      int
	height     int
}

// New creates a TUI model for the given store, applying branch overrides.
func New(s *store.Store, projectDir string) (Model, error) {
	cfg, err := s.ReadConfig()
	if err != nil {
		return Model{}, err
	}
	tickets, err := s.ReadTickets()
	if err != nil {
		return Model{}, err
	}

	branch := internalgit.CurrentBranch(projectDir)
	if branch != "" && !internalgit.IsMainBranch(branch) {
		gs, err := store.ReadGlobalState()
		if err == nil {
			overrides := store.GetBranchOverrides(gs, projectDir, branch)
			tickets = store.ApplyBranchOverrides(tickets, overrides)
		}
	}

	statuses := store.SortedStatuses(cfg.Statuses)
	return Model{
		store:      s,
		cfg:        cfg,
		tickets:    tickets,
		statuses:   statuses,
		branch:     branch,
		projectDir: projectDir,
		width:      80,
		height:     24,
	}, nil
}

// Run starts the Bubble Tea program.
func Run(s *store.Store, projectDir string) error {
	m, err := New(s, projectDir)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// --- Bubble Tea interface ---

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewNew, viewEdit:
		return m.handleInput(msg)
	case viewMove:
		return m.handleMove(msg)
	case viewDetail:
		return m.handleDetail(msg)
	case viewHelp:
		m.mode = viewBoard
		return m, nil
	default:
		return m.handleBoard(msg)
	}
}

func (m Model) handleBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cols := m.statuses
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "left", "h":
		if m.colIdx > 0 {
			m.colIdx--
			m.rowIdx = 0
		}
	case "right", "l":
		if m.colIdx < len(cols)-1 {
			m.colIdx++
			m.rowIdx = 0
		}
	case "up", "k":
		if m.rowIdx > 0 {
			m.rowIdx--
		}
	case "down", "j":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if m.rowIdx < len(col)-1 {
			m.rowIdx++
		}
	case "enter":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 {
			m.mode = viewDetail
		}
	case "n":
		m.mode = viewNew
		m.inputBuf = ""
	case "e":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 {
			m.mode = viewEdit
			m.inputBuf = col[m.rowIdx].Title
		}
	case "m":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 {
			m.boardRowIdx = m.rowIdx
			m.mode = viewMove
			m.rowIdx = 0
		}
	case "d":
		m.deleteSelected()
	case "?":
		m.mode = viewHelp
	}
	return m, nil
}

func (m Model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewBoard
		m.inputBuf = ""
	case "enter":
		title := strings.TrimSpace(m.inputBuf)
		if title != "" {
			if m.cfg.TitleLimit > 0 && utf8.RuneCountInString(title) > m.cfg.TitleLimit {
				title = string([]rune(title)[:m.cfg.TitleLimit])
			}
			if m.mode == viewNew {
				m.addTicket(title)
			} else {
				m.editTicket(title)
			}
		}
		m.mode = viewBoard
		m.inputBuf = ""
	case "backspace":
		r := []rune(m.inputBuf)
		if len(r) > 0 {
			m.inputBuf = string(r[:len(r)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			m.inputBuf += string(msg.Runes)
		}
	}
	return m, nil
}

func (m Model) handleMove(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.rowIdx = m.boardRowIdx
		m.mode = viewBoard
	case "up", "k":
		if m.rowIdx > 0 {
			m.rowIdx--
		}
	case "down", "j":
		if m.rowIdx < len(m.statuses)-1 {
			m.rowIdx++
		}
	case "enter":
		targetStatus := m.statuses[m.rowIdx].Name
		m.rowIdx = m.boardRowIdx
		m.moveSelected(targetStatus)
		m.mode = viewBoard
		m.rowIdx = 0
	}
	return m, nil
}

func (m Model) handleDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "q" {
		m.mode = viewBoard
	}
	return m, nil
}

// --- data mutations ---

func (m *Model) addTicket(title string) {
	id, err := m.store.NextID()
	if err != nil {
		m.err = err
		return
	}
	statusName := m.statuses[m.colIdx].Name
	t := model.Ticket{
		ID:        id,
		Title:     title,
		Status:    statusName,
		CreatedAt: time.Now().UTC(),
	}
	m.tickets = append(m.tickets, t)
	m.persist()
}

func (m *Model) editTicket(title string) {
	col := m.ticketsInCol(m.statuses[m.colIdx].Name)
	if len(col) == 0 {
		return
	}
	selected := col[m.rowIdx]
	for i, t := range m.tickets {
		if t.ID == selected.ID {
			m.tickets[i].Title = title
			break
		}
	}
	m.persist()
}

func (m *Model) moveSelected(targetStatus string) {
	col := m.ticketsInCol(m.statuses[m.colIdx].Name)
	if len(col) == 0 {
		return
	}
	selected := col[m.boardRowIdx]
	for i, t := range m.tickets {
		if t.ID == selected.ID {
			tr := model.Transition{From: t.Status, To: targetStatus, At: time.Now().UTC()}
			m.tickets[i].Status = targetStatus
			m.tickets[i].Transitions = append(m.tickets[i].Transitions, tr)
			break
		}
	}

	// Branch tracking.
	branch := internalgit.CurrentBranch(m.projectDir)
	if branch != "" && !internalgit.IsMainBranch(branch) {
		gs, err := store.ReadGlobalState()
		if err == nil {
			store.SetBranchOverride(&gs, m.projectDir, branch, selected.ID, targetStatus)
			_ = store.WriteGlobalState(gs)
		}
		// Don't write to tickets.yaml on non-main branches.
		return
	}
	m.persist()
}

func (m *Model) deleteSelected() {
	col := m.ticketsInCol(m.statuses[m.colIdx].Name)
	if len(col) == 0 {
		return
	}
	selected := col[m.rowIdx]
	deleted, _ := m.store.ReadDeleted()
	deleted = append(deleted, selected)
	_ = m.store.WriteDeleted(deleted)

	newTickets := make([]model.Ticket, 0, len(m.tickets)-1)
	for _, t := range m.tickets {
		if t.ID != selected.ID {
			newTickets = append(newTickets, t)
		}
	}
	m.tickets = newTickets
	if m.rowIdx > 0 {
		m.rowIdx--
	}
	m.persist()
}

func (m *Model) persist() {
	// Only write tickets that don't have branch overrides pending.
	_ = m.store.WriteTickets(m.tickets)
}

// --- helpers ---

func (m Model) ticketsInCol(statusName string) []model.Ticket {
	var result []model.Ticket
	for _, t := range m.tickets {
		if t.Status == statusName {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (m Model) selectedTicket() *model.Ticket {
	if m.colIdx >= len(m.statuses) {
		return nil
	}
	col := m.ticketsInCol(m.statuses[m.colIdx].Name)
	if m.rowIdx >= len(col) {
		return nil
	}
	t := col[m.rowIdx]
	return &t
}

// --- View ---

func (m Model) View() string {
	switch m.mode {
	case viewDetail:
		return m.viewDetail()
	case viewMove:
		return m.viewMoveMenu()
	case viewNew:
		return m.viewBoard() + "\n" + m.viewInputBar("New ticket title: ")
	case viewEdit:
		return m.viewBoard() + "\n" + m.viewInputBar("Edit title: ")
	case viewHelp:
		return m.viewHelpOverlay()
	default:
		return m.viewBoard()
	}
}

func (m Model) viewBoard() string {
	n := len(m.statuses)
	if n == 0 {
		return "No statuses configured."
	}

	// Divide terminal width evenly; last column absorbs any remainder so the
	// total always equals m.width exactly (1 '│' separator between each col).
	sepTotal := n - 1
	baseColWidth := (m.width - sepTotal) / n
	if baseColWidth < 4 {
		baseColWidth = 4
	}
	remainder := m.width - sepTotal - baseColWidth*n

	// Ticket rows fill everything between the header/col-name/divider and footer.
	// Layout rows: 1 app-header + 1 col-name + 1 divider + ticketRows + 1 footer
	ticketRows := m.height - 4
	if ticketRows < 1 {
		ticketRows = 1
	}
	colHeight := 2 + ticketRows // col-name line + divider line + ticket lines

	// Build each column as a slice of pre-rendered lines.
	colLines := make([][]string, n)
	for ci, st := range m.statuses {
		colWidth := baseColWidth
		if ci == n-1 {
			colWidth += remainder
		}

		tickets := m.ticketsInCol(st.Name)

		header := lipgloss.NewStyle().
			Bold(true).
			Foreground(ansiColor(st.Color)).
			Width(colWidth).
			Render(fmt.Sprintf(" %s (%d)", strings.ToUpper(st.Name), len(tickets)))

		divider := strings.Repeat("─", colWidth)

		lines := make([]string, 0, colHeight)
		lines = append(lines, header, divider)

		for ri := 0; ri < ticketRows; ri++ {
			if ri < len(tickets) {
				t := tickets[ri]
				title := truncate(fmt.Sprintf("[%d] %s", t.ID, t.Title), colWidth-2)
				style := lipgloss.NewStyle().Width(colWidth).PaddingLeft(1)
				if ci == m.colIdx && ri == m.rowIdx && m.mode == viewBoard {
					style = style.Reverse(true)
				}
				lines = append(lines, style.Render(title))
			} else {
				lines = append(lines, lipgloss.NewStyle().Width(colWidth).Render(""))
			}
		}
		colLines[ci] = lines
	}

	// Assemble the board row-by-row so the '│' separator spans every line.
	boardLines := make([]string, colHeight)
	for row := 0; row < colHeight; row++ {
		var sb strings.Builder
		for ci, lines := range colLines {
			if ci > 0 {
				sb.WriteString("│")
			}
			if row < len(lines) {
				sb.WriteString(lines[row])
			}
		}
		boardLines[row] = sb.String()
	}
	board := strings.Join(boardLines, "\n")

	projectName := filepath.Base(m.projectDir)
	branchInfo := ""
	if m.branch != "" {
		branchInfo = "  branch: " + m.branch
	}
	headerLine := lipgloss.NewStyle().Bold(true).Width(m.width).Render(
		fmt.Sprintf(" %s backlog%s", projectName, branchInfo),
	)

	footer := " [n]ew  [d]elete  [m]ove  [e]dit  [?]help  [q]uit"
	if m.err != nil {
		footer = " Error: " + m.err.Error()
	}
	footerLine := lipgloss.NewStyle().Width(m.width).Render(footer)

	return headerLine + "\n" + board + "\n" + footerLine
}

func (m Model) viewDetail() string {
	t := m.selectedTicket()
	if t == nil {
		return "No ticket selected. Press Esc."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Ticket #%d\n", t.ID)
	fmt.Fprintf(&sb, "Title:   %s\n", t.Title)
	fmt.Fprintf(&sb, "Status:  %s\n", t.Status)
	fmt.Fprintf(&sb, "Created: %s\n\n", t.CreatedAt.Format("2006-01-02 15:04"))
	if len(t.Transitions) == 0 {
		fmt.Fprintln(&sb, "No transitions yet.")
	} else {
		fmt.Fprintln(&sb, "Transitions:")
		for _, tr := range t.Transitions {
			fmt.Fprintf(&sb, "  %s → %s  (%s)\n", tr.From, tr.To, tr.At.Format("2006-01-02 15:04"))
		}
	}
	fmt.Fprintln(&sb, "\nPress Esc to go back.")
	return sb.String()
}

func (m Model) viewMoveMenu() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, "Move ticket to:")
	for i, st := range m.statuses {
		prefix := "  "
		if i == m.rowIdx {
			prefix = "> "
		}
		fmt.Fprintf(&sb, "%s%s\n", prefix, st.Name)
	}
	fmt.Fprintln(&sb, "\n[↑/↓] select  [Enter] confirm  [Esc] cancel")
	return sb.String()
}

func (m Model) viewInputBar(label string) string {
	return label + m.inputBuf + "█"
}

func (m Model) viewHelpOverlay() string {
	return `Keyboard shortcuts:

  ← / → or h / l   Move between columns
  ↑ / ↓ or k / j   Move between tickets
  Enter             View ticket detail
  n                 New ticket
  m                 Move ticket to another status
  e                 Edit ticket title
  d                 Delete ticket
  ?                 This help
  q / Ctrl+C        Quit

Press any key to close.`
}

// --- style helpers ---

func ansiColor(name string) lipgloss.Color {
	switch strings.ToLower(name) {
	case "red":
		return lipgloss.Color("1")
	case "green":
		return lipgloss.Color("2")
	case "yellow":
		return lipgloss.Color("3")
	case "blue":
		return lipgloss.Color("4")
	case "magenta":
		return lipgloss.Color("5")
	case "cyan":
		return lipgloss.Color("6")
	case "white":
		return lipgloss.Color("7")
	default:
		return lipgloss.Color(name)
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}


func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// filepath shim — avoid full import just for Base.
var filepath = pathHelper{}

type pathHelper struct{}

func (pathHelper) Base(p string) string {
	// simple last-segment extraction
	p = strings.TrimRight(p, "/")
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		return p[idx+1:]
	}
	return p
}

// ColIdx returns the currently focused column index (exported for testing).
func (m Model) ColIdx() int { return m.colIdx }

// RowIdx returns the currently focused row index (exported for testing).
func (m Model) RowIdx() int { return m.rowIdx }

// Width returns the terminal width tracked by the model.
func (m Model) Width() int { return m.width }

// Height returns the terminal height tracked by the model.
func (m Model) Height() int { return m.height }

// ensure os import used
var _ = os.Stderr
