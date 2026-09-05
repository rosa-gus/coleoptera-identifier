package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/i18n"
)

func TestSelectingViewFitsTerminalHeight(t *testing.T) {
	for _, height := range []int{18, 24, 32, 48} {
		t.Run(fmt.Sprintf("height-%d", height), func(t *testing.T) {
			model := New(nil, i18n.NewLocalizer(i18n.EnglishUS))
			updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: height})
			view := updated.(Model).View()
			renderedHeight := lipgloss.Height(view.Content)

			if renderedHeight > height {
				t.Fatalf(
					"rendered height %d exceeds terminal height %d",
					renderedHeight,
					height,
				)
			}
		})
	}
}

func TestFailureRemainsInEnglishWithPortugueseInterface(t *testing.T) {
	model := New(nil, i18n.NewLocalizer(i18n.PortugueseBR))
	model.screen = failure
	model.err = errors.New("the image is empty")

	rendered := model.View().Content
	for _, expected := range []string{"IDENTIFICATION FAILED", "the image is empty"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("failure view does not contain %q", expected)
		}
	}
}

func TestLanguageMenuAppliesLocaleAndUpdatesPicker(t *testing.T) {
	model := New(nil, i18n.NewLocalizer(i18n.EnglishUS))
	updated, _ := model.Update(tea.KeyPressMsg{Text: "l"})
	model = updated.(Model)
	if !model.languageMenu {
		t.Fatal("language menu did not open")
	}

	updated, _ = model.Update(tea.KeyPressMsg{Text: "down"})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Text: "enter"})
	model = updated.(Model)
	if model.languageMenu {
		t.Fatal("language menu remained open after applying")
	}
	if model.localizer.Locale() != i18n.PortugueseBR {
		t.Fatalf("expected pt-BR, got %s", model.localizer.Locale())
	}
	if !strings.Contains(model.View().Content, "SELECIONE UMA IMAGEM") {
		t.Fatal("TUI did not render the applied Portuguese catalog")
	}
}

func TestLanguageMenuEscapePreservesLocale(t *testing.T) {
	model := New(nil, i18n.NewLocalizer(i18n.EnglishUS))
	updated, _ := model.Update(tea.KeyPressMsg{Text: "l"})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Text: "down"})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Text: "esc"})
	model = updated.(Model)
	if model.languageMenu || model.localizer.Locale() != i18n.EnglishUS {
		t.Fatal("escaping language menu changed the locale")
	}
}
