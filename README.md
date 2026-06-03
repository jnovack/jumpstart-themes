# decklist-print

Generates thermal-printer-ready labels for Magic: The Gathering Jumpstart
theme decks. Each label is a 60 mm × 80 mm 1-bit image: a bold inverted
header strip and a card list that scales to fill the available space.

Output options:

- Single 60 × 80 mm thermal label (CUPS or Bluetooth)
- Individual PNGs for every theme in a set
- 8.5 × 11 in letter pages (3 × 3 grid) for cutting and filing

---

## Requirements

### Hardware

| Use | What you need |
| --- | --- |
| CUPS printing | Any CUPS-registered thermal label printer |
| Bluetooth printing | Phomemo M220 (60 mm roll) |
| Letter sheets | Any inkjet or laser printer |

> **Phomemo model note** — Only the M2xx family has a 70 mm print head and
> can handle 60 mm labels. The M1xx family (M110, M120, …) and the M200/M210
> top out at 57 mm. `--ble` auto-discovers your M220; no model number needed.

### Software

- **Go 1.26+** — `brew install go` or <https://go.dev/dl/>
- **CUPS** — pre-installed on macOS; `apt install cups` on Debian/Ubuntu
- **Bluetooth** — macOS only for `--ble` (uses CoreBluetooth via TinyGo)

---

## Installation

```bash
git clone https://github.com/jnovack/jumpstart-themes
cd jumpstart-themes
make build          # produces bin/decklist-print
```

Add `bin/` to your `PATH`, or run the binary directly.

---

## Quick Start

### List every theme in a set

```bash
decklist-print --set jtla --list
```

### Print a single theme

```bash
# Auto-detect the first CUPS thermal printer
decklist-print --set jtla "Toph" --printer

# Scan for a nearby Phomemo M220 over Bluetooth
decklist-print --set jtla "Toph" --ble

# Save to PNG instead of printing
decklist-print --set jtla "Toph" --png toph.png
```

Theme names are case-sensitive and must match the list exactly.
Multi-word themes (e.g. `"Sparky Sparky 1"`) require quotes in most shells.

### Export every theme at once

```bash
# One dithered PNG per theme (matches thermal-print appearance)
decklist-print --set jtla --all ./labels/

# Letter-size pages (300 DPI, 3 × 3 grid) ready to print and cut
decklist-print --set jtla --all-letter ./letter-pages/
```

---

