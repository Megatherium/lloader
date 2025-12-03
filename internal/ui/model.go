package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lloader/internal/app"
	"lloader/internal/process"

	hfmodels "github.com/Megatherium/hf-go"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
	"golang.org/x/term"
)

// OutputMsg is a message type for output from processes
type OutputMsg struct {
	Output string
}

// CheckOutputMsg is a message to check for new output
type CheckOutputMsg struct{}

// InitMsg is a message to indicate initialization is complete
type InitMsg struct{}

// HFSearchResultMsg contains search results from HuggingFace
type HFSearchResultMsg struct {
	Models []hfmodels.Model
	Err    error
}

// HFQuantsResultMsg contains available quantizations for a model
type HFQuantsResultMsg struct {
	ModelID string
	Quants  []string
	Err     error
}

// HFModelDetailsMsg contains detailed model information
type HFModelDetailsMsg struct {
	Details *hfmodels.ModelDetails
	Err     error
}

// Model represents the application state
type Model struct {
	models       []string
	selected     int
	output       string
	quit         bool
	processMgr   *process.ProcessManager
	focusRight   bool
	outputChan   chan string
	logger       *zap.Logger
	config       *app.Config
	windowWidth  int
	windowHeight int
	scrollOffset int

	// Session overrides (reset each run)
	sessionNGL     int
	sessionCtxSize int

	// Modal state
	showModal     bool
	modalFocusIdx int // 0 = ngl, 1 = ctx-size
	nglInput      textinput.Model
	ctxSizeInput  textinput.Model

	// CLI input mode
	cliInputBuffer string
	cliMode        bool

	// Tab state: 0 = Local, 1 = HuggingFace
	activeTab int

	// HuggingFace search state
	hfSearchInput   textinput.Model
	hfSearchFocused bool
	hfModels        []hfmodels.Model
	hfSelected      int
	hfSearching     bool
	hfClient        *hfmodels.Client

	// Quantization selection modal
	showQuantModal  bool
	quantSelected   int
	selectedHFModel *hfmodels.Model
	availableQuants []string
	loadingQuants   bool

	// Model info modal
	showInfoModal  bool
	modelDetails   *hfmodels.ModelDetails
	loadingDetails bool

	// No quants confirmation modal
	showNoQuantModal bool
}

// NewModel creates a new model
func NewModel(models []string, config *app.Config, logger *zap.Logger) *Model {
	pm := process.NewProcessManager(logger)
	pm.SetTemplates(config.ServerTemplate, config.CLITemplate)

	nglInput := textinput.New()
	nglInput.Placeholder = "99"
	nglInput.CharLimit = 5
	nglInput.Width = 10
	nglInput.SetValue(fmt.Sprintf("%d", config.DefaultNGL))

	ctxInput := textinput.New()
	ctxInput.Placeholder = "0"
	ctxInput.CharLimit = 10
	ctxInput.Width = 10
	ctxInput.SetValue(fmt.Sprintf("%d", config.DefaultCtxSize))

	hfSearch := textinput.New()
	hfSearch.Placeholder = "Search HuggingFace models..."
	hfSearch.CharLimit = 100
	hfSearch.Width = 30

	return &Model{
		models:         models,
		selected:       0,
		output:         "Ready. Select a model and press Enter for server, c for cli, e for config.\nPress 1/2 to switch tabs. In HF tab, press / to search.",
		outputChan:     make(chan string, 100),
		processMgr:     pm,
		logger:         logger,
		config:         config,
		sessionNGL:     config.DefaultNGL,
		sessionCtxSize: config.DefaultCtxSize,
		nglInput:       nglInput,
		ctxSizeInput:   ctxInput,
		hfSearchInput:  hfSearch,
		hfClient:       hfmodels.NewClient(""),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return InitMsg{}
		},
		m.checkOutputCmd(),
	)
}

