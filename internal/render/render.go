// Package render converts a decoded image.Image into a string of ANSI-styled
// characters suitable for printing to a terminal.
//
// The conversion has three knobs:
//   - Width:     number of output columns. Height is derived from the source
//                aspect ratio, compensated for the fact that terminal cells
//                are ~2x taller than wide.
//   - Ramp:      the character set used to encode brightness.
//   - ColorMode: how colors are emitted (none / 256-color / truecolor).
package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/draw"
)

// ColorMode selects how per-cell color is rendered.
type ColorMode int

const (
	// ColorNone emits no ANSI color escapes; the output is plain text.
	ColorNone ColorMode = iota
	// Color256 maps each pixel to the nearest cell in the 6x6x6 cube of the
	// xterm 256-color palette. Compatible with virtually any modern terminal.
	Color256
	// ColorTrue emits 24-bit truecolor escapes. Crisp and exact on Kitty,
	// Alacritty, iTerm2, Windows Terminal, WezTerm.
	ColorTrue
)

// Options controls a single render. Zero values fall back to sane defaults.
type Options struct {
	Width      int       // target columns, default 80
	Ramp       string    // characters from light to dark, default Ramps["standard"]
	ColorMode  ColorMode // default ColorNone
	Invert     bool      // swap light/dark mapping
	CellAspect float64   // height/width of one terminal cell, default 2.0
	Background color.Color
}

// Image converts a decoded image into a multi-line ASCII string.
func Image(src image.Image, opts Options) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.Ramp == "" {
		opts.Ramp = Ramps[DefaultRamp]
	}
	if opts.CellAspect <= 0 {
		opts.CellAspect = 2.0
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return ""
	}

	height := int(float64(opts.Width) * float64(srcH) / float64(srcW) / opts.CellAspect)
	if height < 1 {
		height = 1
	}

	resized := image.NewRGBA(image.Rect(0, 0, opts.Width, height))
	if opts.Background != nil {
		bg := image.NewUniform(opts.Background)
		draw.Draw(resized, resized.Bounds(), bg, image.Point{}, draw.Src)
		draw.CatmullRom.Scale(resized, resized.Bounds(), src, bounds, draw.Over, nil)
	} else {
		draw.CatmullRom.Scale(resized, resized.Bounds(), src, bounds, draw.Src, nil)
	}

	ramp := []rune(opts.Ramp)
	if opts.Invert {
		for i, j := 0, len(ramp)-1; i < j; i, j = i+1, j-1 {
			ramp[i], ramp[j] = ramp[j], ramp[i]
		}
	}

	var b strings.Builder
	b.Grow((opts.Width + 16) * height)

	for y := 0; y < height; y++ {
		for x := 0; x < opts.Width; x++ {
			r, g, bl, _ := resized.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
			lum := luminance(r8, g8, b8)
			// Ramp runs light->dark, so brighter pixels pick the early
			// (sparser) glyphs. Hence (255 - lum) rather than lum.
			idx := int((255.0 - lum) * float64(len(ramp)-1) / 255.0)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}

			switch opts.ColorMode {
			case ColorTrue:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm%c", r8, g8, b8, ramp[idx])
			case Color256:
				fmt.Fprintf(&b, "\x1b[38;5;%dm%c", rgbTo256(r8, g8, b8), ramp[idx])
			default:
				b.WriteRune(ramp[idx])
			}
		}
		if opts.ColorMode != ColorNone {
			b.WriteString("\x1b[0m")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// luminance returns a perceptual brightness in [0, 255] using Rec. 601.
func luminance(r, g, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// rgbTo256 maps an RGB triplet to the nearest xterm 256-color palette index.
// The palette is laid out as 16 system colors + a 6x6x6 RGB cube + 24 grays.
// For prose-quality terminals (240+ colors) the 6x6x6 cube is the right target.
func rgbTo256(r, g, b uint8) uint8 {
	cube := func(v uint8) uint8 {
		switch {
		case v < 48:
			return 0
		case v < 115:
			return 1
		default:
			return (uint8(v) - 35) / 40
		}
	}
	ir, ig, ib := cube(r), cube(g), cube(b)
	if ir == ig && ig == ib {
		// Prefer the gray ramp for desaturated cells; it's smoother.
		avg := (int(r) + int(g) + int(b)) / 3
		if avg < 8 {
			return 16
		}
		if avg > 238 {
			return 231
		}
		return uint8(232 + (avg-8)/10)
	}
	return 16 + 36*ir + 6*ig + ib
}
