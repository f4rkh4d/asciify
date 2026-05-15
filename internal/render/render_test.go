package render

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func countLines(s string) int {
	n := strings.Count(s, "\n")
	return n
}

func TestImage_WhiteUsesLightestRampChar(t *testing.T) {
	out := Image(solid(40, 20, color.White), Options{Width: 20, Ramp: " .:-=+*#%@"})
	// Strip newlines; every cell should be the lightest character (space).
	stripped := strings.ReplaceAll(out, "\n", "")
	for _, r := range stripped {
		if r != ' ' {
			t.Fatalf("expected all spaces for a white image, got %q in: %q", r, stripped)
		}
	}
}

func TestImage_BlackUsesDarkestRampChar(t *testing.T) {
	out := Image(solid(40, 20, color.Black), Options{Width: 20, Ramp: " .:-=+*#%@"})
	stripped := strings.ReplaceAll(out, "\n", "")
	for _, r := range stripped {
		if r != '@' {
			t.Fatalf("expected all '@' for a black image, got %q in: %q", r, stripped)
		}
	}
}

func TestImage_InvertSwapsExtremes(t *testing.T) {
	out := Image(solid(20, 10, color.White), Options{Width: 10, Ramp: " .:-=+*#%@", Invert: true})
	stripped := strings.ReplaceAll(out, "\n", "")
	for _, r := range stripped {
		if r != '@' {
			t.Fatalf("expected inverted white -> '@', got %q", r)
		}
	}
}

func TestImage_AspectRatioCompensated(t *testing.T) {
	// Square source rendered at width=20 should yield ~10 rows because
	// terminal cells are 2x taller than wide.
	out := Image(solid(40, 40, color.Black), Options{Width: 20})
	if got := countLines(out); got < 8 || got > 12 {
		t.Fatalf("expected ~10 lines for a square image at width 20, got %d", got)
	}
}

func TestImage_TrueColorEmitsEscape(t *testing.T) {
	out := Image(solid(8, 4, color.RGBA{200, 50, 100, 255}), Options{Width: 4, ColorMode: ColorTrue})
	if !strings.Contains(out, "\x1b[38;2;200;50;100m") {
		t.Fatalf("expected truecolor escape in output, got %q", out)
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "\x1b[0m") {
		t.Fatalf("expected reset escape at end of each line, got %q", out)
	}
}

func TestRgbTo256_Greys(t *testing.T) {
	idx := rgbTo256(128, 128, 128)
	if idx < 232 || idx > 255 {
		t.Fatalf("expected gray ramp index for desaturated input, got %d", idx)
	}
}

func TestImage_EmptyImageReturnsEmpty(t *testing.T) {
	out := Image(image.NewRGBA(image.Rect(0, 0, 0, 0)), Options{})
	if out != "" {
		t.Fatalf("expected empty output for zero-size image, got %q", out)
	}
}

func TestHalfBlock_EmitsForegroundAndBackground(t *testing.T) {
	// Two-row image: top row red, bottom row blue.
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for x := 0; x < 4; x++ {
		img.Set(x, 0, color.RGBA{255, 0, 0, 255})
		img.Set(x, 1, color.RGBA{0, 0, 255, 255})
	}
	out := Image(img, Options{Width: 4, Mode: ModeHalfBlock, ColorMode: ColorTrue})
	if !strings.Contains(out, "\x1b[38;2;255;0;0;48;2;0;0;255m▀") {
		t.Fatalf("expected red-fg + blue-bg half-block escape, got %q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Fatalf("expected reset after row, got %q", out)
	}
}

func TestHalfBlock_DoublesVerticalResolution(t *testing.T) {
	// A 40x40 square image at width=20 should yield ~10 rows in ASCII mode
	// (2:1 cell aspect) and ~20 rows of *image content* (10 terminal lines
	// since each line packs 2 pixels) in half-block mode. Compare line counts.
	src := solid(40, 40, color.Black)
	ascii := Image(src, Options{Width: 20})
	half := Image(src, Options{Width: 20, Mode: ModeHalfBlock, ColorMode: ColorTrue})
	asciiLines := strings.Count(ascii, "\n")
	halfLines := strings.Count(half, "\n")
	// In half-block, terminal lines == ascii lines (because we render 2x rows
	// then divide by 2), but each line carries 2 source pixels. The contract
	// is: same number of terminal rows for a square image at the same width.
	if halfLines < asciiLines-2 || halfLines > asciiLines+2 {
		t.Fatalf("half-block (%d) should produce ~same row count as ascii (%d) for a square image",
			halfLines, asciiLines)
	}
}

func TestHalfBlock_FallsBackToASCIIWithoutColor(t *testing.T) {
	// ModeHalfBlock with ColorNone is nonsense; the renderer should not emit
	// raw half-block characters without color escapes. We fall back to ASCII.
	out := Image(solid(8, 8, color.Black), Options{Width: 8, Mode: ModeHalfBlock, ColorMode: ColorNone})
	if strings.Contains(out, "▀") {
		t.Fatalf("expected no half-block char when color is disabled, got %q", out)
	}
}
