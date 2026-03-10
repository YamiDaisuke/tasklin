package tui

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
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
	viewBoard      viewMode = iota
	viewDetail              // ticket detail overlay
	viewMove                // pick target status
	viewNew                 // type new ticket title
	viewEdit                // edit ticket title
	viewHelp                // help overlay
	viewConfig              // config settings list
	viewConfigEdit          // editing a single config field
	viewStatuses            // manage statuses list
	viewStatusEdit          // editing name or color of a status
)

// cfgFieldDef describes one editable config field.
type cfgFieldDef struct {
	label string
	kind  string // "bool", "string", "int"
}

var configFields = []cfgFieldDef{
	{"Auto-commit on Done", "bool"},
	{"Default Done status", "string"},
	{"Title limit (0 = unlimited)", "int"},
	{"Manage statuses", "statuses"},
}

// Model is the Bubble Tea model.
type Model struct {
	store          *store.Store
	cfg            model.Config
	tickets        []model.Ticket // runtime (branch overrides applied)
	statuses       []model.Status // sorted
	colIdx         int            // focused column
	rowIdx         int            // focused row within column
	boardRowIdx    int            // rowIdx saved before entering move mode
	colScroll      []int          // per-column scroll offsets
	cfgRowIdx      int            // focused row in config screen
	statusRowIdx   int            // focused row in statuses screen
	statusEditStep int            // 0=name, 1=color
	statusEditNew  bool           // true when adding a new status
	statusTmpName  string         // holds name between step 0 and 1 when adding
	mode           viewMode
	inputBuf       string
	err            error
	branch         string
	projectDir     string
	width          int
	height         int
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
		colScroll:  make([]int, len(statuses)),
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

// commitDoneMsg is returned after an auto-commit attempt finishes.
type commitDoneMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case commitDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil
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
	case viewConfig, viewConfigEdit:
		return m.handleConfig(msg)
	case viewStatuses, viewStatusEdit:
		return m.handleStatuses(msg)
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
			m.clampScroll()
		}
	case "right", "l":
		if m.colIdx < len(cols)-1 {
			m.colIdx++
			m.rowIdx = 0
			m.clampScroll()
		}
	case "up", "k":
		if m.rowIdx > 0 {
			m.rowIdx--
			m.clampScroll()
		}
	case "down", "j":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if m.rowIdx < len(col)-1 {
			m.rowIdx++
			m.clampScroll()
		}
	case "enter":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 {
			m.mode = viewDetail
		} else {
			m.mode = viewNew
			m.inputBuf = ""
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
	case "shift+left":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 && m.colIdx > 0 {
			ticket := col[m.rowIdx]
			targetStatus := m.statuses[m.colIdx-1].Name
			m.boardRowIdx = m.rowIdx
			m.moveSelected(targetStatus)
			m.colIdx--
			m.rowIdx = 0
			m.clampScroll()
			return m, m.autoCommitCmd(ticket, targetStatus)
		}
	case "shift+right":
		col := m.ticketsInCol(cols[m.colIdx].Name)
		if len(col) > 0 && m.colIdx < len(cols)-1 {
			ticket := col[m.rowIdx]
			targetStatus := m.statuses[m.colIdx+1].Name
			m.boardRowIdx = m.rowIdx
			m.moveSelected(targetStatus)
			m.colIdx++
			m.rowIdx = 0
			m.clampScroll()
			return m, m.autoCommitCmd(ticket, targetStatus)
		}
	case "d":
		m.deleteSelected()
	case "c":
		m.mode = viewConfig
		m.cfgRowIdx = 0
	case "?":
		m.mode = viewHelp
	}
	return m, nil
}

