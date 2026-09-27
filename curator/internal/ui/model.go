package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rosa-gus/coleoptera_identifier/curator/internal/curation"
)

const accent = "#66A3BF"

var (
	blue          = lipgloss.Color(accent)
	white         = lipgloss.Color("#EAF3F7")
	muted         = lipgloss.Color("#8EA5B0")
	black         = lipgloss.Color("#0B1318")
	titleStyle    = lipgloss.NewStyle().Foreground(black).Background(blue).Bold(true).Padding(0, 1)
	sectionStyle  = lipgloss.NewStyle().Foreground(blue).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(blue).Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E6BA72"))
)

type phase int

const (
	enterID phase = iota
	review
	confirm
	saved
)

type field struct{ label, key string }
type group struct {
	title  string
	fields []field
}

var groups = []group{
	{"TAXONOMY", []field{{"Coleoptera", "is_coleoptera"}, {"Family", "family"}, {"Genus", "genus"}, {"Species", "species"}, {"Label rank", "label_rank"}, {"Negative category", "negative_category"}}},
	{"CREDITS", []field{{"Photographer", "photographer"}, {"Photo license", "image_license"}, {"Observation URL", "source_record_url"}}},
	{"VERIFICATION", []field{{"Catalog source", "brazil_catalog_source"}, {"CTFB taxon URL", "brazil_catalog_taxon_url"}, {"Catalog version", "brazil_catalog_version"}, {"Catalog checked", "brazil_catalog_checked_at"}, {"Model coverage", "model_catalog_presence"}, {"Reviewer", "reviewer"}}},
	{"FILE & OBSERVATION", []field{{"Filename", "filename"}, {"Sample ID", "sample_id"}, {"Split", "split"}, {"Country", "observed_country"}, {"Observed on", "observed_at"}, {"Accessed on", "accessed_at"}, {"Notes", "notes"}}},
}

type observationMsg struct {
	observation curation.Observation
	err         error
}
type photoMsg struct {
	data               []byte
	extension, preview string
	err                error
}
type savedMsg struct {
	path string
	err  error
}
type openedPhotoMsg struct{ err error }

type Model struct {
	client           *curation.Client
	dataDir          string
	input            textinput.Model
	spinner          spinner.Model
	fields           map[string]string
	observation      curation.Observation
	photoIndex       int
	photoBytes       []byte
	extension        string
	preview          string
	phase            phase
	groupIndex       int
	fieldIndex       int
	editing          bool
	taxonomyReviewed bool
	creditsReviewed  bool
	busy             bool
	message          string
	width, height    int
}

