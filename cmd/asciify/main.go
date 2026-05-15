// asciify converts an image (JPEG, PNG, GIF, WebP) into ASCII art on the
// terminal. With --color it emits ANSI color escapes; animated GIFs play
// frame by frame.
//
// Usage:
//
//	asciify [flags] <path>
//	cat picture.jpg | asciify [flags] -
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/term"
	_ "golang.org/x/image/webp"

	"github.com/f4rkh4d/asciify/internal/render"
)

var version = "dev" // overwritten by ldflags at release time

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "asciify:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("asciify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "asciify %s - render an image as ASCII art\n\n", version)
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  asciify [flags] <path>")
		fmt.Fprintln(stderr, "  cat picture.jpg | asciify [flags] -")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
		fmt.Fprintln(stderr, "\nNamed ramps:")
		for name, glyphs := range render.Ramps {
			fmt.Fprintf(stderr, "  %-9s %q\n", name, glyphs)
		}
	}

	width := fs.Int("w", 0, "output width in columns (0 = autodetect terminal)")
	ramp := fs.String("ramp", render.DefaultRamp, "named ramp from the table below, or a custom string of characters")
	colorMode := fs.String("color", "auto", `color mode: "none", "256", "true", or "auto" (truecolor if $COLORTERM is set, otherwise 256)`)
	invert := fs.Bool("invert", false, "swap light and dark mapping; useful for dark terminals with bright images")
	bg := fs.String("bg", "", "background color for images with transparency: \"black\", \"white\", or \"#RRGGBB\"")
	animate := fs.Bool("animate", true, "for GIFs, play frames in place; set --animate=false to print only the first frame")
	loop := fs.Int("loop", 1, "for animated GIFs, number of loops to play; 0 means forever")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return nil
	}

	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return errors.New("missing image path; pass a file or '-' for stdin")
	}
	if len(rest) > 1 {
		return fmt.Errorf("expected exactly one path, got %d", len(rest))
	}

	var (
		reader io.Reader
		source string
	)
	if rest[0] == "-" {
		reader = bufio.NewReader(stdin)
		source = "<stdin>"
	} else {
		f, err := os.Open(rest[0])
		if err != nil {
			return err
		}
		defer f.Close()
		reader = bufio.NewReader(f)
		source = rest[0]
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}

	cm, err := parseColorMode(*colorMode)
	if err != nil {
		return err
	}

	w := *width
	if w == 0 {
		w = detectWidth(stdout)
	}

	rampGlyphs, err := resolveRamp(*ramp)
	if err != nil {
		return err
	}

	bgColor, err := parseBackground(*bg)
	if err != nil {
		return err
	}

	opts := render.Options{
		Width:      w,
		Ramp:       rampGlyphs,
		ColorMode:  cm,
		Invert:     *invert,
		Background: bgColor,
	}

	if isGIF(data) && *animate {
		g, err := gif.DecodeAll(strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("decoding GIF: %w", err)
		}
		return playGIF(stdout, g, opts, *loop)
	}

	img, format, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("decoding image: %w", err)
	}
	_ = format

	fmt.Fprint(stdout, render.Image(img, opts))
	return nil
}

func parseColorMode(s string) (render.ColorMode, error) {
	switch strings.ToLower(s) {
	case "none", "off", "no":
		return render.ColorNone, nil
	case "256":
		return render.Color256, nil
	case "true", "truecolor", "24bit", "24":
		return render.ColorTrue, nil
	case "auto":
		if os.Getenv("COLORTERM") == "truecolor" || os.Getenv("COLORTERM") == "24bit" {
			return render.ColorTrue, nil
		}
		if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" {
			return render.ColorNone, nil
		}
		return render.Color256, nil
	default:
		return 0, fmt.Errorf("unknown color mode %q (want none / 256 / true / auto)", s)
	}
}

func resolveRamp(s string) (string, error) {
	if glyphs, ok := render.Ramps[s]; ok {
		return glyphs, nil
	}
	if len(s) < 2 {
		return "", fmt.Errorf("custom ramp must be at least 2 characters; got %q", s)
	}
	return s, nil
}

func parseBackground(s string) (color.Color, error) {
	if s == "" {
		return nil, nil
	}
	switch strings.ToLower(s) {
	case "black":
		return color.Black, nil
	case "white":
		return color.White, nil
	}
	if !strings.HasPrefix(s, "#") || len(s) != 7 {
		return nil, fmt.Errorf("bg must be \"black\", \"white\", or #RRGGBB; got %q", s)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return nil, fmt.Errorf("bg: %w", err)
	}
	return color.RGBA{r, g, b, 255}, nil
}

func detectWidth(w io.Writer) int {
	type fd interface{ Fd() uintptr }
	if f, ok := w.(fd); ok {
		if cols, _, err := term.GetSize(int(f.Fd())); err == nil && cols > 0 {
			return cols
		}
	}
	return 80
}

func isGIF(data []byte) bool {
	return len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
}

func playGIF(out io.Writer, g *gif.GIF, opts render.Options, loops int) error {
	if len(g.Image) == 0 {
		return errors.New("GIF has no frames")
	}
	frames, delays := renderFrames(g, opts)

	if loops <= 0 {
		for {
			if err := playOnce(out, frames, delays); err != nil {
				return err
			}
		}
	}
	for i := 0; i < loops; i++ {
		if err := playOnce(out, frames, delays); err != nil {
			return err
		}
	}
	return nil
}

func renderFrames(g *gif.GIF, opts render.Options) ([]string, []time.Duration) {
	composite := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	if opts.Background != nil {
		bg := image.NewUniform(opts.Background)
		copyOver(composite, bg)
	}
	frames := make([]string, 0, len(g.Image))
	delays := make([]time.Duration, 0, len(g.Image))
	for i, frame := range g.Image {
		copyOverFrame(composite, frame)
		frames = append(frames, render.Image(composite, opts))
		delay := time.Duration(g.Delay[i]) * 10 * time.Millisecond
		if delay <= 0 {
			delay = 100 * time.Millisecond
		}
		delays = append(delays, delay)
	}
	return frames, delays
}

func copyOver(dst *image.RGBA, src image.Image) {
	b := dst.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
}

func copyOverFrame(dst *image.RGBA, src image.Image) {
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.At(x, y)
			_, _, _, a := c.RGBA()
			if a == 0 {
				continue
			}
			dst.Set(x, y, c)
		}
	}
}

func playOnce(out io.Writer, frames []string, delays []time.Duration) error {
	for i, f := range frames {
		// Move cursor to top-left and clear screen below; cheap and works
		// in every ANSI terminal without depending on an alt screen.
		if _, err := fmt.Fprint(out, "\x1b[H\x1b[J", f); err != nil {
			return err
		}
		time.Sleep(delays[i])
	}
	return nil
}