// checkOutputCmd creates a command to periodically check for output
func (m *Model) checkOutputCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return CheckOutputMsg{}
	})
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle modals first
		if m.showModal {
			return m.updateModal(msg)
		}
		if m.showQuantModal {
			return m.updateQuantModal(msg)
		}
		if m.showInfoModal {
			return m.updateInfoModal(msg)
		}
		if m.showNoQuantModal {
			return m.updateNoQuantModal(msg)
		}

		// Handle HF search input mode
		if m.hfSearchFocused {
			return m.updateHFSearch(msg)
		}

		// Handle CLI input mode
		if m.cliMode && m.focusRight && m.processMgr.IsRunning() {
			return m.updateCliInput(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			m.processMgr.Stop()
			return m, tea.Quit
		case "1":
			m.activeTab = 0
		case "2":
			m.activeTab = 1
		case "/":
			if m.activeTab == 1 && !m.focusRight {
				m.hfSearchFocused = true
				m.hfSearchInput.Focus()
				return m, nil
			}
		case "up":
			if m.focusRight {
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			} else if m.activeTab == 0 {
				m.selected--
				if m.selected < 0 {
					m.selected = len(m.models) - 1
				}
			} else if m.activeTab == 1 && len(m.hfModels) > 0 {
				m.hfSelected--
				if m.hfSelected < 0 {
					m.hfSelected = len(m.hfModels) - 1
				}
			}
		case "down":
			if m.focusRight {
				m.scrollOffset++
			} else if m.activeTab == 0 {
				m.selected = (m.selected + 1) % len(m.models)
			} else if m.activeTab == 1 && len(m.hfModels) > 0 {
				m.hfSelected = (m.hfSelected + 1) % len(m.hfModels)
			}
		case "enter":
			if m.activeTab == 0 {
				m.output += "Enter key pressed - starting server\n"
				m.startServer()
			} else if m.activeTab == 1 && len(m.hfModels) > 0 {
				m.selectedHFModel = &m.hfModels[m.hfSelected]
				m.loadingQuants = true
				m.availableQuants = nil
				m.output += fmt.Sprintf("Fetching available quantizations for %s...\n", m.selectedHFModel.ID)
				return m, m.fetchQuants(m.selectedHFModel.ID)
			}
		case "c":
			if m.activeTab == 0 {
				m.startCli()
			} else if m.activeTab == 1 && len(m.hfModels) > 0 {
				m.selectedHFModel = &m.hfModels[m.hfSelected]
				m.loadingQuants = true
				m.availableQuants = nil
				m.output += fmt.Sprintf("Fetching available quantizations for %s...\n", m.selectedHFModel.ID)
				return m, m.fetchQuants(m.selectedHFModel.ID)
			}
		case "i":
			if m.activeTab == 1 && len(m.hfModels) > 0 {
				model := m.hfModels[m.hfSelected]
				m.loadingDetails = true
				m.output += fmt.Sprintf("Fetching details for %s...\n", model.ID)
				return m, m.fetchModelDetails(model.ID)
			}
		case "e":
			m.showModal = true
			m.modalFocusIdx = 0
			m.nglInput.Focus()
			m.ctxSizeInput.Blur()
		case "tab":
			m.focusRight = !m.focusRight
		case "ctrl+l":
			m.output = ""
			m.scrollOffset = 0
		default:
			if m.focusRight && m.processMgr.IsRunning() {
				m.logger.Debug("Key pressed", zap.String("key", msg.String()))
			}
		}
	case HFSearchResultMsg:
		m.hfSearching = false
		if msg.Err != nil {
			m.output += fmt.Sprintf("HF search error: %v\n", msg.Err)
		} else {
			m.hfModels = msg.Models
			m.hfSelected = 0
			m.output += fmt.Sprintf("Found %d models\n", len(msg.Models))
		}
	case HFQuantsResultMsg:
		m.loadingQuants = false
		if msg.Err != nil {
			m.output += fmt.Sprintf("Error fetching quants: %v\n", msg.Err)
		} else if len(msg.Quants) == 0 {
			m.showNoQuantModal = true
			m.output += "No quantizations found for this model\n"
		} else {
			m.availableQuants = msg.Quants
			m.quantSelected = 0
			m.showQuantModal = true
			m.output += fmt.Sprintf("Found %d quantizations\n", len(msg.Quants))
		}
	case HFModelDetailsMsg:
		m.loadingDetails = false
		if msg.Err != nil {
			m.output += fmt.Sprintf("Error fetching details: %v\n", msg.Err)
		} else {
			m.modelDetails = msg.Details
			m.showInfoModal = true
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.scrollOffset = 0
		return m, nil
	case InitMsg:
		m.output += "Init completed - starting output monitoring\n"
		return m, nil
	case CheckOutputMsg:
		select {
		case output := <-m.outputChan:
			m.output += output
			m.scrollOffset = len(strings.Split(m.output, "\n"))
			return m, m.checkOutputCmd()
		default:
			return m, m.checkOutputCmd()
		}
	}
	return m, cmd
}

