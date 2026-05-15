# asciify

> Turn any image into ASCII art on your terminal. JPEG, PNG, GIF, WebP. 24-bit color. Half-block mode that doubles vertical resolution for photo-quality renders. Animated GIFs play in place.

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/go-1.22%2B-00ADD8?style=flat-square">
  <img alt="license" src="https://img.shields.io/badge/license-MIT-25f4ee?style=flat-square">
  <img alt="ci" src="https://img.shields.io/github/actions/workflow/status/f4rkh4d/asciify/ci.yml?style=flat-square">
</p>

A small, single-binary Go tool in the spirit of [jp2a](https://github.com/Talinx/jp2a), but more forgiving about formats and friendlier with modern terminals.

## Install

### With Homebrew (macOS, Linux)

```bash
brew install f4rkh4d/tap/asciify
```

### From a release binary

```bash
curl -L https://github.com/f4rkh4d/asciify/releases/latest/download/asciify-darwin-arm64.tar.gz | tar xz
mv asciify /usr/local/bin/
```

Linux and Windows builds are attached to every release.

### With Go

```bash
go install github.com/f4rkh4d/asciify/cmd/asciify@latest
```

### From source

```bash
git clone https://github.com/f4rkh4d/asciify
cd asciify
go build -o asciify ./cmd/asciify
```

## Usage

```bash
asciify picture.jpg              # auto width, auto color, half-block render
asciify -w 120 photo.png         # 120 columns
asciify --mode ascii photo.jpg   # classic glyph ramp (jp2a-style)
asciify --color none cat.gif     # plain text, no ANSI escapes
asciify --color true mona.jpg    # explicit 24-bit truecolor
asciify --ramp blocks logo.png   # ASCII mode with Unicode block glyphs
asciify --invert dark-on-dark.png

cat picture.jpg | asciify -      # read from stdin
```

### Render modes

| Mode         | What it does                                                                                  |
| ------------ | --------------------------------------------------------------------------------------------- |
| `half-block` | Each terminal cell encodes two stacked pixels via `▀` + fg/bg colors. Doubles vertical resolution. Default when color is on. |
| `ascii`      | Classic jp2a-style: pixel brightness maps to a character from a ramp; color tints the glyph.   |
| `auto`       | Picks `half-block` if color is on, `ascii` if `--color none`.                                 |

Half-block makes a real difference on photos because it preserves color faithfully and gives you twice the vertical detail. ASCII mode is the right choice for monochrome terminals, logs, or any place where the output needs to be readable as plain text.

### Animated GIFs

```bash
asciify --color true --loop 3 dancing-cat.gif    # play 3 times
asciify --loop 0 metronome.gif                   # play forever (Ctrl-C to stop)
asciify --animate=false single-frame.gif         # print first frame only
```

### Flags

| Flag         | Default      | Description                                                        |
| ------------ | ------------ | ------------------------------------------------------------------ |
| `-w`         | terminal w   | output width in columns; 0 autodetects terminal size               |
| `--mode`     | `auto`       | `ascii`, `half-block`, or `auto`                                   |
| `--ramp`     | `standard`   | ASCII mode: named ramp or a custom string of characters            |
| `--color`    | `auto`       | `none`, `256`, `true`, or `auto` (picks based on `$COLORTERM`)     |
| `--invert`   | `false`      | swap light and dark mapping (ASCII mode only)                      |
| `--bg`       | `none`       | background color for transparent pixels: `black`, `white`, `#RRGGBB` |
| `--animate`  | `true`       | for GIFs, play frames in place                                     |
| `--loop`     | `1`          | how many times to play an animation; `0` means forever             |
| `--version`  |              | print version and exit                                             |

### Named ramps

| Name      | Glyphs                                                                  |
| --------- | ----------------------------------------------------------------------- |
| `standard`| `` .:-=+*#%@``                                                          |
| `dense`   | 70 graded glyphs, best at wide widths                                   |
| `blocks`  | Unicode shading blocks (` ░▒▓█`); looks great at small sizes            |
| `minimal` | five-glyph minimalist set (` .oO@`)                                     |
| `binary`  | two-glyph "ink on / ink off"                                            |

Or pass a custom ramp: `--ramp ' .,-+*#'`.

## How it works

The pipeline is small and boring on purpose:

1. **Decode.** `image/jpeg`, `image/png`, `image/gif` come from the Go standard library; WebP is handled by `golang.org/x/image/webp`. New formats slot in by adding a side-effect import.
2. **Resize.** Catmull-Rom scale to the target width. ASCII mode halves the height to compensate for the 2:1 cell aspect; half-block mode keeps it 1:1 because each cell encodes two pixels.
3. **Render.**
   - *ASCII mode*: Rec. 601 luminance picks a glyph from the ramp; per-pixel RGB optionally tints it.
   - *Half-block mode*: each cell becomes `\x1b[38;2;TopRGB;48;2;BotRGBm▀`, packing two pixel colors into one terminal cell.
4. **Reset.** End every row with `\x1b[0m` so the terminal state stays clean.

For animated GIFs the frames are composed into a single RGBA canvas (so disposal modes don't ghost), rendered up-front, then printed one at a time with `\x1b[H\x1b[J` between them.

## Compatibility

| Terminal              | `--color true` | `--color 256` |
| --------------------- | -------------- | ------------- |
| iTerm2                | yes            | yes           |
| Alacritty             | yes            | yes           |
| Kitty                 | yes            | yes           |
| WezTerm               | yes            | yes           |
| Windows Terminal      | yes            | yes           |
| macOS Terminal.app    | falls back     | yes           |
| tmux (`-T xterm-256`) | passes through | yes           |

If your terminal mangles colors, try `--color 256` or `--color none`.

## Development

```bash
go test ./...
go vet ./...
go run ./cmd/asciify -w 60 examples/sample.jpg
```

The render core lives in `internal/render` and has no terminal-specific code, so it's easy to embed asciify in a larger Go program if you ever want to.

## License

[MIT](LICENSE).
