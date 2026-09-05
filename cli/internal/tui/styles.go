package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
)

var (
	primaryGreen = lipgloss.Color("#B8D446")
	mutedGreen   = lipgloss.Color("#73852C")
	darkGreen    = lipgloss.Color("#2F3612")
	black        = lipgloss.Color("#070903")
	offWhite     = lipgloss.Color("#F3F7E9")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(black).
			Background(primaryGreen).
			Padding(0, 1)
	headerLine   = lipgloss.NewStyle().Foreground(primaryGreen)
	sectionTitle = lipgloss.NewStyle().Foreground(primaryGreen).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(mutedGreen).Bold(true)
	valueStyle   = lipgloss.NewStyle().Foreground(offWhite)
	subtle       = lipgloss.NewStyle().Foreground(mutedGreen)
	helpStyle    = lipgloss.NewStyle().Foreground(mutedGreen)
	noticeStyle  = lipgloss.NewStyle().Foreground(primaryGreen).Italic(true)
	panelStyle   = lipgloss.NewStyle().
			Foreground(offWhite).
			Background(black).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryGreen).
			Padding(1, 2)
)

// RenderResult apresenta contratos atuais e futuros da API com a mesma
// identidade visual.
func RenderResult(result api.Identification, width int) string {
	width = max(width, 32)
	var body strings.Builder
	body.WriteString(sectionTitle.Render("RESULTADO"))
	body.WriteString("\n\n")

	name := result.Prediction
	if name == "" {
		name = result.Identification
	}
	writeField(&body, "Identificação", displayValue(name))
	writeField(&body, "Confiança", fmt.Sprintf("%.1f%%  %s", result.Confidence*100, confidenceBar(result.Confidence, 18)))

	if result.ReachedRank != "" {
		writeField(&body, "Nível confiável", result.ReachedRank)
	}
	if result.Taxonomy != nil {
		body.WriteString("\n")
		body.WriteString(renderTaxonomy(*result.Taxonomy))
	} else if result.Order != "" {
		writeField(&body, "Ordem", result.Order)
	}
	if result.DetectedClass != "" {
		writeField(&body, "Classe detectada", result.DetectedClass)
	}
	if len(result.Candidates) > 0 {
		body.WriteString("\n")
		body.WriteString(labelStyle.Render("Alternativas"))
		body.WriteString("\n")
		for index, candidate := range result.Candidates {
			if index == 3 {
				break
			}
			body.WriteString(fmt.Sprintf("  %d. %s  %s\n", index+1, valueStyle.Render(safeText(candidate.ScientificName)), subtle.Render(fmt.Sprintf("%.1f%%", candidate.Confidence*100))))
		}
	}
	if result.RequiresExpertConfirmation {
		body.WriteString("\n")
		body.WriteString(noticeStyle.Width(width).Render("A espécie requer confirmação especializada."))
	}

	return body.String()
}

func writeField(body *strings.Builder, label, value string) {
	body.WriteString(labelStyle.Render(label + ":"))
	body.WriteString(" ")
	body.WriteString(valueStyle.Render(safeText(value)))
	body.WriteString("\n")
}

func displayValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "não informado"
	}
	return value
}

func confidenceBar(confidence float64, width int) string {
	confidence = min(max(confidence, 0), 1)
	filled := int(confidence*float64(width) + 0.5)
	return lipgloss.NewStyle().Foreground(primaryGreen).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(darkGreen).Render(strings.Repeat("░", width-filled))
}

func renderTaxonomy(taxonomy api.Taxonomy) string {
	ranks := []struct {
		label string
		value string
	}{
		{"Reino", taxonomy.Kingdom},
		{"Filo", taxonomy.Phylum},
		{"Classe", taxonomy.Class},
		{"Ordem", taxonomy.Order},
		{"Família", taxonomy.Family},
		{"Gênero", taxonomy.Genus},
		{"Espécie", taxonomy.Species},
	}

	var body strings.Builder
	body.WriteString(labelStyle.Render("Taxonomia"))
	body.WriteString("\n")
	depth := 0
	for _, rank := range ranks {
		if rank.value == "" {
			continue
		}
		body.WriteString(strings.Repeat("  ", depth))
		if depth > 0 {
			body.WriteString(subtle.Render("└─ "))
		}
		body.WriteString(valueStyle.Render(safeText(rank.value)))
		body.WriteString(subtle.Render("  " + rank.label))
		body.WriteString("\n")
		depth++
	}
	return body.String()
}