// updateModal handles input when modal is visible
func (m *Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.showModal = false
		m.nglInput.Blur()
		m.ctxSizeInput.Blur()
		return m, nil
	case "enter":
		// Save values and close modal
		if ngl, err := strconv.Atoi(m.nglInput.Value()); err == nil {
			m.sessionNGL = ngl
		}
		if ctx, err := strconv.Atoi(m.ctxSizeInput.Value()); err == nil && ctx >= 0 {
			m.sessionCtxSize = ctx
		}
		m.showModal = false
		m.nglInput.Blur()
		m.ctxSizeInput.Blur()
		m.output += fmt.Sprintf("Session config updated: NGL=%d, CtxSize=%d\n", m.sessionNGL, m.sessionCtxSize)
		return m, nil
	case "tab", "down":
		m.modalFocusIdx = (m.modalFocusIdx + 1) % 2
		if m.modalFocusIdx == 0 {
			m.nglInput.Focus()
			m.ctxSizeInput.Blur()
		} else {
			m.nglInput.Blur()
			m.ctxSizeInput.Focus()
		}
		return m, nil
	case "shift+tab", "up":
		m.modalFocusIdx = (m.modalFocusIdx + 1) % 2
		if m.modalFocusIdx == 0 {
			m.nglInput.Focus()
			m.ctxSizeInput.Blur()
		} else {
			m.nglInput.Blur()
			m.ctxSizeInput.Focus()
		}
		return m, nil
	}

	// Pass input to the focused field
	if m.modalFocusIdx == 0 {
		m.nglInput, cmd = m.nglInput.Update(msg)
	} else {
		m.ctxSizeInput, cmd = m.ctxSizeInput.Update(msg)
	}
	return m, cmd
}

// updateCliInput handles input when in CLI mode
func (m *Model) updateCliInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quit = true
		m.processMgr.Stop()
		return m, tea.Quit
	case "esc":
		m.cliMode = false
		m.cliInputBuffer = ""
		m.output += "\n[Exited CLI input mode]\n"
		return m, nil
	case "enter":
		input := m.cliInputBuffer + "\n"
		if err := m.processMgr.WriteToStdin([]byte(input)); err != nil {
			m.output += fmt.Sprintf("\n[Error sending input: %v]\n", err)
		}
		m.cliInputBuffer = ""
		return m, nil
	case "backspace":
		if len(m.cliInputBuffer) > 0 {
			m.cliInputBuffer = m.cliInputBuffer[:len(m.cliInputBuffer)-1]
		}
		return m, nil
	default:
		// Append printable characters
		if len(msg.String()) == 1 {
			m.cliInputBuffer += msg.String()
		} else if msg.Type == tea.KeySpace {
			m.cliInputBuffer += " "
		}
		return m, nil
	}
}

