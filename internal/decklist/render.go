package decklist

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	outerMargin = 2
	innerPad    = 6
	titleMaxPt  = 28.0
	titleMinPt  = 8.0
	cardMaxPt   = 16.0
	cardMinPt   = 5.0
)

var (
	fontOnce    sync.Once
	boldFont    *opentype.Font
	regularFont *opentype.Font
)

func parsedFonts() (*opentype.Font, *opentype.Font) {
	fontOnce.Do(func() {
		var err error
		if boldFont, err = opentype.Parse(gobold.TTF); err != nil {
			panic("decklist: parse gobold: " + err.Error())
		}
		if regularFont, err = opentype.Parse(goregular.TTF); err != nil {
			panic("decklist: parse goregular: " + err.Error())
		}
	})
	return boldFont, regularFont
}

// TitleFontPt returns the largest point size in [titleMinPt, titleMaxPt] at
// which every theme name fits within the title stripe of a labelWidth-px wide
// label rendered at dpi dots-per-inch. Call once per batch and reuse the
// result so every label in the set shares an identical title size.
func TitleFontPt(themes []string, labelWidth int, dpi float64) float64 {
	bold, _ := parsedFonts()
	availW := labelWidth - 2*outerMargin - 2*innerPad
	for pt := titleMaxPt; pt >= titleMinPt; pt -= 0.5 {
		face := mustFace(bold, pt, dpi)
		fits := true
		for _, t := range themes {
			if font.MeasureString(face, strings.ToUpper(t)).Round() > availW {
				fits = false
				break
			}
		}
		face.Close()
		if fits {
			return pt
		}
	}
	return titleMinPt
}

// RenderTheme creates a width×height RGBA image for the named theme rendered
// at dpi dots-per-inch. titlePt must come from TitleFontPt so every label in
// a batch shares the same header size.
//
// Layout:
//   - outerMargin px white border on all edges
//   - Black title stripe: Go Bold white text, centered
//   - innerPad px white gap
//   - Card list: Go Regular, non-basics alphabetically then basics;
//     auto-sized and evenly spaced to fill remaining height
func RenderTheme(theme string, cards []Card, titlePt, dpi float64, width, height int) image.Image {
	bold, regular := parsedFonts()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)

	titleFace := mustFace(bold, titlePt, dpi)
	defer titleFace.Close()
	titleM := titleFace.Metrics()
	titleLineH := (titleM.Ascent + titleM.Descent).Round()

	// Black title stripe with white text.
	y := outerMargin
	stripeH := innerPad + titleLineH + innerPad
	draw.Draw(dst,
		image.Rect(outerMargin, y, width-outerMargin, y+stripeH),
		image.Black, image.Point{}, draw.Src)

	titleText := strings.ToUpper(theme)
	titleW := font.MeasureString(titleFace, titleText).Round()
	tx := outerMargin + (width-2*outerMargin-titleW)/2
	if tx < outerMargin+innerPad {
		tx = outerMargin + innerPad
	}
	drawString(dst, titleFace, titleText, color.White, tx, y+innerPad+titleM.Ascent.Round())
	y += stripeH + innerPad

	if len(cards) == 0 {
		return dst
	}

	available := height - y - outerMargin
	availW := width - 2*outerMargin - 2*innerPad
	n := len(cards)

	cardPt := fitCardFont(regular, cards, availW, available, cardMaxPt, cardMinPt, dpi)
	cardFace := mustFace(regular, cardPt, dpi)
	defer cardFace.Close()
	cardM := cardFace.Metrics()
	cardLineH := (cardM.Ascent + cardM.Descent).Round()

	spacing := 0
	if n > 1 {
		if spacing = (available - cardLineH*n) / (n - 1); spacing < 0 {
			spacing = 0
		}
	}

	for _, c := range cards {
		drawString(dst, cardFace, cardLabel(c), color.Black,
			outerMargin+innerPad, y+cardM.Ascent.Round())
		y += cardLineH + spacing
	}

	return dst
}

func mustFace(f *opentype.Font, size, dpi float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Sprintf("decklist: new face %.1fpt/%.0fdpi: %v", size, dpi, err))
	}
	return face
}

func fitCardFont(f *opentype.Font, cards []Card, availWidth, availHeight int, maxPt, minPt, dpi float64) float64 {
	n := len(cards)
	for pt := maxPt; pt >= minPt; pt -= 0.5 {
		face := mustFace(f, pt, dpi)
		m := face.Metrics()
		lineH := (m.Ascent + m.Descent).Round()
		fits := lineH*n <= availHeight
		if fits {
			for _, c := range cards {
				if font.MeasureString(face, cardLabel(c)).Round() > availWidth {
					fits = false
					break
				}
			}
		}
		face.Close()
		if fits {
			return pt
		}
	}
	return minPt
}

func cardLabel(c Card) string {
	if c.Qty > 1 {
		return fmt.Sprintf("%d %s", c.Qty, c.Name)
	}
	return c.Name
}

func drawString(dst draw.Image, face font.Face, text string, c color.Color, x, y int) {
	(&font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}).DrawString(text)
}