func New(client *curation.Client, dataDir, initialID string) Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "iNaturalist observation ID"
	input.SetWidth(30)
	input.SetValue(initialID)
	input.Focus()
	activity := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(blue)),
	)
	return Model{client: client, dataDir: dataDir, input: input, spinner: activity, fields: map[string]string{}, width: 88, height: 28}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case observationMsg:
		m.busy = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.observation = msg.observation
		m.photoIndex = 0
		m.phase = review
		m.groupIndex, m.fieldIndex = 0, 0
		m.populate()
		m.busy = true
		m.message = "Fetching photo..."
		return m, tea.Batch(m.spinner.Tick, m.photoCommand())
	case photoMsg:
		m.busy = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.photoBytes, m.extension, m.preview = msg.data, msg.extension, msg.preview
		m.setPhotoFields()
		m.message = ""
		return m, nil
	case savedMsg:
		m.busy = false
		if msg.err != nil {
			m.phase = review
			m.message = msg.err.Error()
			return m, nil
		}
		m.phase, m.message = saved, "Saved "+msg.path
		return m, nil
	case openedPhotoMsg:
		if msg.err != nil {
			m.message = "Could not open photo in browser: " + msg.err.Error()
		} else {
			m.message = ""
		}
		return m, nil
	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		if m.editing {
			switch key {
			case "enter":
				selected := groups[m.groupIndex].fields[m.fieldIndex]
				m.fields[selected.key] = strings.TrimSpace(m.input.Value())
				m.editing = false
				m.input.Blur()
				if m.groupIndex == 0 || m.groupIndex == 2 {
					m.taxonomyReviewed = false
				}
				if m.groupIndex == 1 {
					m.creditsReviewed = false
				}
				return m, nil
			case "esc":
				m.editing = false
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(message)
			return m, cmd
		}
		switch m.phase {
		case enterID:
			if key == "q" {
				return m, tea.Quit
			}
			if key == "enter" {
				m.busy, m.message = true, "Fetching observation..."
				return m, tea.Batch(m.spinner.Tick, fetchCommand(m.client, strings.TrimSpace(m.input.Value())))
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(message)
			return m, cmd
		case review:
			switch key {
			case "q":
				return m, tea.Quit
			case "o":
				return m, openPhotoCommand(m.observation.Photos[m.photoIndex].ID)
			case "left", "h":
				m.groupIndex = (m.groupIndex + len(groups) - 1) % len(groups)
				m.fieldIndex = 0
			case "right", "l", "tab":
				m.groupIndex = (m.groupIndex + 1) % len(groups)
				m.fieldIndex = 0
			case "up", "k":
				m.fieldIndex = (m.fieldIndex + len(groups[m.groupIndex].fields) - 1) % len(groups[m.groupIndex].fields)
			case "down", "j":
				m.fieldIndex = (m.fieldIndex + 1) % len(groups[m.groupIndex].fields)
			case "enter":
				selected := groups[m.groupIndex].fields[m.fieldIndex]
				m.input.SetValue(m.fields[selected.key])
				m.input.SetWidth(max(20, min(m.width-46, 54)))
				m.input.Focus()
				m.editing = true
			case "[", "]":
				step := 1
				if key == "[" {
					step = -1
				}
				m.photoIndex = (m.photoIndex + step + len(m.observation.Photos)) % len(m.observation.Photos)
				m.photoBytes, m.preview = nil, ""
				m.creditsReviewed = false
				m.busy = true
				m.message = "Fetching photo..."
				return m, tea.Batch(m.spinner.Tick, m.photoCommand())
			case "t":
				m.taxonomyReviewed = !m.taxonomyReviewed
			case "c":
				m.creditsReviewed = !m.creditsReviewed
			case "n":
				if m.fields["is_coleoptera"] == "true" {
					m.fields["is_coleoptera"], m.fields["label_rank"] = "false", "outside_scope"
				} else {
					m.fields["is_coleoptera"], m.fields["label_rank"] = "true", "species"
				}
				m.taxonomyReviewed = false
			case "ctrl+s":
				input := m.saveInput()
				if err := curation.Validate(input); err != nil {
					m.message = err.Error()
					return m, nil
				}
				m.phase = confirm
			}
		case confirm:
			switch key {
			case "esc":
				m.phase = review
			case "enter":
				m.busy = true
				m.message = "Saving photo, credits, and manifest..."
				return m, tea.Batch(m.spinner.Tick, saveCommand(m.dataDir, m.saveInput()))
			}
		case saved:
			switch key {
			case "q":
				return m, tea.Quit
			case "r":
				m.phase = enterID
				m.input.SetValue("")
				m.input.Focus()
				m.message = ""
				m.observation = curation.Observation{}
				m.fields = map[string]string{}
				return m, nil
			}
		}
	}
	if !m.busy && (m.phase == enterID || m.editing) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(message)
		return m, cmd
	}
	return m, nil
}

func (m *Model) populate() {
	obs := m.observation
	today := time.Now().Format("2006-01-02")
	isBeetle := obs.Order == "Coleoptera"
	status, rank, category := "false", "outside_scope", "other_organisms"
	if isBeetle {
		status, rank, category = "true", "species", ""
	}
	m.fields = map[string]string{
		"is_coleoptera": status, "family": obs.Family, "genus": obs.Genus,
		"species": obs.Species, "label_rank": rank, "negative_category": category,
		"brazil_catalog_source": "CTFB", "brazil_catalog_checked_at": "",
		"model_catalog_presence": "unknown", "observation_id": fmt.Sprint(obs.ID),
		"source": "iNaturalist", "source_record_url": fmt.Sprintf("https://www.inaturalist.org/observations/%d", obs.ID),
		"observed_country": obs.Country, "observed_at": obs.ObservedOn,
		"accessed_at": today,
	}
	m.taxonomyReviewed, m.creditsReviewed = false, false
}

func (m *Model) setPhotoFields() {
	photo := m.observation.Photos[m.photoIndex]
	m.fields["photographer"] = curation.PhotoAuthor(photo)
	m.fields["image_license"] = curation.LicenseURL(photo.LicenseCode)
	m.fields["sample_id"] = fmt.Sprintf("inat-%d-%d", m.observation.ID, photo.ID)
	m.fields["photo_id"] = fmt.Sprint(photo.ID)
	species := m.fields["species"]
	if species == "" {
		species = m.fields["family"]
	}
	m.fields["filename"] = fmt.Sprintf("%s-%d%s", slug(species), photo.ID, m.extension)
	m.fields["notes"] = fmt.Sprintf("iNaturalist photo ID %d", photo.ID)
}

func slug(value string) string {
	value = strings.ToLower(value)
	var result strings.Builder
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			result.WriteRune(ch)
		} else if result.Len() > 0 && !strings.HasSuffix(result.String(), "-") {
			result.WriteByte('-')
		}
	}
	return strings.Trim(result.String(), "-")
}