// updateHFSearch handles input when HF search is focused
func (m *Model) updateHFSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.hfSearchFocused = false
		m.hfSearchInput.Blur()
		return m, nil
	case "enter":
		m.hfSearchFocused = false
		m.hfSearchInput.Blur()
		query := m.hfSearchInput.Value()
		if query != "" {
			m.hfSearching = true
			m.output += fmt.Sprintf("Searching HuggingFace for '%s'...\n", query)
			return m, m.searchHFModels(query)
		}
		return m, nil
	}

	m.hfSearchInput, cmd = m.hfSearchInput.Update(msg)
	return m, cmd
}

// searchHFModels performs async search on HuggingFace
func (m *Model) searchHFModels(query string) tea.Cmd {
	return func() tea.Msg {
		models, err := m.hfClient.ListModels(hfmodels.ListModelsOptions{
			Search:      query,
			LibraryName: "gguf",
			Limit:       20,
			Sort:        "downloads",
			Direction:   -1,
		})
		return HFSearchResultMsg{Models: models, Err: err}
	}
}

// fetchQuants fetches available quantizations for a model
func (m *Model) fetchQuants(modelID string) tea.Cmd {
	return func() tea.Msg {
		quants, err := m.hfClient.GetAvailableQuants(modelID)
		return HFQuantsResultMsg{ModelID: modelID, Quants: quants, Err: err}
	}
}

// fetchModelDetails fetches detailed information about a model
func (m *Model) fetchModelDetails(modelID string) tea.Cmd {
	return func() tea.Msg {
		details, err := m.hfClient.GetModelDetails(modelID)
		return HFModelDetailsMsg{Details: details, Err: err}
	}
}

// updateQuantModal handles input when quantization modal is visible
func (m *Model) updateQuantModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.availableQuants) == 0 {
		m.showQuantModal = false
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.showQuantModal = false
		m.selectedHFModel = nil
		m.availableQuants = nil
		return m, nil
	case "up":
		m.quantSelected--
		if m.quantSelected < 0 {
			m.quantSelected = len(m.availableQuants) - 1
		}
		return m, nil
	case "down":
		m.quantSelected = (m.quantSelected + 1) % len(m.availableQuants)
		return m, nil
	case "enter":
		if m.selectedHFModel != nil && m.quantSelected < len(m.availableQuants) {
			quant := m.availableQuants[m.quantSelected]
			m.showQuantModal = false
			m.startHFServer(m.selectedHFModel.ID, quant)
			m.selectedHFModel = nil
			m.availableQuants = nil
		}
		return m, nil
	case "c":
		if m.selectedHFModel != nil && m.quantSelected < len(m.availableQuants) {
			quant := m.availableQuants[m.quantSelected]
			m.showQuantModal = false
			m.startHFCli(m.selectedHFModel.ID, quant)
			m.selectedHFModel = nil
			m.availableQuants = nil
		}
		return m, nil
	}
	return m, nil
}

// updateInfoModal handles input when info modal is visible
func (m *Model) updateInfoModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "i", "q":
		m.showInfoModal = false
		m.modelDetails = nil
		return m, nil
	}
	return m, nil
}

