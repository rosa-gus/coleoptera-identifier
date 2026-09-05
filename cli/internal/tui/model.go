// Package tui contains the interactive Coleoptera client.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
	minPanelWidth = 42
	maxPanelWidth = 88
)

type screen int

const (
	selecting screen = iota
	loading
	result
	failure
)

type identifyResultMsg struct {
	result api.Identification
	err    error
}

// Model contains the main Bubble Tea interface state.
type Model struct {
	client       *api.Client
	picker       filepicker.Model
	spinner      spinner.Model
	screen       screen
	selectedPath string
	result       api.Identification
	err          error
	width        int
	height       int
}

// New creates a TUI connected to the API client.
func New(client *api.Client) Model {
	currentDirectory, err := os.Getwd()
	if err != nil {
		currentDirectory = "."
	}

	picker := filepicker.New()
	picker.CurrentDirectory = currentDirectory
	picker.AllowedTypes = []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp",
		".JPG", ".JPEG", ".PNG", ".GIF", ".BMP", ".WEBP",
	}
	picker.ShowHidden = false
	picker.ShowPermissions = false
	picker.AutoHeight = false
	picker.SetHeight(7)
	applyPickerStyles(&picker)

	activity := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(primaryGreen)),
	)

	return Model{
		client:  client,
		picker:  picker,
		spinner: activity,
		screen:  selecting,
		width:   defaultWidth,
		height:  defaultHeight,
	}
}

func (m Model) Init() tea.Cmd {
	return m.picker.Init()
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.picker.SetHeight(clamp(msg.Height-18, 4, 16))
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if (m.screen == result || m.screen == failure) && (msg.String() == "r" || msg.String() == "enter") {
			m.reset()
			return m, m.picker.Init()
		}
	}

	switch m.screen {
	case selecting:
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(message)
		if selected, path := m.picker.DidSelectFile(message); selected {
			absolutePath, err := filepath.Abs(path)
			if err != nil {
				m.screen = failure
				m.err = err
				return m, nil
			}
			m.selectedPath = absolutePath
			m.screen = loading
			return m, tea.Batch(cmd, m.spinner.Tick, identifyCmd(m.client, absolutePath))
		}
		if disabled, _ := m.picker.DidSelectDisabledFile(message); disabled {
			m.err = fmt.Errorf("selecione uma imagem JPG, PNG, GIF, BMP ou WebP")
		}
		return m, cmd

	case loading:
		if msg, ok := message.(identifyResultMsg); ok {
			if msg.err != nil {
				m.screen = failure
				m.err = msg.err
			} else {
				m.screen = result
				m.result = msg.result
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	panelWidth := clamp(m.width-4, minPanelWidth, maxPanelWidth)
	contentWidth := panelWidth - 6

	var body strings.Builder
	body.WriteString(renderHeader(panelWidth))
	body.WriteString("\n\n")

	switch m.screen {
	case selecting:
		body.WriteString(sectionTitle.Render("SELECIONE UMA IMAGEM"))
		body.WriteString("\n")
		body.WriteString(subtle.Render(truncateText(safeText(m.picker.CurrentDirectory), contentWidth)))
		body.WriteString("\n\n")
		body.WriteString(m.picker.View())
		if m.err != nil {
			body.WriteString("\n" + noticeStyle.Render(safeText(m.err.Error())))
		}
		body.WriteString("\n" + helpStyle.Render("↑/↓ navegar  •  enter abrir/selecionar  •  ← voltar  •  q sair"))

	case loading:
		body.WriteString(sectionTitle.Render("ANALISANDO"))
		body.WriteString("\n\n")
		body.WriteString(fmt.Sprintf("%s Enviando a imagem e aguardando a identificação…", m.spinner.View()))
		body.WriteString("\n\n")
		body.WriteString(subtle.Render(safeText(filepath.Base(m.selectedPath))))
		body.WriteString("\n\n" + helpStyle.Render("q cancelar"))

	case result:
		body.WriteString(RenderResult(m.result, contentWidth))
		body.WriteString("\n\n" + helpStyle.Render("r/enter analisar outra  •  q sair"))

	case failure:
		body.WriteString(sectionTitle.Render("NÃO FOI POSSÍVEL IDENTIFICAR"))
		body.WriteString("\n\n")
		body.WriteString(noticeStyle.Width(contentWidth).Render(safeText(m.err.Error())))
		body.WriteString("\n\n" + helpStyle.Render("r/enter tentar novamente  •  q sair"))
	}

	panel := panelStyle.Width(panelWidth).Render(body.String())
	view := tea.NewView(lipgloss.NewStyle().Padding(1, 2).Render(panel))
	view.AltScreen = true
	view.WindowTitle = "Coleoptera Identifier"
	view.BackgroundColor = black
	view.ForegroundColor = offWhite
	return view
}

func (m *Model) reset() {
	m.screen = selecting
	m.selectedPath = ""
	m.result = api.Identification{}
	m.err = nil
	m.picker.Path = ""
}

func identifyCmd(client *api.Client, path string) tea.Cmd {
	return func() tea.Msg {
		result, err := client.Identify(context.Background(), path)
		return identifyResultMsg{result: result, err: err}
	}
}

func applyPickerStyles(picker *filepicker.Model) {
	picker.Cursor = "›"
	picker.Styles.Cursor = lipgloss.NewStyle().Foreground(primaryGreen).Bold(true)
	picker.Styles.Directory = lipgloss.NewStyle().Foreground(primaryGreen).Bold(true)
	picker.Styles.File = lipgloss.NewStyle().Foreground(offWhite)
	picker.Styles.Selected = lipgloss.NewStyle().Foreground(primaryGreen).Bold(true)
	picker.Styles.DisabledFile = lipgloss.NewStyle().Foreground(darkGreen)
	picker.Styles.DisabledSelected = lipgloss.NewStyle().Foreground(mutedGreen)
	picker.Styles.FileSize = lipgloss.NewStyle().Foreground(mutedGreen).Width(7).Align(lipgloss.Right)
	picker.Styles.EmptyDirectory = lipgloss.NewStyle().Foreground(mutedGreen).SetString("Nenhuma imagem encontrada.")
}

func renderHeader(width int) string {
	title := titleStyle.Render("🪲 COLEOPTERA IDENTIFIER")
	lineWidth := max(width-lipgloss.Width(title)-8, 3)
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		"  ",
		headerLine.Render(strings.Repeat("━", lineWidth)),
	)
}

func safeText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, value)
}

func truncateText(value string, width int) string {
	if width < 2 || lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width("…"+string(runes)) > width {
		runes = runes[1:]
	}
	return "…" + string(runes)
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
