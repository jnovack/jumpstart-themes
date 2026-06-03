package decklist

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"github.com/go-pdf/fpdf"
)

const (
	// LetterDPI is the resolution used when rendering labels for a letter page.
	LetterDPI = 300.0

	letterWidthIn  = 8.5
	letterHeightIn = 11.0
	letterMarginIn = 0.375 // standard printer safe margin
	letterGapMM    = 3.0   // gap between adjacent labels

	// Cut-line appearance.
	cutDashPx = 12 // length of each dash in pixels (~1 mm at 300 DPI)
	cutGapPx  = 8  // length of each gap between dashes
	cutThickPx = 1 // line weight in pixels
)

var cutColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}

// LetterLabelPx returns the pixel dimensions of one label when rendered for
// inclusion in a letter-size page (at LetterDPI).
func LetterLabelPx(labelWidthMM, labelHeightMM float64) (w, h int) {
	w = int(math.Round(labelWidthMM * LetterDPI / 25.4))
	h = int(math.Round(labelHeightMM * LetterDPI / 25.4))
	return
}

// ComposeLetterPages arranges labels in a portrait 8.5"×11" grid (3 columns ×
// 3 rows = 9 per page) centered on a white canvas. Labels must all be the same
// pixel size. Returns one image per page; leftover slots on the last page are
// left white.
//
// Dashed cut lines are drawn at every label boundary — outer edges and inter-
// label gaps — extending edge-to-edge across the page so a paper trimmer or
// guillotine can be aligned without guesswork.
func ComposeLetterPages(labels []image.Image, labelW, labelH int) []image.Image {
	pageW := int(math.Round(letterWidthIn * LetterDPI))
	pageH := int(math.Round(letterHeightIn * LetterDPI))
	margin := int(math.Round(letterMarginIn * LetterDPI * 25.4 / 25.4))
	gap := int(math.Round(letterGapMM * LetterDPI / 25.4))

	// How many columns and rows fit in the usable area?
	usableW := pageW - 2*margin
	usableH := pageH - 2*margin
	cols := (usableW + gap) / (labelW + gap)
	rows := (usableH + gap) / (labelH + gap)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	perPage := cols * rows

	// Center the grid on the page.
	gridW := cols*labelW + (cols-1)*gap
	gridH := rows*labelH + (rows-1)*gap
	startX := (pageW - gridW) / 2
	startY := (pageH - gridH) / 2

	// Pre-compute cut-line positions.
	//
	// Outer edges land on the label boundary; interior lines land in the
	// centre of each inter-label gap so both neighbours get equal white space
	// after cutting.
	vCuts := cutPositions(startX, labelW, gap, cols)
	hCuts := cutPositions(startY, labelH, gap, rows)

	var pages []image.Image
	for i := 0; i < len(labels); i += perPage {
		page := image.NewRGBA(image.Rect(0, 0, pageW, pageH))
		draw.Draw(page, page.Bounds(), image.White, image.Point{}, draw.Src)

		// Place labels.
		batch := labels[i:min(i+perPage, len(labels))]
		for j, lbl := range batch {
			col := j % cols
			row := j / cols
			x := startX + col*(labelW+gap)
			y := startY + row*(labelH+gap)
			lb := lbl.Bounds()
			draw.Draw(page,
				image.Rect(x, y, x+lb.Dx(), y+lb.Dy()),
				lbl, lb.Min, draw.Over)
		}

		// Draw cut lines on top so they are visible at label edges.
		for _, x := range vCuts {
			drawVDash(page, x, 0, pageH)
		}
		for _, y := range hCuts {
			drawHDash(page, y, 0, pageW)
		}

		pages = append(pages, page)
	}
	return pages
}

// ComposeLetterPDF generates the same letter-page grid as ComposeLetterPages
// but returns a multi-page PDF ready for browser viewing and printing.
// Each PDF page is letter size (8.5 × 11 in) with labels and cut lines.
func ComposeLetterPDF(labels []image.Image, labelW, labelH int) ([]byte, error) {
	pages := ComposeLetterPages(labels, labelW, labelH)

	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "in",
		Size:           fpdf.SizeType{Wd: letterWidthIn, Ht: letterHeightIn},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	for i, pg := range pages {
		pdf.AddPage()
		var buf bytes.Buffer
		if err := png.Encode(&buf, pg); err != nil {
			return nil, fmt.Errorf("encode page %d: %w", i+1, err)
		}
		name := fmt.Sprintf("p%04d", i+1)
		pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, &buf)
		pdf.ImageOptions(name, 0, 0, letterWidthIn, letterHeightIn,
			false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return out.Bytes(), nil
}

// cutPositions returns the pixel positions of cut lines for one axis.
//
//   - The first position is the outer edge (start of the first label).
//   - One position per inter-label gap, centred in the gap.
//   - The last position is the outer edge (end of the last label).
func cutPositions(start, labelSize, gap, count int) []int {
	pos := make([]int, 0, count+1)
	pos = append(pos, start)
	for i := 0; i < count-1; i++ {
		// Right/bottom edge of label i, plus half the gap.
		pos = append(pos, start+(i+1)*labelSize+i*gap+gap/2)
	}
	pos = append(pos, start+count*labelSize+(count-1)*gap)
	return pos
}

// drawHDash draws a horizontal dashed line at row y, from x1 to x2.
func drawHDash(dst draw.Image, y, x1, x2 int) {
	period := cutDashPx + cutGapPx
	for x := x1; x < x2; x++ {
		if (x-x1)%period < cutDashPx {
			for t := 0; t < cutThickPx; t++ {
				dst.Set(x, y+t, cutColor)
			}
		}
	}
}

// drawVDash draws a vertical dashed line at column x, from y1 to y2.
func drawVDash(dst draw.Image, x, y1, y2 int) {
	period := cutDashPx + cutGapPx
	for y := y1; y < y2; y++ {
		if (y-y1)%period < cutDashPx {
			for t := 0; t < cutThickPx; t++ {
				dst.Set(x+t, y, cutColor)
			}
		}
	}
}