// updateNoQuantModal handles input when no-quant confirmation modal is visible
func (m *Model) updateNoQuantModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.showNoQuantModal = false
		m.selectedHFModel = nil
		return m, nil
	case "enter", "y":
		if m.selectedHFModel != nil {
			m.showNoQuantModal = false
			m.startHFServer(m.selectedHFModel.ID, "")
			m.selectedHFModel = nil
		}
		return m, nil
	case "c":
		if m.selectedHFModel != nil {
			m.showNoQuantModal = false
			m.startHFCli(m.selectedHFModel.ID, "")
			m.selectedHFModel = nil
		}
		return m, nil
	}
	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	// Use stored window dimensions or fall back to defaults
	width := m.windowWidth
	height := m.windowHeight
	if width == 0 {
		if cols := os.Getenv("COLUMNS"); cols != "" {
			if w, err := strconv.Atoi(cols); err == nil {
				width = w
			}
		}
		if width == 0 {
			if w, _, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
				width = w
			} else {
				width = 80
			}
		}
	}
	if height == 0 {
		if rows := os.Getenv("LINES"); rows != "" {
			if h, err := strconv.Atoi(rows); err == nil {
				height = h
			}
		}
		if height == 0 {
			if _, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
				height = h
			} else {
				height = 24
			}
		}
	}

	// Define styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	modelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DDDDEE")).
		Padding(0, 1)

	selectedModelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111111")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	outputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A6A6A6")).
		Padding(0, 1)

	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	leftPaneWidth := width / 3
	rightPaneWidth := width - leftPaneWidth - 2 // -2 for border spacing
	// Reserve space for borders (2), padding (2), title (1), blank line (1), status bar (1)
	paneHeight := height - 7
	if paneHeight < 5 {
		paneHeight = 5
	}
	outputHeight := paneHeight - 4 // title + blank + padding

	// Render tabs
	tab1 := inactiveTabStyle.Render(" 1:Local ")
	tab2 := inactiveTabStyle.Render(" 2:HuggingFace ")
	if m.activeTab == 0 {
		tab1 = activeTabStyle.Render(" 1:Local ")
	} else {
		tab2 = activeTabStyle.Render(" 2:HuggingFace ")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, tab1, tab2)

	// Create left pane content based on active tab
	var leftContent string
	if m.activeTab == 0 {
		// Local models tab
		var modelList strings.Builder
		for i, model := range m.models {
			if i == m.selected {
				modelList.WriteString(selectedModelStyle.Render(" > " + model))
			} else {
				modelList.WriteString(modelStyle.Render("   " + model))
			}
			modelList.WriteString("\n")
		}
		leftContent = modelList.String()
	} else {
		// HuggingFace tab
		var hfContent strings.Builder

		// Search input
		if m.hfSearchFocused {
			hfContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("> "))
			hfContent.WriteString(m.hfSearchInput.View())
		} else {
			hfContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("/ to search: "))
			if m.hfSearchInput.Value() != "" {
				hfContent.WriteString(m.hfSearchInput.Value())
			} else {
				hfContent.WriteString("...")
			}
		}
		hfContent.WriteString("\n\n")

		if m.hfSearching {
			hfContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")).Render("Searching..."))
		} else if len(m.hfModels) == 0 {
			hfContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("No results. Press / to search."))
		} else {
			for i, model := range m.hfModels {
				displayName := model.ID
				if len(displayName) > leftPaneWidth-8 {
					displayName = displayName[:leftPaneWidth-11] + "..."
				}
				if i == m.hfSelected {
					hfContent.WriteString(selectedModelStyle.Render(" > " + displayName))
				} else {
					hfContent.WriteString(modelStyle.Render("   " + displayName))
				}
				hfContent.WriteString("\n")
			}
		}
		leftContent = hfContent.String()
	}

	leftPane := lipgloss.NewStyle().
		Width(leftPaneWidth).
		Height(paneHeight).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#DDDDEE")).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				tabs,
				"",
				leftContent,
			),
		)

	// Create right pane (output) with scrolling
	outputLines := strings.Split(m.output, "\n")
	totalLines := len(outputLines)

	// Auto-scroll to bottom if scrollOffset would show past the end
	maxScroll := totalLines - outputHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}

	// Calculate visible range
	start := m.scrollOffset
	end := start + outputHeight
	if end > totalLines {
		end = totalLines
	}
	if start > end {
		start = end
	}

	visibleOutput := strings.Join(outputLines[start:end], "\n")

	rightPane := lipgloss.NewStyle().
		Width(rightPaneWidth).
		Height(paneHeight).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#A6A6A6")).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				titleStyle.Render(" Shell Output "),
				"",
				outputStyle.Render(visibleOutput),
			),
		)

	// Combine panes
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		rightPane,
	)

	// Add status bar
	var statusText string
	if m.cliMode && m.cliInputBuffer != "" {
		statusText = fmt.Sprintf(" > %s_ ", m.cliInputBuffer)
	} else if m.cliMode {
		statusText = " > _ (CLI mode - type and press Enter, Esc to exit) "
	} else if m.activeTab == 0 && len(m.models) > 0 {
		statusText = fmt.Sprintf(" Selected: %s | NGL: %d | CtxSize: %d ", m.models[m.selected], m.sessionNGL, m.sessionCtxSize)
	} else if m.activeTab == 1 && len(m.hfModels) > 0 {
		statusText = fmt.Sprintf(" HF: %s | NGL: %d | CtxSize: %d ", m.hfModels[m.hfSelected].ID, m.sessionNGL, m.sessionCtxSize)
	} else {
		statusText = fmt.Sprintf(" NGL: %d | CtxSize: %d | Press 1/2 for tabs ", m.sessionNGL, m.sessionCtxSize)
	}
	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Width(width).
		Render(statusText)

	result := lipgloss.JoinVertical(lipgloss.Top, content, status)

	// Render modal overlays if visible
	if m.showModal {
		result = m.renderModal(result, width, height)
	}
	if m.showQuantModal {
		result = m.renderQuantModal(result, width, height)
	}
	if m.showInfoModal {
		result = m.renderInfoModal(result, width, height)
	}
	if m.showNoQuantModal {
		result = m.renderNoQuantModal(result, width, height)
	}

	return result
}

