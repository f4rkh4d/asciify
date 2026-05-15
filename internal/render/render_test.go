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