func (m Model) saveInput() curation.SaveInput {
	fields := make(map[string]string, len(m.fields))
	for key, value := range m.fields {
		if key != "filename" && key != "negative_category" {
			fields[key] = value
		}
	}
	return curation.SaveInput{Fields: fields, PhotoBytes: m.photoBytes, Extension: m.extension, Filename: m.fields["filename"], NegativeCategory: m.fields["negative_category"], ObservationID: m.observation.ID, PhotoID: m.observation.Photos[m.photoIndex].ID, TaxonomyReviewed: m.taxonomyReviewed, CreditsReviewed: m.creditsReviewed}
}

func fetchCommand(client *curation.Client, id string) tea.Cmd {
	return func() tea.Msg {
		observation, err := client.Fetch(context.Background(), id)
		return observationMsg{observation, err}
	}
}

func (m Model) photoCommand() tea.Cmd {
	photo := m.observation.Photos[m.photoIndex]
	return func() tea.Msg {
		data, extension, err := m.client.FetchPhoto(context.Background(), photo)
		if err != nil {
			return photoMsg{err: err}
		}
		preview, err := imagePreview(data, 27, 5)
		return photoMsg{data, extension, preview, err}
	}
}

func saveCommand(directory string, input curation.SaveInput) tea.Cmd {
	return func() tea.Msg { path, err := curation.Save(directory, input); return savedMsg{path, err} }
}

func photoPageURL(photoID int64) string {
	return fmt.Sprintf("https://www.inaturalist.org/photos/%d", photoID)
}

func openPhotoCommand(photoID int64) tea.Cmd {
	return func() tea.Msg {
		var command *exec.Cmd
		url := photoPageURL(photoID)
		switch runtime.GOOS {
		case "darwin":
			command = exec.Command("open", url)
		case "windows":
			command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			command = exec.Command("xdg-open", url)
		}
		return openedPhotoMsg{err: command.Run()}
	}
}

func (m Model) View() tea.View {
	width := min(max(m.width-4, 58), 108)
	var body strings.Builder
	body.WriteString(titleStyle.Render("🔎 coleoptera identifier"))
	body.WriteString(mutedStyle.Render(" · photo curator"))
	body.WriteString("\n")
	switch m.phase {
	case enterID:
		body.WriteString("\n" + sectionStyle.Render("iNaturalist observation ID") + "\n\n")
		if m.busy {
			body.WriteString(m.loadingView() + "\n\n")
		} else {
			body.WriteString(m.input.View() + "\n\n")
		}
		body.WriteString(mutedStyle.Render("Fetch metadata, select a photo, review, then save."))
		body.WriteString("\n\n" + mutedStyle.Render("Enter fetch  •  q quit"))
	case review:
		body.WriteString("\n" + sectionStyle.Render(fmt.Sprintf("Observation %d", m.observation.ID)))
		body.WriteString("\n" + m.reviewBody(width))
		body.WriteString("\n" + mutedStyle.Render("←→ tabs  ↑↓ field  Enter edit  [ ] photo  o browser  t/c verify  Ctrl+S save  q quit"))
	case confirm:
		body.WriteString("\n" + sectionStyle.Render("Confirm save") + "\n\n")
		body.WriteString("Observation: " + m.fields["observation_id"] + "  Photo: " + fmt.Sprint(m.observation.Photos[m.photoIndex].ID) + "\n")
		body.WriteString("Species: " + m.fields["species"] + "\n")
		body.WriteString("Photographer: " + m.fields["photographer"] + "\n")
		body.WriteString("License: " + m.fields["image_license"] + "\n")
		body.WriteString("File: " + m.fields["filename"] + "\n\n")
		if m.busy {
			body.WriteString(m.loadingView())
		} else {
			body.WriteString(mutedStyle.Render("Enter save photo, credits, and CSV  •  Esc return"))
		}
	case saved:
		body.WriteString("\n" + sectionStyle.Render("Saved") + "\n\n")
		body.WriteString(m.message + "\n\n" + mutedStyle.Render("r new observation  •  q quit"))
	}
	if m.message != "" && !m.busy && m.phase != saved {
		body.WriteString("\n\n" + warningStyle.Render(m.message))
	}
	view := tea.NewView(lipgloss.NewStyle().Foreground(white).Background(black).Padding(1, 2).Width(width).Render(body.String()))
	view.AltScreen = true
	view.WindowTitle = "🔎 coleoptera identifier · photo curator"
	return view
}

func (m Model) loadingView() string {
	return m.spinner.View() + " " + m.message
}