// renderModal renders the config modal overlay
func (m *Model) renderModal(base string, width, height int) string {
	modalWidth := 40

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	focusedLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)

	nglLabel := labelStyle.Render("NGL (GPU Layers):")
	ctxLabel := labelStyle.Render("Context Size:")
	if m.modalFocusIdx == 0 {
		nglLabel = focusedLabel.Render("> NGL (GPU Layers):")
	} else {
		ctxLabel = focusedLabel.Render("> Context Size:")
	}

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true).Render("Session Config"),
		"",
		nglLabel,
		m.nglInput.View(),
		"",
		ctxLabel,
		m.ctxSizeInput.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Enter: Save | Esc: Cancel | Tab: Switch"),
	)

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF79C6")).
		Background(lipgloss.Color("#282A36"))

	modal := modalStyle.Render(modalContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")))
}

// renderQuantModal renders the quantization selection modal
func (m *Model) renderQuantModal(base string, width, height int) string {
	if len(m.availableQuants) == 0 {
		return base
	}

	modalWidth := 50
	listHeight := len(m.availableQuants)
	if listHeight > 15 {
		listHeight = 15
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)

	var quantList strings.Builder
	startIdx := 0
	if m.quantSelected >= listHeight {
		startIdx = m.quantSelected - listHeight + 1
	}
	endIdx := startIdx + listHeight
	if endIdx > len(m.availableQuants) {
		endIdx = len(m.availableQuants)
	}

	for i := startIdx; i < endIdx; i++ {
		q := m.availableQuants[i]
		if i == m.quantSelected {
			quantList.WriteString(selectedStyle.Render("> " + q))
		} else {
			quantList.WriteString(labelStyle.Render("  " + q))
		}
		if i < endIdx-1 {
			quantList.WriteString("\n")
		}
	}

	modelName := ""
	if m.selectedHFModel != nil {
		modelName = m.selectedHFModel.ID
		if len(modelName) > modalWidth-6 {
			modelName = modelName[:modalWidth-9] + "..."
		}
	}

	scrollInfo := ""
	if len(m.availableQuants) > listHeight {
		scrollInfo = fmt.Sprintf(" (%d/%d)", m.quantSelected+1, len(m.availableQuants))
	}

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true).Render("Select Quantization"+scrollInfo),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(modelName),
		"",
		quantList.String(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Enter: Server | c: CLI | Esc: Cancel"),
	)

	modal := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#50FA7B")).
		Background(lipgloss.Color("#282A36")).
		Render(modalContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")))
}

