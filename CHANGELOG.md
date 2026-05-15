# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-05-15

### Added
- Initial release.
- CLI `asciify` that converts JPEG, PNG, GIF, and WebP images into ASCII art on the terminal.
- Color modes: `none`, `256`, `true`, `auto` (picks based on `$COLORTERM`).
- Named character ramps: `standard`, `dense`, `blocks`, `minimal`, `binary`; or any custom string.
- Animated GIF playback with per-frame delay honoring the source file, with a `--loop` counter (0 = forever).
- Width autodetection from the controlling terminal; explicit `-w` override.
- Stdin input via `asciify -`.
- Cross-platform release binaries built in CI on every `v*` tag.