func (m Model) handleConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Sub-mode: editing a field value.
	if m.mode == viewConfigEdit {
		switch msg.String() {
		case "esc":
			m.mode = viewConfig
			m.inputBuf = ""
		case "enter":
			val := strings.TrimSpace(m.inputBuf)
			switch m.cfgRowIdx {
			case 1: // DefaultDoneStatus
				if val != "" {
					m.cfg.DefaultDoneStatus = val
				}
			case 2: // TitleLimit
				if val == "" {
					m.cfg.TitleLimit = 0
				} else if n, err := strconv.Atoi(val); err == nil && n >= 0 {
					m.cfg.TitleLimit = n
				}
			}
			_ = m.store.WriteConfig(m.cfg)
			m.mode = viewConfig
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

	// viewConfig navigation.
	switch msg.String() {
	case "esc", "q", "c":
		m.mode = viewBoard
	case "up", "k":
		if m.cfgRowIdx > 0 {
			m.cfgRowIdx--
		}
	case "down", "j":
		if m.cfgRowIdx < len(configFields)-1 {
			m.cfgRowIdx++
		}
	case "enter", " ":
		f := configFields[m.cfgRowIdx]
		switch f.kind {
		case "bool":
			switch m.cfgRowIdx {
			case 0:
				m.cfg.AutoCommitOnDone = !m.cfg.AutoCommitOnDone
			}
			_ = m.store.WriteConfig(m.cfg)
		case "string":
			m.inputBuf = m.cfg.DefaultDoneStatus
			m.mode = viewConfigEdit
		case "int":
			if m.cfg.TitleLimit == 0 {
				m.inputBuf = ""
			} else {
				m.inputBuf = strconv.Itoa(m.cfg.TitleLimit)
			}
			m.mode = viewConfigEdit
		case "statuses":
			m.statusRowIdx = 0
			m.mode = viewStatuses
		}
	}
	return m, nil
}

func (m Model) handleStatuses(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Sub-mode: editing name or color of a status.
	if m.mode == viewStatusEdit {
		switch msg.String() {
		case "esc":
			m.mode = viewStatuses
			m.inputBuf = ""
		case "enter":
			val := strings.TrimSpace(m.inputBuf)
			if m.statusEditStep == 0 {
				if val == "" {
					return m, nil // name must not be empty
				}
				if m.statusEditNew {
					m.statusTmpName = val
					m.inputBuf = ""
				} else {
					m.updateStatusName(m.statusRowIdx, val)
					m.inputBuf = m.statuses[m.statusRowIdx].Color
				}
				m.statusEditStep = 1
			} else {
				if m.statusEditNew {
					m.addStatus(m.statusTmpName, val)
					m.statusRowIdx = len(m.statuses) - 1
				} else {
					m.updateStatusColor(m.statusRowIdx, val)
				}
				_ = m.store.WriteConfig(m.cfg)
				m.mode = viewStatuses
				m.inputBuf = ""
			}
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

	// viewStatuses navigation.
	switch msg.String() {
	case "esc", "q":
		m.mode = viewConfig
		if m.colIdx >= len(m.statuses) {
			m.colIdx = len(m.statuses) - 1
		}
	case "up", "k":
		if m.statusRowIdx > 0 {
			m.statusRowIdx--
		}
	case "down", "j":
		if m.statusRowIdx < len(m.statuses)-1 {
			m.statusRowIdx++
		}
	case "shift+up":
		if m.statusRowIdx > 0 {
			m.swapStatusOrder(m.statusRowIdx, m.statusRowIdx-1)
			m.statusRowIdx--
		}
	case "shift+down":
		if m.statusRowIdx < len(m.statuses)-1 {
			m.swapStatusOrder(m.statusRowIdx, m.statusRowIdx+1)
			m.statusRowIdx++
		}
	case "n":
		m.statusEditNew = true
		m.statusEditStep = 0
		m.inputBuf = ""
		m.mode = viewStatusEdit
	case "e":
		if len(m.statuses) > 0 {
			m.statusEditNew = false
			m.statusEditStep = 0
			m.inputBuf = m.statuses[m.statusRowIdx].Name
			m.mode = viewStatusEdit
		}
	case "d":
		if len(m.statuses) > 2 {
			m.deleteStatus(m.statusRowIdx)
			if m.statusRowIdx >= len(m.statuses) {
				m.statusRowIdx = len(m.statuses) - 1
			}
		}
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
		col := m.ticketsInCol(m.statuses[m.colIdx].Name)
		var ticket model.Ticket
		if len(col) > 0 {
			ticket = col[m.boardRowIdx]
		}
		m.rowIdx = m.boardRowIdx
		m.moveSelected(targetStatus)
		m.mode = viewBoard
		m.rowIdx = 0
		return m, m.autoCommitCmd(ticket, targetStatus)
	}
	return m, nil
}

// autoCommitCmd returns a tea.Cmd that suspends the TUI, runs an interactive
// git add -p, then commits if anything was staged. Returns nil if the feature
// is disabled or conditions are not met.
func (m Model) autoCommitCmd(ticket model.Ticket, targetStatus string) tea.Cmd {
	if !m.cfg.AutoCommitOnDone || targetStatus != m.cfg.DefaultDoneStatus {
		return nil
	}
	gitRoot := internalgit.RepoRoot(m.projectDir)
	if gitRoot == "" {
		return nil
	}
	commitMsg := fmt.Sprintf("[%d] %s", ticket.ID, ticket.Title)
	cmd := exec.Command("sh", "-c",
		`cd "$GIT_ROOT" && git add -p; git diff --cached --quiet || git commit -m "$COMMIT_MSG"`)
	cmd.Env = append(os.Environ(), "GIT_ROOT="+gitRoot, "COMMIT_MSG="+commitMsg)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return commitDoneMsg{err: err}
	})
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

func (m *Model) addStatus(name, color string) {
	maxID, maxOrder := 0, 0
	for _, s := range m.cfg.Statuses {
		if s.ID > maxID {
			maxID = s.ID
		}
		if s.Order > maxOrder {
			maxOrder = s.Order
		}
	}
	m.cfg.Statuses = append(m.cfg.Statuses, model.Status{
		ID:    maxID + 1,
		Name:  name,
		Color: color,
		Order: maxOrder + 1,
	})
	m.statuses = store.SortedStatuses(m.cfg.Statuses)
	m.colScroll = make([]int, len(m.statuses))
}

func (m *Model) updateStatusName(idx int, name string) {
	old := m.statuses[idx].Name
	for k := range m.cfg.Statuses {
		if m.cfg.Statuses[k].Name == old {
			m.cfg.Statuses[k].Name = name
			break
		}
	}
	// Migrate tickets that reference the old status name.
	for k := range m.tickets {
		if m.tickets[k].Status == old {
			m.tickets[k].Status = name
		}
	}
	m.statuses = store.SortedStatuses(m.cfg.Statuses)
	m.persist()
}

func (m *Model) updateStatusColor(idx int, color string) {
	name := m.statuses[idx].Name
	for k := range m.cfg.Statuses {
		if m.cfg.Statuses[k].Name == name {
			m.cfg.Statuses[k].Color = color
			break
		}
	}
	m.statuses = store.SortedStatuses(m.cfg.Statuses)
}

func (m *Model) deleteStatus(idx int) {
	if len(m.statuses) <= 2 {
		return
	}
	name := m.statuses[idx].Name
	out := make([]model.Status, 0, len(m.cfg.Statuses)-1)
	for _, s := range m.cfg.Statuses {
		if s.Name != name {
			out = append(out, s)
		}
	}
	m.cfg.Statuses = out
	m.statuses = store.SortedStatuses(m.cfg.Statuses)
	m.colScroll = make([]int, len(m.statuses))
	_ = m.store.WriteConfig(m.cfg)
}

func (m *Model) swapStatusOrder(i, j int) {
	nameI, nameJ := m.statuses[i].Name, m.statuses[j].Name
	orderI, orderJ := m.statuses[i].Order, m.statuses[j].Order
	for k := range m.cfg.Statuses {
		switch m.cfg.Statuses[k].Name {
		case nameI:
			m.cfg.Statuses[k].Order = orderJ
		case nameJ:
			m.cfg.Statuses[k].Order = orderI
		}
	}
	m.statuses = store.SortedStatuses(m.cfg.Statuses)
	_ = m.store.WriteConfig(m.cfg)
}

// --- helpers ---

// ticketRows returns the number of visible ticket rows on the board.
func (m Model) ticketRows() int {
	rows := m.height - 6
	if rows < 1 {
		rows = 1
	}
	return rows
}

// clampScroll adjusts colScroll[colIdx] so that rowIdx stays visible.
func (m *Model) clampScroll() {
	if m.colIdx >= len(m.colScroll) {
		return
	}
	vis := m.ticketRows()
	scroll := m.colScroll[m.colIdx]
	if m.rowIdx < scroll {
		scroll = m.rowIdx
	} else if m.rowIdx >= scroll+vis {
		scroll = m.rowIdx - vis + 1
	}
	m.colScroll[m.colIdx] = scroll
}

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
	case viewConfig, viewConfigEdit:
		return m.viewConfigScreen()
	case viewStatuses, viewStatusEdit:
		return m.viewStatusesScreen()
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
	// Layout rows: 3 app-header + 1 col-name + 1 divider + ticketRows + 1 footer
	ticketRows := m.ticketRows()
	colHeight := 2 + ticketRows // col-name line + divider line + ticket lines

	// Build each column as a slice of pre-rendered lines.
	colLines := make([][]string, n)
	for ci, st := range m.statuses {
		colWidth := baseColWidth
		if ci == n-1 {
			colWidth += remainder
		}

		tickets := m.ticketsInCol(st.Name)

		statusColor := ansiColor(st.Color)
		focused := ci == m.colIdx && m.mode == viewBoard

		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(statusColor).
			Width(colWidth)
		if focused {
			headerStyle = headerStyle.Underline(true)
		}
		header := headerStyle.Render(fmt.Sprintf(" %s (%d)", strings.ToUpper(st.Name), len(tickets)))

		divider := lipgloss.NewStyle().Foreground(statusColor).Render(strings.Repeat("─", colWidth))

		lines := make([]string, 0, colHeight)
		lines = append(lines, header, divider)

		// Determine scroll offset for this column.
		scrollOffset := 0
		if ci < len(m.colScroll) {
			scrollOffset = m.colScroll[ci]
		}

		// Scrollbar geometry — only shown when tickets overflow the viewport.
		needsBar := len(tickets) > ticketRows
		var thumbTop, thumbSize int
		if needsBar {
			thumbSize = ticketRows * ticketRows / len(tickets)
			if thumbSize < 1 {
				thumbSize = 1
			}
			maxScroll := len(tickets) - ticketRows
			if maxScroll > 0 {
				thumbTop = scrollOffset * (ticketRows - thumbSize) / maxScroll
			}
			if thumbTop+thumbSize > ticketRows {
				thumbTop = ticketRows - thumbSize
			}
		}
		barChar := func(ri int) string {
			if !needsBar {
				return ""
			}
			if ri >= thumbTop && ri < thumbTop+thumbSize {
				return lipgloss.NewStyle().Foreground(statusColor).Render("┃")
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("╎")
		}

		// Content width shrinks by 1 when a scrollbar is present.
		contentWidth := colWidth
		if needsBar {
			contentWidth = colWidth - 1
		}

		for ri := 0; ri < ticketRows; ri++ {
			ti := scrollOffset + ri // absolute ticket index
			bar := barChar(ri)
			if ti < len(tickets) {
				t := tickets[ti]
				title := truncate(fmt.Sprintf("[%d] %s", t.ID, t.Title), contentWidth-3)
				if focused && ti == m.rowIdx {
					indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("▌")
					text := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Width(contentWidth - 1).Render(" " + title)
					lines = append(lines, indicator+text+bar)
				} else {
					style := lipgloss.NewStyle().Width(contentWidth).PaddingLeft(2).Foreground(lipgloss.Color("252"))
					lines = append(lines, style.Render(title)+bar)
				}
			} else if ri == 0 && len(tickets) == 0 && focused {
				// Placeholder row for empty focused column.
				style := lipgloss.NewStyle().Width(contentWidth).PaddingLeft(2).
					Foreground(lipgloss.Color("240")).Italic(true)
				lines = append(lines, style.Render("New ticket...")+bar)
			} else {
				lines = append(lines, lipgloss.NewStyle().Width(contentWidth).Render("")+bar)
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
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("│"))
			}
			if row < len(lines) {
				sb.WriteString(lines[row])
			}
		}
		boardLines[row] = sb.String()
	}
	board := strings.Join(boardLines, "\n")

	const (
		titleLine1 = "╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔"
		titleLine2 = " ║ ╠═╣╚═╗╠╩╗║  ║║║║"
		titleLine3 = " ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝"
	)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	barStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Width(m.width)
	meta := ""
	if m.branch != "" {
		meta += "⎇ " + m.branch
	}
	headerLine := barStyle.Render(" "+accentStyle.Render(titleLine1)) + "\n" +
		barStyle.Render(" "+accentStyle.Render(titleLine2)) + "\n" +
		barStyle.Render(" "+accentStyle.Render(titleLine3)+"  "+dimStyle.Render(meta))

	var footerContent string
	if m.err != nil {
		footerContent = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render(" error: " + m.err.Error())
	} else {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
		sepStr := dimStyle.Render("  │  ")
		type hint struct{ key, label string }
		hints := []hint{{"n", "new"}, {"d", "del"}, {"m", "move"}, {"e", "edit"}, {"c", "config"}, {"?", "help"}, {"q", "quit"}}
		parts := make([]string, len(hints))
		for i, h := range hints {
			parts[i] = keyStyle.Render(h.key) + " " + h.label
		}
		footerContent = " " + strings.Join(parts, sepStr)
	}
	footerLine := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250")).
		Width(m.width).
		Render(footerContent)

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

func (m Model) viewStatusesScreen() string {
	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true)
	fmt.Fprintln(&sb, titleStyle.Render("Statuses")+"  (Esc to go back)\n")

	if m.mode == viewStatusEdit {
		var label string
		if m.statusEditStep == 0 {
			if m.statusEditNew {
				label = "New status name: "
			} else {
				label = fmt.Sprintf("Rename %q: ", m.statuses[m.statusRowIdx].Name)
			}
		} else {
			name := m.statusTmpName
			if !m.statusEditNew {
				name = m.statuses[m.statusRowIdx].Name
			}
			label = fmt.Sprintf("Color for %q (ANSI name or code): ", name)
		}
		fmt.Fprintln(&sb, label+m.inputBuf+"█")
		fmt.Fprintln(&sb, "\n[Enter] confirm  [Esc] cancel")
		return sb.String()
	}

	selectedStyle := lipgloss.NewStyle().Reverse(true)
	for i, st := range m.statuses {
		swatch := lipgloss.NewStyle().Foreground(ansiColor(st.Color)).Render("■")
		line := fmt.Sprintf("  %-22s %s %-10s  order %d", st.Name, swatch, st.Color, st.Order)
		if i == m.statusRowIdx {
			line = selectedStyle.Render(fmt.Sprintf("  %-22s %-12s  order %d", st.Name, st.Color, st.Order))
		}
		fmt.Fprintln(&sb, line)
	}

	deleteHint := "[d]elete"
	if len(m.statuses) <= 2 {
		deleteHint = "(min 2, can't delete)"
	}
	fmt.Fprintf(&sb, "\n[n]ew  [e]dit  %s  Shift+[↑/↓] reorder  [Esc] back\n", deleteHint)
	return sb.String()
}

func (m Model) viewConfigScreen() string {
	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true)
	fmt.Fprintln(&sb, titleStyle.Render("Configuration")+"  (Esc to save & close)\n")

	selectedStyle := lipgloss.NewStyle().Reverse(true)
	for i, f := range configFields {
		var val string
		switch i {
		case 0:
			if m.cfg.AutoCommitOnDone {
				val = "on"
			} else {
				val = "off"
			}
		case 1:
			val = m.cfg.DefaultDoneStatus
		case 2:
			if m.cfg.TitleLimit == 0 {
				val = "unlimited"
			} else {
				val = strconv.Itoa(m.cfg.TitleLimit)
			}
		case 3:
			val = fmt.Sprintf("%d configured →", len(m.statuses))
		}

		if m.mode == viewConfigEdit && i == m.cfgRowIdx {
			val = m.inputBuf + "█"
		}

		line := fmt.Sprintf("  %-30s %s", f.label, val)
		if i == m.cfgRowIdx {
			line = selectedStyle.Render(line)
		}
		fmt.Fprintln(&sb, line)
	}

	fmt.Fprintln(&sb, "\n[↑/↓] navigate  [Enter/Space] toggle/edit  [Esc] save & close")
	return sb.String()
}

func (m Model) viewHelpOverlay() string {
	return `Keyboard shortcuts:

  ← / → or h / l   Move between columns
  ↑ / ↓ or k / j   Move between tickets
  Shift+← / →      Move selected ticket to adjacent column
  Enter             View ticket detail
  n                 New ticket
  m                 Move ticket to another status
  e                 Edit ticket title
  d                 Delete ticket
  c                 Open config settings
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