// renderInfoModal renders the model info modal
func (m *Model) renderInfoModal(base string, width, height int) string {
	if m.modelDetails == nil {
		return base
	}

	modalWidth := 60
	d := m.modelDetails

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))

	var info strings.Builder
	info.WriteString(labelStyle.Render("Model: "))
	info.WriteString(valueStyle.Render(d.ID) + "\n\n")

	info.WriteString(labelStyle.Render("Downloads: "))
	info.WriteString(infoStyle.Render(fmt.Sprintf("%d", d.Downloads)) + "\n")

	info.WriteString(labelStyle.Render("Likes: "))
	info.WriteString(infoStyle.Render(fmt.Sprintf("%d", d.Likes)) + "\n")

	if d.PipelineTag != "" {
		info.WriteString(labelStyle.Render("Task: "))
		info.WriteString(infoStyle.Render(d.PipelineTag) + "\n")
	}

	if d.GGUFInfo != nil {
		info.WriteString("\n")
		info.WriteString(labelStyle.Render("Architecture: "))
		info.WriteString(infoStyle.Render(d.GGUFInfo.Architecture) + "\n")

		info.WriteString(labelStyle.Render("Context Length: "))
		info.WriteString(infoStyle.Render(fmt.Sprintf("%d", d.GGUFInfo.ContextLength)) + "\n")

		sizeMB := float64(d.GGUFInfo.Total) / (1024 * 1024)
		sizeGB := sizeMB / 1024
		if sizeGB >= 1 {
			info.WriteString(labelStyle.Render("Size: "))
			info.WriteString(infoStyle.Render(fmt.Sprintf("%.2f GB", sizeGB)) + "\n")
		} else {
			info.WriteString(labelStyle.Render("Size: "))
			info.WriteString(infoStyle.Render(fmt.Sprintf("%.2f MB", sizeMB)) + "\n")
		}
	}

	if license := d.CardData.GetLicense(); license != "" {
		info.WriteString(labelStyle.Render("License: "))
		info.WriteString(infoStyle.Render(license) + "\n")
	}

	// Show available quants
	quants := hfmodels.ExtractQuantsFromSiblings(d.Siblings)
	if len(quants) > 0 {
		info.WriteString("\n")
		info.WriteString(labelStyle.Render("Available Quantizations:\n"))
		quantStr := strings.Join(quants, ", ")
		if len(quantStr) > modalWidth-4 {
			// Wrap long quant list
			for i := 0; i < len(quants); i += 5 {
				end := i + 5
				if end > len(quants) {
					end = len(quants)
				}
				info.WriteString(strings.TrimLeft("  "+strings.Join(quants[i:end], ", "), " ") + "\n")
			}
		} else {
			info.WriteString("  " + quantStr + "\n")
		}
	}

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true).Render("Model Information"),
		"",
		info.String(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Press Esc to close"),
	)

	modal := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF79C6")).
		Background(lipgloss.Color("#282A36")).
		Render(modalContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")))
}

