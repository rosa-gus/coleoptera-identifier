package tui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderImagePreviewFitsRequestedArea(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beetle.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	imageData := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	green := color.NRGBA{R: 184, G: 212, B: 70, A: 255}
	for y := range 40 {
		for x := range 80 {
			imageData.SetNRGBA(x, y, green)
		}
	}
	if err := png.Encode(file, imageData); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	preview, err := renderImagePreview(path, 24, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview, "▀") || !strings.Contains(preview, "\x1b[38;2;") {
		t.Fatalf("preview does not contain colored terminal cells: %q", preview)
	}
	if width := lipgloss.Width(preview); width > 24 {
		t.Fatalf("preview width %d exceeds requested width", width)
	}
	if height := lipgloss.Height(preview); height > 6 {
		t.Fatalf("preview height %d exceeds requested height", height)
	}
}
