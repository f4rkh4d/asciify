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

// Mode selects which rendering strategy to use.
type Mode int

const (
	// ModeASCII maps pixel brightness to a character from a ramp, then
	// optionally tints it with a foreground color. One cell = one pixel.
	ModeASCII Mode = iota
	// ModeHalfBlock prints the Unicode upper half-block (U+2580) and uses
	// foreground + background colors to encode two stacked pixels per cell.
	// Doubles vertical resolution and bypasses the brightness-to-glyph step.
	// Requires color (ColorMode != ColorNone).
	ModeHalfBlock
)

// Options controls a single render. Zero values fall back to sane defaults.
type Options struct {
	Width      int       // target columns, default 80
	Ramp       string    // characters from light to dark, default Ramps["standard"]
	ColorMode  ColorMode // default ColorNone
	Mode       Mode      // default ModeASCII
	Invert     bool      // swap light/dark mapping (ASCII mode only)
	CellAspect float64   // height/width of one terminal cell, default 2.0
	Background color.Color
}

// Image converts a decoded image into a multi-line string ready to print.
func Image(src image.Image, opts Options) string {
	opts = withDefaults(opts)
	if opts.Mode == ModeHalfBlock && opts.ColorMode != ColorNone {
		return renderHalfBlock(src, opts)
	}
	return renderASCII(src, opts)
}

func withDefaults(opts Options) Options {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.Ramp == "" {
		opts.Ramp = Ramps[DefaultRamp]
	}
	if opts.CellAspect <= 0 {
		opts.CellAspect = 2.0
	}
	return opts
}

func renderASCII(src image.Image, opts Options) string {
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

	resized := scale(src, opts.Width, height, opts.Background)

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

func renderHalfBlock(src image.Image, opts Options) string {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return ""
	}
	// Each terminal row encodes two image rows, so the source is sampled at
	// 2x the row count. With a 2:1 cell aspect, that makes pixels square in
	// the final output.
	rows := int(float64(opts.Width) * float64(srcH) / float64(srcW))
	if rows%2 == 1 {
		rows++
	}
	if rows < 2 {
		rows = 2
	}
	resized := scale(src, opts.Width, rows, opts.Background)

	var b strings.Builder
	b.Grow((opts.Width*32 + 16) * (rows / 2))

	for y := 0; y < rows; y += 2 {
		for x := 0; x < opts.Width; x++ {
			tr, tg, tb := rgb8(resized.At(x, y))
			br, bg, bb := rgb8(resized.At(x, y+1))
			switch opts.ColorMode {
			case ColorTrue:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀",
					tr, tg, tb, br, bg, bb)
			case Color256:
				fmt.Fprintf(&b, "\x1b[38;5;%d;48;5;%dm▀",
					rgbTo256(tr, tg, tb), rgbTo256(br, bg, bb))
			}
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}

func scale(src image.Image, w, h int, bg color.Color) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if bg != nil {
		draw.Draw(dst, dst.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	}
	return dst
}

func rgb8(c color.Color) (r, g, b uint8) {
	r16, g16, b16, _ := c.RGBA()
	return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)
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