// renderNoQuantModal renders the no-quant confirmation modal
func (m *Model) renderNoQuantModal(base string, width, height int) string {
	modalWidth := 55

	modelName := ""
	if m.selectedHFModel != nil {
		modelName = m.selectedHFModel.ID
		if len(modelName) > modalWidth-6 {
			modelName = modelName[:modalWidth-9] + "..."
		}
	}

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true).Render("No Quantizations Found"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render("Model: "+modelName),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render("Could not detect available quantizations."),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render("Try anyway without specifying a quant?"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("y/Enter: Server | c: CLI | n/Esc: Cancel"),
	)

	modal := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFB86C")).
		Background(lipgloss.Color("#282A36")).
		Render(modalContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")))
}

// startServer starts the llama-server process
func (m *Model) startServer() {
	modelName := m.models[m.selected]
	modelPath := filepath.Join(m.config.ModelsDir, modelName)

	m.output = fmt.Sprintf("Starting llama-server for %s (NGL=%d, CtxSize=%d)...\n", modelName, m.sessionNGL, m.sessionCtxSize)

	if err := m.processMgr.StartServer(modelPath, modelName, m.sessionNGL, m.sessionCtxSize); err != nil {
		m.output += "Error starting server: " + err.Error() + "\n"
		if m.logger != nil {
			m.logger.Error("Failed to start server", zap.Error(err))
		}
		return
	}

	m.output += "Process started (checking for output...)\n"
	go m.readOutput()
}

// startCli starts the llama-cli process
func (m *Model) startCli() {
	modelName := m.models[m.selected]
	modelPath := filepath.Join(m.config.ModelsDir, modelName)

	m.output = fmt.Sprintf("Starting llama-cli for %s (NGL=%d, CtxSize=%d)...\n", modelName, m.sessionNGL, m.sessionCtxSize)

	if err := m.processMgr.StartCLI(modelPath, modelName, m.sessionNGL, m.sessionCtxSize); err != nil {
		m.output += "Error starting CLI: " + err.Error() + "\n"
		if m.logger != nil {
			m.logger.Error("Failed to start CLI", zap.Error(err))
		}
		return
	}

	m.focusRight = true // Switch focus to right pane for interactive CLI
	m.cliMode = true    // Enable CLI input mode
	m.output += "CLI process started - type your message and press Enter...\n"
	go m.readOutput()
}

// startHFServer starts the llama-server with a HuggingFace model
func (m *Model) startHFServer(hfModel, quant string) {
	m.output = fmt.Sprintf("Starting llama-server for HF model %s:%s (NGL=%d, CtxSize=%d)...\n",
		hfModel, quant, m.sessionNGL, m.sessionCtxSize)

	if err := m.processMgr.StartServerHF(hfModel, quant, m.sessionNGL, m.sessionCtxSize); err != nil {
		m.output += "Error starting server: " + err.Error() + "\n"
		if m.logger != nil {
			m.logger.Error("Failed to start HF server", zap.Error(err))
		}
		return
	}

	m.output += "Process started (model will be downloaded if needed)...\n"
	go m.readOutput()
}

// startHFCli starts the llama-cli with a HuggingFace model
func (m *Model) startHFCli(hfModel, quant string) {
	m.output = fmt.Sprintf("Starting llama-cli for HF model %s:%s (NGL=%d, CtxSize=%d)...\n",
		hfModel, quant, m.sessionNGL, m.sessionCtxSize)

	if err := m.processMgr.StartCLIHF(hfModel, quant, m.sessionNGL, m.sessionCtxSize); err != nil {
		m.output += "Error starting CLI: " + err.Error() + "\n"
		if m.logger != nil {
			m.logger.Error("Failed to start HF CLI", zap.Error(err))
		}
		return
	}

	m.focusRight = true
	m.cliMode = true
	m.output += "CLI process started (model will be downloaded if needed)...\n"
	go m.readOutput()
}

// readOutput reads from stdout and stderr pipes
func (m *Model) readOutput() {
	stdoutPipe, stderrPipe := m.processMgr.GetOutputPipes()
	if stdoutPipe == nil {
		return
	}

	// Read from both pipes in goroutines
	go m.readPipe(stdoutPipe)
	if stderrPipe != nil {
		go m.readPipe(stderrPipe)
	}
}

// readPipe reads from a single pipe and sends output to the channel
func (m *Model) readPipe(pipe *os.File) {
	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			output := string(buf[:n])
			select {
			case m.outputChan <- output:
			default:
				// Channel is full, drop the message
			}
		}
		if err != nil {
			if m.logger != nil {
				m.logger.Debug("Pipe read error", zap.Error(err))
			}
			return
		}
	}
}