## Flag Reference

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--set` | string | `jtla` | Set code — must match `assets/<code>.txt` |
| `--list` | bool | — | Print all theme names and exit |
| `--theme` | string | — | Theme to render (alternative to positional arg) |
| `--png` | string | — | Write PNG to file (`-` for stdout) |
| `--printer` | bool | — | Send to first CUPS thermal printer (auto-detected) |
| `--ble` | bool | — | Send via Bluetooth to nearby Phomemo M220 |
| `--all` | string | — | Output dir for one PNG per theme |
| `--all-letter` | string | — | Output dir for letter-size pages |
| `--media` | string | `Custom.60x80mm` | CUPS media option |
| `--width` | float | `60.0` | Label width in mm |
| `--height` | float | `80.0` | Label height in mm |
| `--dpi` | int | `203` | Render DPI for `--png` and `--all` |
| `--version` | bool | — | Print version and exit |
| `--log-level` | string | `info` | `debug` / `info` / `warn` / `error` |

`--printer` and `--ble` are mutually exclusive.
`--all` and `--all-letter` are mutually exclusive.
A single-theme invocation requires at least one of `--png`, `--printer`, or `--ble`.

---

## Adding a New Set

Drop a plain-text file named `<code>.txt` into `assets/` and rebuild.

```bash
cp my-new-set.txt assets/tlk.txt
make build
decklist-print --set tlk --list
```

The file is embedded into the binary at compile time; no runtime file access
is needed.

### Decklist file format

Each non-blank line describes one card entry:

```text
<qty> <name> (<SET>) <collector-number> [#Theme1 [#Theme 2 ...]]
```

Example:

```text
1 Badgermole (TLA) 166 #Bumi #Earthbending 2 #Toph
7 Forest (TLA) 286 #Toph
```

- **Quantity** — how many copies are in the pack (basics are usually 7)
- **Name** — full card name; may contain spaces and punctuation
- **Set code** — upper-case letters inside parentheses
- **Collector number** — integer
- **Themes** — zero or more `#Tag` tokens; tags may contain spaces; parsed
  by splitting on space-then-hash

A card belongs to every theme listed on its line. Lines that don't match
the pattern are silently skipped.

### Adding basics

Each non-special theme needs seven basic lands so the deck is complete.
Append one line per theme at the bottom of the file:

```text
7 Forest (TLA) 286 #Toph
7 Mountain (TLA) 285 #Zuko
```

The **Shrines** and **Musicians** themes are exceptions — they use a custom
mix of basics already present in the file.

---

## Developer Notes

### Project layout

```text
jumpstart-themes/
├── assets/
│   ├── *.txt             # one file per set; embedded at compile time
│   └── assets.go         # //go:embed *.txt → assets.FS
├── cmd/
│   └── decklist-print/
│       ├── main.go       # flag wiring, output routing
│       └── buildinfo.go  # ldflags targets
├── internal/
│   ├── buildinfo/        # version populated from vcs.* build settings
│   └── decklist/
│       ├── parse.go      # decklist parser + ThemeCards / AllThemes
│       ├── parse_test.go
│       ├── render.go     # label image renderer (Go fonts, DPI-aware)
│       └── letter.go     # letter-page compositor
├── scripts/
│   ├── go.mk             # build / test / vet targets
│   └── variables.mk      # APPLICATION, VERSION, REVISION, BUILD_DATE
└── Makefile              # includes scripts/go.mk + scripts/variables.mk
```

### Rendering pipeline

```text
decklist file
    │
    ▼
ParseReader()          → []Card (qty, name, set, number, []themes)
    │
    ▼
ThemeCards(theme)      → []Card (non-basics α-sorted, basics last)
    │
    ▼
TitleFontPt(allThemes) → float64 (largest pt where longest theme name fits)
    │
    ▼
RenderTheme()          → image.RGBA
  ├─ outerMargin (2 px white border on all edges)
  ├─ black title stripe: Go Bold, white text, centered
  ├─ innerPad gap
  └─ card list: Go Regular, auto-sized + evenly spaced to fill height
    │
    ▼
dither()               → image.Paletted (1-bit, Floyd-Steinberg)
    │
    ▼
writePNG / CUPS lp / phomemo.PrintImage
```

### Font choices

Card labels use the **Go fonts** (`golang.org/x/image/font/gofont`), which
are bundled in the `golang.org/x/image` module already required by
`thermal-tools`. No extra font files are embedded.

| Role | Font | Rationale |
| --- | --- | --- |
| Title | `gobold` | Legible at large pixel-doubled sizes |
| Card list | `goregular` | Clean, neutral, scales well to small sizes |

Both are rendered via `golang.org/x/image/font/opentype` at `printerDPI`
(203 for thermal output, 300 for letter pages). The auto-sizing loop steps
down from a maximum point size in 0.5 pt increments until every card name
fits within the available width and all lines fit within the available height.

### Title font consistency

`TitleFontPt(allThemes, labelWidth, dpi)` iterates every theme name in the
set and returns the largest point size at which the **longest** name fits.
This value is computed once before any rendering so all labels in a batch
share an identical header size regardless of how short the individual theme
name is.

### DPI and dimensions

| Target | DPI | Label px |
| --- | --- | --- |
| Thermal printer (`--printer`, `--ble`) | 203 | 480 × 639 |
| Individual PNGs (`--png`, `--all`) | 203 (configurable) | 480 × 639 |
| Letter pages (`--all-letter`) | 300 | 709 × 945 |

`mmToPx(mm, dpi) = round(mm × dpi / 25.4)`

### Letter page layout (300 DPI)

```text
Page      2550 × 3300 px  (8.5 × 11 in)
Margin     113 px          (0.375 in each side)
Label      709 × 945 px   (60 × 80 mm)
Gap         35 px          (3 mm)

Columns  = (2550 − 2×113 + 35) / (709 + 35) = 3
Rows     = (3300 − 2×113 + 35) / (945 + 35) = 3
Per page = 9

Grid     2197 × 2905 px  centred on the canvas
```

Labels are not dithered on letter pages — antialiased grayscale looks
better through a laser/inkjet halftone screen than pre-dithered 1-bit data.

### Bluetooth printing (Phomemo M220)

Printing delegates to the `phomemo` package from
[`github.com/jnovack/thermal-tools`](https://github.com/jnovack/thermal-tools).
The protocol is ESC/POS-style raster over GATT (BLE Write Without Response,
182-byte chunks, 60 ms inter-chunk pacing).

`phomemo.Open(ctx, "")` scans for a device advertising as `"M220"` **or**
`"Mr.inM220"` (a firmware-variant name used by some units). Passing an empty
string triggers this dual-name scan automatically, so the user does not need
to know or type any device identifier.

The M220 print head is 70 mm wide (560 px at 203 DPI). Our 60 mm label
image (480 px) is transmitted at full M220 width, with the rightmost 80 px
left white — the unprinted margin falls off the 60 mm roll edge.

### CUPS printing

`cups.New("", media)` selects the printer via `lpoptions` metadata scoring:
labels matching keywords (`pedoolo`, `thermal`, `label`, `203dpi`, …) receive
a positive score; multifunction office printers are penalised. The
highest-scoring printer wins. The PNG is written to a temp file and handed
to `lp(1)` with `-o media=<spec> -o fit-to-page`.

### Dependency note

`thermal-tools` is referenced via a `replace` directive pointing to the local
checkout at `../thermal-tools`. This keeps both projects in sync during
active development. To build `decklist-print` on a new machine, clone both
repositories side-by-side:

```bash
git clone https://github.com/jnovack/thermal-tools
git clone https://github.com/jnovack/jumpstart-themes
cd jumpstart-themes
make build
```
