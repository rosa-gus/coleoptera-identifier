package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestSelectingViewFitsTerminalHeight(t *testing.T) {
	for _, height := range []int{18, 24, 32, 48} {
		t.Run(fmt.Sprintf("height-%d", height), func(t *testing.T) {
			model := New(nil)
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
