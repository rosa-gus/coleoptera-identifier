package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rosa-gus/coleoptera_identifier/cli/internal/api"
	"github.com/rosa-gus/coleoptera_identifier/cli/internal/i18n"
)

var (
	primaryGreen = lipgloss.Color("#B8D446")
	mutedGreen   = lipgloss.Color("#73852C")
	darkGreen    = lipgloss.Color("#2F3612")
	trackGreen   = lipgloss.Color("#3F4A20")
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

// RenderResult presents current and future API contracts with the same visual identity.
func RenderResult(result api.Identification, width int, localizer *i18n.Localizer) string {
	return renderResultWithPreview(result, "", width, localizer)
}

func renderResultWithPreview(
	result api.Identification,
	preview string,
	width int,
	localizer *i18n.Localizer,
) string {
	width = max(width, 32)
	var body strings.Builder
	body.WriteString(sectionTitle.Render(localizer.Text(i18n.TUIResult)))
	body.WriteString("\n\n")
	if preview != "" {
		body.WriteString(lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(preview))
		body.WriteString("\n\n")
	}
	if result.Status == "not_coleoptera" {
		body.WriteString(noticeStyle.Width(width).Render(localizer.Text(i18n.TUIOutOfScope)))
		return body.String()
	}

	name := result.Prediction
	if name == "" {
		name = result.Identification
	}
	writeField(&body, localizer.Text(i18n.ResultIdentification), displayValue(name, localizer))
	writeScoreField(&body, result.Confidence, localizer)
	if !result.ConfidenceCalibrated {
		writeField(&body, localizer.Text(i18n.ResultCalibration), localizer.Text(i18n.ResultUncalibrated))
	}

	if result.ReachedRank != "" {
		writeField(&body, localizer.Text(i18n.ResultSuggestedRank), rankLabel(result.ReachedRank, localizer))
	}
	if result.Taxonomy != nil {
		body.WriteString("\n")
		body.WriteString(renderTaxonomy(*result.Taxonomy, localizer))
	} else if result.Order != "" {
		writeField(&body, localizer.Text(i18n.ResultOrder), result.Order)
	}
	if result.DetectedClass != "" {
		writeField(&body, localizer.Text(i18n.ResultDetectedClass), result.DetectedClass)
	}
	if result.DetectedOrder != "" {
		writeField(&body, localizer.Text(i18n.ResultSuggestedOrder), result.DetectedOrder)
	}
	if len(result.Candidates) > 0 {
		body.WriteString("\n")
		body.WriteString(labelStyle.Render(localizer.Text(i18n.ResultAlternatives)))
		body.WriteString("\n")
		for index, candidate := range result.Candidates {
			if index == 3 {
				break
			}
			body.WriteString(fmt.Sprintf("  %d. %s  %s\n", index+1, valueStyle.Render(safeText(candidate.ScientificName)), subtle.Render(localizer.Percent(candidate.Confidence))))
		}
	}
	if result.Disclaimer != "" {
		body.WriteString("\n\n")
		body.WriteString(subtle.Width(width).Render(safeText(result.Disclaimer)))
	}

	return body.String()
}

func writeField(body *strings.Builder, label, value string) {
	body.WriteString(labelStyle.Render(label + ":"))
	body.WriteString(" ")
	body.WriteString(valueStyle.Render(safeText(value)))
	body.WriteString("\n")
}

func writeScoreField(body *strings.Builder, confidence float64, localizer *i18n.Localizer) {
	body.WriteString(labelStyle.Render(localizer.Text(i18n.ResultModelScore) + ":"))
	body.WriteString(" ")
	body.WriteString(valueStyle.Render(localizer.Percent(confidence)))
	body.WriteString("  ")
	body.WriteString(confidenceBar(confidence, 18))
	body.WriteString("\n")
}

func displayValue(value string, localizer *i18n.Localizer) string {
	if strings.TrimSpace(value) == "" {
		return localizer.Text(i18n.ResultNotProvided)
	}
	return value
}

func confidenceBar(confidence float64, width int) string {
	confidence = min(max(confidence, 0), 1)
	filled := int(confidence*float64(width) + 0.5)
	return lipgloss.NewStyle().Foreground(primaryGreen).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(trackGreen).Render(strings.Repeat("░", width-filled))
}

func renderTaxonomy(taxonomy api.Taxonomy, localizer *i18n.Localizer) string {
	ranks := []struct {
		label string
		value string
	}{
		{localizer.Text(i18n.RankKingdom), taxonomy.Kingdom},
		{localizer.Text(i18n.RankPhylum), taxonomy.Phylum},
		{localizer.Text(i18n.RankClass), taxonomy.Class},
		{localizer.Text(i18n.RankOrder), taxonomy.Order},
		{localizer.Text(i18n.RankFamily), taxonomy.Family},
		{localizer.Text(i18n.RankGenus), taxonomy.Genus},
		{localizer.Text(i18n.RankSpecies), taxonomy.Species},
	}

	var body strings.Builder
	body.WriteString(labelStyle.Render(localizer.Text(i18n.ResultTaxonomy)))
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

func rankLabel(rank string, localizer *i18n.Localizer) string {
	labels := map[string]i18n.Key{
		"kingdom": i18n.RankKingdom,
		"phylum":  i18n.RankPhylum,
		"class":   i18n.RankClass,
		"order":   i18n.RankOrder,
		"family":  i18n.RankFamily,
		"genus":   i18n.RankGenus,
		"species": i18n.RankSpecies,
	}
	if key, ok := labels[strings.ToLower(rank)]; ok {
		return localizer.Text(key)
	}
	return rank
}
