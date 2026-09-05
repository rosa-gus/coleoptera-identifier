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
