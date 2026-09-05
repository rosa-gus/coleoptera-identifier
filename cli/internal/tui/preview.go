package tui

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const (
	previewMaxWidth  = 32
	previewMaxRows   = 7
	previewMaxPixels = 25_000_000
)

func renderImagePreview(path string, maxWidth int, maxRows int) (string, error) {
	if maxWidth < 1 || maxRows < 1 {
		return "", nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	configuration, _, err := image.DecodeConfig(file)
	if err != nil {
		return "", err
	}
	if configuration.Width < 1 || configuration.Height < 1 {
		return "", fmt.Errorf("image has invalid dimensions")
	}
	if int64(configuration.Width)*int64(configuration.Height) > previewMaxPixels {
		return "", fmt.Errorf("image dimensions exceed the preview limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	decoded, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}
	return renderTerminalImage(decoded, maxWidth, maxRows), nil
}

func renderTerminalImage(source image.Image, maxWidth int, maxRows int) string {
	bounds := source.Bounds()
	sourceWidth := bounds.Dx()
	sourceHeight := bounds.Dy()
	if sourceWidth < 1 || sourceHeight < 1 || maxWidth < 1 || maxRows < 1 {
		return ""
	}

	aspectRatio := float64(sourceWidth) / float64(sourceHeight)
	width := maxWidth
	rows := int(math.Ceil(float64(width) / (aspectRatio * 2)))
	if rows > maxRows {
		rows = maxRows
		width = max(1, int(math.Round(aspectRatio*float64(rows)*2)))
		width = min(width, maxWidth)
	}
	rows = max(rows, 1)

	var preview strings.Builder
	for row := range rows {
		for column := range width {
			top := previewColor(source, bounds, column, row*2, width, rows*2)
			bottom := previewColor(source, bounds, column, row*2+1, width, rows*2)
			fmt.Fprintf(
				&preview,
				"\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				top.R,
				top.G,
				top.B,
				bottom.R,
				bottom.G,
				bottom.B,
			)
		}
		preview.WriteString("\x1b[0m")
		if row < rows-1 {
			preview.WriteByte('\n')
		}
	}
	return preview.String()
}

func previewColor(
	source image.Image,
	bounds image.Rectangle,
	column int,
	row int,
	width int,
	height int,
) color.NRGBA {
	x := bounds.Min.X + min(column*bounds.Dx()/width, bounds.Dx()-1)
	y := bounds.Min.Y + min(row*bounds.Dy()/height, bounds.Dy()-1)
	pixel := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
	alpha := uint16(pixel.A)
	return color.NRGBA{
		R: uint8((uint16(pixel.R)*alpha + 7*(255-alpha)) / 255),
		G: uint8((uint16(pixel.G)*alpha + 9*(255-alpha)) / 255),
		B: uint8((uint16(pixel.B)*alpha + 3*(255-alpha)) / 255),
		A: 255,
	}
}