func (m Model) reviewBody(width int) string {
	leftWidth := 31
	rightWidth := max(25, width-leftWidth-5)
	photo := m.observation.Photos[m.photoIndex]
	var left strings.Builder
	left.WriteString(sectionStyle.Render(fmt.Sprintf("PHOTO %d/%d", m.photoIndex+1, len(m.observation.Photos))))
	left.WriteString(mutedStyle.Render(fmt.Sprintf(" · ID %d", photo.ID)))
	left.WriteString("\n")
	preview := m.preview
	if preview != "" {
		lines := strings.Split(preview, "\n")
		maxRows := max(1, min(5, m.height-20))
		if m.message != "" {
			maxRows = max(1, maxRows-2)
		}
		if len(lines) > maxRows {
			preview = strings.Join(lines[:maxRows], "\n")
		}
	}
	if preview == "" {
		if m.busy {
			left.WriteString(m.loadingView())
		} else {
			left.WriteString(mutedStyle.Render("Image unavailable"))
		}
	} else {
		left.WriteString(preview)
	}
	if !m.busy {
		photoURL := photoPageURL(photo.ID)
		left.WriteString("\n\n " + ansi.SetHyperlink(photoURL) + selectedStyle.Render("↗ Open full photo [o]") + ansi.ResetHyperlink())
	}
	left.WriteString("\n\n" + sectionStyle.Render("TARGET"))
	left.WriteString("\n" + ellipsize(m.fields["family"], leftWidth-4))
	left.WriteString("\n" + ellipsize(m.fields["filename"], leftWidth-4))
	left.WriteString("\n\n" + sectionStyle.Render("REVIEW"))
	left.WriteString("\n" + check(m.taxonomyReviewed) + " Taxonomy  [t]")
	left.WriteString("\n" + check(m.creditsReviewed) + " Credits   [c]")
	var right strings.Builder
	for index, group := range groups {
		name := group.title
		if index == m.groupIndex {
			right.WriteString(selectedStyle.Render("[" + name + "]"))
		} else {
			right.WriteString(mutedStyle.Render(" " + name + " "))
		}
		if index < len(groups)-1 {
			right.WriteString(" ")
		}
	}
	right.WriteString("\n\n")
	for index, item := range groups[m.groupIndex].fields {
		value := m.fields[item.key]
		if value == "" {
			value = "—"
		}
		prefix := "  "
		if index == m.fieldIndex {
			prefix = "> "
		}
		label := fmt.Sprintf("%-17s", item.label)
		line := prefix + label + ellipsize(value, max(8, rightWidth-22))
		if index == m.fieldIndex {
			line = selectedStyle.Render(line)
		}
		right.WriteString(line + "\n")
		if index == m.fieldIndex && m.editing {
			right.WriteString("  " + m.input.View() + "\n")
		}
	}
	leftPanel := lipgloss.NewStyle().Width(leftWidth).Border(lipgloss.RoundedBorder()).BorderForeground(blue).Padding(0, 1).Render(left.String())
	rightPanel := lipgloss.NewStyle().Width(rightWidth).Border(lipgloss.RoundedBorder()).BorderForeground(blue).Padding(0, 1).Render(right.String())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
}

func check(value bool) string {
	if value {
		return selectedStyle.Render("[x]")
	}
	return "[ ]"
}
func ellipsize(value string, width int) string {
	if width < 2 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-1]) + "…"
}

func imagePreview(data []byte, width, rows int) (string, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("photo could not be decoded: %w", err)
	}
	if config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 25_000_000 {
		return "", fmt.Errorf("photo dimensions are not supported")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("photo could not be decoded: %w", err)
	}
	bounds := source.Bounds()
	ratio := float64(bounds.Dx()) / float64(bounds.Dy())
	actualRows := min(rows, max(1, int(math.Ceil(float64(width)/(ratio*2)))))
	actualWidth := min(width, max(1, int(math.Round(ratio*float64(actualRows)*2))))
	var result strings.Builder
	for row := range actualRows {
		for column := range actualWidth {
			top := previewColor(source, bounds, column, row*2, actualWidth, actualRows*2)
			bottom := previewColor(source, bounds, column, row*2+1, actualWidth, actualRows*2)
			fmt.Fprintf(&result, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
		}
		result.WriteString("\x1b[0m")
		if row < actualRows-1 {
			result.WriteByte('\n')
		}
	}
	return result.String(), nil
}

func previewColor(source image.Image, bounds image.Rectangle, x, y, width, height int) color.NRGBA {
	x = bounds.Min.X + min(x*bounds.Dx()/width, bounds.Dx()-1)
	y = bounds.Min.Y + min(y*bounds.Dy()/height, bounds.Dy()-1)
	return color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
}
