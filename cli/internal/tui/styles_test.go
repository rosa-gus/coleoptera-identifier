package tui

import (
	"strings"
	"testing"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
)

func TestRenderResultSupportsCurrentContract(t *testing.T) {
	t.Parallel()
	result := api.Identification{
		Identification: "ladybug",
		Confidence:     0.91,
		Order:          "Coleoptera",
	}

	rendered := RenderResult(result, 64)
	for _, expected := range []string{"ladybug", "91.0%", "Coleoptera", "não calibrado"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("resultado não contém %q", expected)
		}
	}
}

func TestRenderResultPreservesScoreBarANSISequences(t *testing.T) {
	result := api.Identification{
		Identification: "ladybug",
		Confidence:     0.78,
	}

	rendered := RenderResult(result, 64)
	for _, marker := range []string{
		"[38;2;184;212;70m",
		"[38;2;47;54;18m",
	} {
		index := strings.Index(rendered, marker)
		if index < 1 || rendered[index-1] != '\x1b' {
			t.Fatalf("sequência ANSI sem ESC antes de %q", marker)
		}
	}
}

func TestSafeTextRemovesTerminalControlCharacters(t *testing.T) {
	t.Parallel()
	if got := safeText("ladybug\x1b[31m"); strings.ContainsRune(got, '\x1b') {
		t.Fatalf("sequência de controle não removida: %q", got)
	}
}
