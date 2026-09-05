package tui

import (
	"strings"
	"testing"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/i18n"
)

func TestRenderResultSupportsCurrentContract(t *testing.T) {
	t.Parallel()
	result := api.Identification{
		Identification: "ladybug",
		Confidence:     0.91,
		Order:          "Coleoptera",
	}

	rendered := RenderResult(result, 64, i18n.NewLocalizer(i18n.EnglishUS))
	for _, expected := range []string{"ladybug", "91.0%", "Coleoptera", "uncalibrated"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("result does not contain %q", expected)
		}
	}
}

func TestRenderResultPreservesScoreBarANSISequences(t *testing.T) {
	result := api.Identification{
		Identification: "ladybug",
		Confidence:     0.78,
	}

	rendered := RenderResult(result, 64, i18n.NewLocalizer(i18n.EnglishUS))
	for _, marker := range []string{
		"[38;2;184;212;70m",
		"[38;2;63;74;32m",
	} {
		index := strings.Index(rendered, marker)
		if index < 1 || rendered[index-1] != '\x1b' {
			t.Fatalf("ANSI sequence has no ESC before %q", marker)
		}
	}
}

func TestRenderResultUsesOnlyBackendDisclaimer(t *testing.T) {
	result := api.Identification{
		Identification:             "Scarabaeidae",
		Confidence:                 0.78,
		RequiresExpertConfirmation: true,
		Disclaimer:                 "Automated visual hypothesis.",
	}

	rendered := RenderResult(result, 64, i18n.NewLocalizer(i18n.EnglishUS))
	if strings.Contains(rendered, "A espécie requer confirmação especializada.") {
		t.Fatal("rendered result contains the removed duplicate disclaimer")
	}
	if !strings.Contains(rendered, result.Disclaimer) {
		t.Fatal("rendered result does not contain the backend disclaimer")
	}
}

func TestRenderResultUsesPortugueseCatalog(t *testing.T) {
	result := api.Identification{
		Identification: "Scarabaeidae",
		Confidence:     0.781,
		ReachedRank:    "family",
	}

	rendered := RenderResult(result, 64, i18n.NewLocalizer(i18n.PortugueseBR))
	for _, expected := range []string{"RESULTADO", "Identificação", "78,1%", "Família"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("Portuguese result does not contain %q", expected)
		}
	}
}

func TestSafeTextRemovesTerminalControlCharacters(t *testing.T) {
	t.Parallel()
	if got := safeText("ladybug\x1b[31m"); strings.ContainsRune(got, '\x1b') {
		t.Fatalf("control sequence was not removed: %q", got)
	}
}
