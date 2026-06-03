package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jnovack/flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jnovack/jumpstart-themes/assets"
	"github.com/jnovack/jumpstart-themes/internal/buildinfo"
	"github.com/jnovack/jumpstart-themes/internal/decklist"
	"github.com/jnovack/thermal-tools/pkg/cups"
	"github.com/jnovack/thermal-tools/pkg/phomemo"
)

func main() {
	buildinfo.PopulateFromBuildInfo()

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "", flag.ExitOnError)

	logLevel     := fs.String("log-level", "info",          "log level (debug|info|warn|error)")
	showVer      := fs.Bool("version",     false,           "print version and exit")
	setCode      := fs.String("set",       "jtla",          "set code matching an assets/<code>.txt file")
	listThemes   := fs.Bool("list",        false,           "list all available themes and exit")
	themeName    := fs.String("theme",     "",              "theme name to render (or positional argument)")
	pngOut       := fs.String("png",       "",              "write PNG to file ('-' for stdout)")
	cupsPrint    := fs.Bool("printer",     false,           "print via CUPS (auto-detects first thermal printer)")
	blePrint     := fs.Bool("ble",        false,           "print via Bluetooth (scans for nearby Phomemo M220)")
	media        := fs.String("media",     "Custom.60x80mm","CUPS media option")
	allDir       := fs.String("all",       "",              "output dir: one dithered PNG per theme")
	allLetterDir := fs.String("all-letter",    "",            "output dir: 8.5\"×11\" PNG pages (3×3 grid, 300 DPI)")
	allLetterPDF := fs.String("all-letter-pdf","",           "output file: multi-page PDF of all letter pages")
	widthMM      := fs.Float64("width",    60.0,             "label width in mm")
	heightMM     := fs.Float64("height",   80.0,            "label height in mm")
	dpiFlag      := fs.Int("dpi",          203,             "render DPI for --png and --all output")

	_ = fs.Parse(os.Args[1:])

	lvl, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(lvl)

	if *showVer {
		fmt.Printf("decklist-print %s (%s) built %s\n",
			buildinfo.Version, buildinfo.Revision, buildinfo.BuildRFC3339)
		os.Exit(0)
	}

	if *allDir != "" && *allLetterDir != "" {
		fatal("--all and --all-letter are mutually exclusive")
	}
	if *cupsPrint && *blePrint {
		fatal("--printer and --ble are mutually exclusive")
	}

	// Load the requested set.
	deckData, err := assets.FS.ReadFile(*setCode + ".txt")
	if err != nil {
		log.Fatal().Err(err).Str("set", *setCode).Msg("set not found in assets; run --list to see available sets")
	}
	cards, err := decklist.ParseReader(bytes.NewReader(deckData))
	if err != nil {
		log.Fatal().Err(err).Msg("parse decklist")
	}
	allThemes := decklist.AllThemes(cards)

	if *listThemes {
		for _, t := range allThemes {
			fmt.Println(t)
		}
		os.Exit(0)
	}

	// ------------------------------------------------------------------ --all
	if *allDir != "" {
		if err := os.MkdirAll(*allDir, 0o755); err != nil {
			log.Fatal().Err(err).Str("dir", *allDir).Msg("create output dir")
		}
		dpi := float64(*dpiFlag)
		w := mmToPx(*widthMM, *dpiFlag)
		h := mmToPx(*heightMM, *dpiFlag)
		titlePt := decklist.TitleFontPt(allThemes, w, dpi)
		log.Info().Float64("titlePt", titlePt).Int("themes", len(allThemes)).Msg("rendering all themes")
		for _, t := range allThemes {
			tc := decklist.ThemeCards(cards, t)
			img := decklist.RenderTheme(t, tc, titlePt, dpi, w, h)
			path := filepath.Join(*allDir, safeFilename(t)+".png")
			if err := writePNG(dither(img), path); err != nil {
				log.Error().Err(err).Str("theme", t).Msg("write PNG")
				continue
			}
			log.Info().Str("theme", t).Str("file", path).Msg("wrote")
		}
		os.Exit(0)
	}

	// ------------------------------------------------------------ --all-letter
	if *allLetterDir != "" {
		if err := os.MkdirAll(*allLetterDir, 0o755); err != nil {
			log.Fatal().Err(err).Str("dir", *allLetterDir).Msg("create output dir")
		}
		lDPI := decklist.LetterDPI
		lW, lH := decklist.LetterLabelPx(*widthMM, *heightMM)
		titlePt := decklist.TitleFontPt(allThemes, lW, lDPI)
		log.Info().
			Float64("dpi", lDPI).Int("labelW", lW).Int("labelH", lH).
			Float64("titlePt", titlePt).Int("themes", len(allThemes)).
			Msg("rendering all themes for letter layout")

		var labels []image.Image
		for _, t := range allThemes {
			labels = append(labels, decklist.RenderTheme(
				t, decklist.ThemeCards(cards, t), titlePt, lDPI, lW, lH))
			log.Debug().Str("theme", t).Msg("rendered")
		}
		for i, page := range decklist.ComposeLetterPages(labels, lW, lH) {
			path := filepath.Join(*allLetterDir,
				fmt.Sprintf("%s-letter-%03d.png", *setCode, i+1))
			if err := writePNG(page, path); err != nil {
				log.Fatal().Err(err).Str("path", path).Msg("write page")
			}
			log.Info().Str("path", path).Msg("wrote page")
		}
		os.Exit(0)
	}

	// ------------------------------------------------ --all-letter-pdf
	if *allLetterPDF != "" {
		lDPI := decklist.LetterDPI
		lW, lH := decklist.LetterLabelPx(*widthMM, *heightMM)
		titlePt := decklist.TitleFontPt(allThemes, lW, lDPI)
		log.Info().
			Float64("dpi", lDPI).Int("labelW", lW).Int("labelH", lH).
			Float64("titlePt", titlePt).Int("themes", len(allThemes)).
			Msg("rendering all themes for letter PDF")

		var labels []image.Image
		for _, t := range allThemes {
			labels = append(labels, decklist.RenderTheme(
				t, decklist.ThemeCards(cards, t), titlePt, lDPI, lW, lH))
			log.Debug().Str("theme", t).Msg("rendered")
		}

		pdfBytes, err := decklist.ComposeLetterPDF(labels, lW, lH)
		if err != nil {
			log.Fatal().Err(err).Msg("compose letter PDF")
		}
		if err := os.WriteFile(*allLetterPDF, pdfBytes, 0o644); err != nil {
			log.Fatal().Err(err).Str("path", *allLetterPDF).Msg("write PDF")
		}
		log.Info().Str("path", *allLetterPDF).Int("pages", len(labels)/9+1).Msg("wrote PDF")
		os.Exit(0)
	}

	// --------------------------------------------------- single-theme mode
	theme := strings.TrimSpace(*themeName)
	if theme == "" && len(fs.Args()) > 0 {
		theme = strings.Join(fs.Args(), " ")
	}
	if theme == "" {
		printUsage()
		os.Exit(1)
	}
	if *pngOut == "" && !*cupsPrint && !*blePrint {
		fatal("specify at least one output: --png <file>, --printer, or --ble")
	}

	themeCards := decklist.ThemeCards(cards, theme)
	if len(themeCards) == 0 {
		log.Fatal().Str("theme", theme).Msg("theme not found — use --list to see available themes")
	}

	dpi := float64(*dpiFlag)
	w := mmToPx(*widthMM, *dpiFlag)
	h := mmToPx(*heightMM, *dpiFlag)
	titlePt := decklist.TitleFontPt(allThemes, w, dpi)
	log.Info().Str("theme", theme).Int("cards", len(themeCards)).
		Float64("titlePt", titlePt).Msg("rendering")

	img := dither(decklist.RenderTheme(theme, themeCards, titlePt, dpi, w, h))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *pngOut != "" {
		if err := writePNG(img, *pngOut); err != nil {
			log.Fatal().Err(err).Str("file", *pngOut).Msg("write PNG")
		}
		log.Info().Str("file", *pngOut).Msg("wrote PNG")
	}

	if *cupsPrint {
		d := cups.New("", *media) // empty name = auto-detect first thermal printer
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			log.Fatal().Err(err).Msg("encode PNG for CUPS")
		}
		log.Info().Msg("printing via CUPS")
		if err := d.Print(ctx, theme, buf.Bytes()); err != nil {
			log.Fatal().Err(err).Msg("CUPS print")
		}
	}

	if *blePrint {
		// phomemo.Open("") scans for "M220" or any "Mr.in*" advertisement —
		// the M220 family's two firmware-variant names — without requiring the
		// user to know or type the BLE device name.
		log.Info().Msg("scanning for nearby Phomemo M220 (Ctrl+C to cancel)")
		p, err := phomemo.Open(ctx, "")
		if err != nil {
			log.Fatal().Err(err).Msg("Bluetooth connect — is the M220 powered on and nearby?")
		}
		defer p.Close()
		log.Info().Msg("printing via Bluetooth")
		if err := p.PrintImage(img); err != nil {
			log.Fatal().Err(err).Msg("Bluetooth print")
		}
	}
}


func mmToPx(mm float64, dpi int) int {
	if mm <= 0 {
		return 0
	}
	return int(math.Round(mm * float64(dpi) / 25.4))
}

func safeFilename(theme string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, theme)
}

func writePNG(img image.Image, path string) error {
	if path == "-" {
		return png.Encode(os.Stdout, img)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

func dither(img image.Image) *image.Paletted {
	b := img.Bounds()
	dst := image.NewPaletted(b, color.Palette{color.White, color.Black})
	draw.FloydSteinberg.Draw(dst, b, img, b.Min)
	return dst
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error: "+msg)
	os.Exit(1)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage:
  decklist-print --set <code> <theme>  [--png <file>] [--printer] [--ble <model>]
  decklist-print --set <code> --list
  decklist-print --set <code> --all <dir>
  decklist-print --set <code> --all-letter <dir>

printing:
  --printer          send to first thermal printer detected in CUPS
  --ble              send via Bluetooth (scans for nearby Phomemo M220)

output:
  --png <file>       write PNG (use '-' for stdout)
  --all <dir>        write one dithered PNG per theme
  --all-letter <dir> write 8.5"x11" letter pages (3x3 grid, 300 DPI)`)
}
