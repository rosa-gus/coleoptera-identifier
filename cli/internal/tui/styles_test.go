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
	for _, expected := range []string{"ladybug", "91.0%", "Coleoptera"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("resultado não contém %q", expected)
		}
	}
}

func TestSafeTextRemovesTerminalControlCharacters(t *testing.T) {
	t.Parallel()
	if got := safeText("ladybug\x1b[31m"); strings.ContainsRune(got, '\x1b') {
		t.Fatalf("sequência de controle não removida: %q", got)
	}
}
