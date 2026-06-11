// Package render contains PDF and HTML rendering for token boards.
package render

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/owner/tokenBoardCreator/internal/assets"
	"github.com/owner/tokenBoardCreator/internal/board"
)

// pageDims holds the physical page dimensions in mm.
type pageDims struct {
	width  float64
	height float64
}

var pageSizes = map[string]pageDims{
	"letter": {215.9, 279.4},
	"a4":     {210.0, 297.0},
}

// PDF generates a token board PDF and writes it to cfg.Output.
func PDF(ctx context.Context, cfg board.Config) error {
	theme, err := assets.LoadTheme(cfg.Theme)
	if err != nil {
		return fmt.Errorf("loading theme: %w", err)
	}

	dims := pageSizes[cfg.PageSize]
	fpdfSize := cfg.PageSize
	if cfg.PageSize == "letter" {
		fpdfSize = "Letter"
	} else {
		fpdfSize = "A4"
	}

	pdf := fpdf.New("P", "mm", fpdfSize, "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	w := dims.width
	h := dims.height

	// Zone heights derived from layout fractions.
	headerH := h * board.HeaderFraction
	nameH := h * board.NameFraction
	tokenH := h * board.TokenFraction
	footerH := h * board.FooterFraction

	// Zone Y origins.
	headerY := 0.0
	nameY := headerH
	tokenY := headerH + nameH
	footerY := tokenY + tokenH

	// --- Header ---
	drawRect(pdf, 0, headerY, w, headerH, theme.HeaderBg, theme.HeaderBg)

	// Title on the left.
	pdf.SetFont("Helvetica", "B", 18)
	setTextColor(pdf, theme.HeaderText)
	pdf.SetXY(10, headerY+10)
	pdf.CellFormat(w/2-15, 10, cfg.Title, "", 0, "LM", false, 0, "")

	// Reward on the right: image then text, or just text.
	rewardX := w/2 + 5
	rewardW := w/2 - 15
	if cfg.RewardImage != "" {
		imgErr := placeImage(pdf, cfg.RewardImage, rewardX, headerY+5, rewardW, headerH-10, false)
		if imgErr != nil {
			// Fall back to text if image fails.
			pdf.SetXY(rewardX, headerY+10)
			pdf.SetFont("Helvetica", "B", 16)
			pdf.CellFormat(rewardW, 10, cfg.RewardText, "", 0, "CM", false, 0, "")
		}
	} else {
		pdf.SetFont("Helvetica", "B", 22)
		setTextColor(pdf, theme.HeaderText)
		pdf.SetXY(rewardX, headerY+headerH/2-11)
		pdf.CellFormat(rewardW, 22, cfg.RewardText, "", 0, "CM", false, 0, "")
	}

	// Divider line.
	setDrawColor(pdf, theme.TokenBorder)
	pdf.SetLineWidth(0.5)
	pdf.Line(0, nameY, w, nameY)

	// --- Name band ---
	if cfg.ChildName != "" {
		drawRect(pdf, 0, nameY, w, nameH, theme.NameBg, theme.NameBg)
		pdf.SetFont("Helvetica", "B", 20)
		setTextColor(pdf, theme.NameText)
		pdf.SetXY(0, nameY+nameH/2-10)
		pdf.CellFormat(w, 20, cfg.ChildName, "", 0, "CM", false, 0, "")
	}

	// Line below name band.
	pdf.Line(0, tokenY, w, tokenY)

	// --- Token row ---
	drawRect(pdf, 0, tokenY, w, tokenH, theme.TokenBg, theme.TokenBg)

	margin := 10.0
	totalTokenW := w - 2*margin
	n := float64(cfg.TokenCount)
	gap := 4.0
	slotW := (totalTokenW - gap*(n-1)) / n
	slotH := tokenH - 20
	if slotH > slotW {
		slotH = slotW // keep slots square-ish
	}
	slotY := tokenY + (tokenH-slotH)/2

	for i := 0; i < cfg.TokenCount; i++ {
		slotX := margin + float64(i)*(slotW+gap)
		// Draw rounded rectangle slot.
		setFillColor(pdf, theme.TokenBg)
		setDrawColor(pdf, theme.TokenBorder)
		pdf.SetLineWidth(1.0)
		pdf.RoundedRect(slotX, slotY, slotW, slotH, 4, "1234", "FD")

		// Draw token centered inside slot with padding.
		pad := 4.0
		if err := drawToken(pdf, cfg.TokenStyle, slotX+pad, slotY+pad, slotW-2*pad, slotH-2*pad, theme); err != nil {
			// Non-fatal: leave slot empty on token draw error.
			_ = err
		}
	}

	// --- Footer ---
	pdf.Line(0, footerY, w, footerY)
	drawRect(pdf, 0, footerY, w, footerH, theme.FooterBg, theme.FooterBg)
	drawFooterBorder(pdf, 0, footerY, w, footerH, theme.FooterBorder)

	if err := pdf.OutputFileAndClose(cfg.Output); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}
	return nil
}

// drawToken dispatches to the appropriate renderer for the given style.
// x, y, w, h are the bounding box in mm.
func drawToken(pdf *fpdf.Fpdf, style string, x, y, w, h float64, theme assets.Theme) error {
	switch {
	case board.IsBuiltinStyle(style):
		return drawBuiltinToken(pdf, style, x, y, w, h, theme)
	case board.IsPNGAssetStyle(style):
		name := board.PNGAssetName(style)
		data, err := assets.TokenPNG(name)
		if err != nil {
			return fmt.Errorf("loading embedded token PNG %q: %w", name, err)
		}
		return drawPNGTokenFromBytes(pdf, data, name, x, y, w, h)
	default:
		// Treat style as a disk path.
		return placeImage(pdf, style, x, y, w, h, true)
	}
}

// drawBuiltinToken renders a token using fpdf primitives.
func drawBuiltinToken(pdf *fpdf.Fpdf, style string, x, y, w, h float64, theme assets.Theme) error {
	cx := x + w/2
	cy := y + h/2
	r := math.Min(w, h) / 2

	setFillColor(pdf, theme.TokenFill)
	setDrawColor(pdf, theme.TokenBorder)
	pdf.SetLineWidth(0.5)

	switch style {
	case "circle":
		pdf.Circle(cx, cy, r, "FD")

	case "star":
		drawStarPrimitive(pdf, cx, cy, r)

	case "smiley":
		// Face.
		pdf.Circle(cx, cy, r, "FD")
		// Eyes.
		pdf.SetFillColor(60, 40, 0)
		eyeR := r * 0.12
		pdf.Circle(cx-r*0.3, cy-r*0.25, eyeR, "F")
		pdf.Circle(cx+r*0.3, cy-r*0.25, eyeR, "F")
		// Smile arc (approximated with a Bézier curve).
		setDrawColor(pdf, "#3C2800")
		pdf.SetLineWidth(r * 0.08)
		drawSmileArc(pdf, cx, cy, r)

	case "thumbsup":
		// Simplified thumb using rectangles and a circle.
		setFillColor(pdf, theme.TokenFill)
		palmX := cx - w*0.25
		palmY := cy - h*0.05
		palmW := w * 0.5
		palmH := h * 0.45
		pdf.Rect(palmX, palmY, palmW, palmH, "FD")
		// Thumb shaft.
		thumbX := cx
		thumbY := cy - h*0.42
		thumbW := w * 0.22
		thumbH := h * 0.42
		pdf.Rect(thumbX, thumbY, thumbW, thumbH, "FD")
		// Thumb tip.
		pdf.Circle(thumbX+thumbW/2, thumbY, thumbW/2, "FD")
	}
	return nil
}

// drawStarPrimitive draws a 5-pointed star using fpdf polygon.
func drawStarPrimitive(pdf *fpdf.Fpdf, cx, cy, outerR float64) {
	inner := outerR * 0.4
	pts := make([]fpdf.PointType, 10)
	for i := 0; i < 10; i++ {
		angle := math.Pi/2 + float64(i)*math.Pi/5.0
		r := outerR
		if i%2 == 1 {
			r = inner
		}
		pts[i] = fpdf.PointType{
			X: cx + r*math.Cos(angle),
			Y: cy - r*math.Sin(angle),
		}
	}
	pdf.Polygon(pts, "FD")
}

// drawSmileArc draws a smile arc using fpdf.Arc centered below the face center.
func drawSmileArc(pdf *fpdf.Fpdf, cx, cy, r float64) {
	// Arc at cx, cy+r*0.25 with rx=r*0.45, ry=r*0.35, sweeping 210°→330° (upward arc = smile).
	pdf.Arc(cx, cy+r*0.25, r*0.45, r*0.35, 0, 210, 330, "D")
}

// drawPNGTokenFromBytes writes the PNG bytes to a temp file and uses pdf.Image.
func drawPNGTokenFromBytes(pdf *fpdf.Fpdf, data []byte, name string, x, y, w, h float64) error {
	tmp, err := os.CreateTemp("", "token_"+name+"_*.png")
	if err != nil {
		return fmt.Errorf("creating temp file for token PNG: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing token PNG temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing token PNG temp file: %w", err)
	}
	return placeImage(pdf, tmp.Name(), x, y, w, h, true)
}

// placeImage places an image file onto the PDF, fitting it within the bounding box.
func placeImage(pdf *fpdf.Fpdf, path string, x, y, w, h float64, keepAspect bool) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("image file %q: %w", path, err)
	}
	imgW := w
	imgH := h
	if keepAspect {
		// fpdf.Image with w=0 or h=0 auto-scales; use w and let height follow.
		imgH = 0
	}
	pdf.Image(path, x, y, imgW, imgH, false, "", 0, "")
	if pdf.Err() {
		return fmt.Errorf("placing image %q: %s", path, pdf.Error())
	}
	return nil
}

// drawFooterBorder draws a decorative dashed border inside the footer zone.
func drawFooterBorder(pdf *fpdf.Fpdf, x, y, w, h float64, colorHex string) {
	setDrawColor(pdf, colorHex)
	pdf.SetLineWidth(1.5)
	pdf.SetDashPattern([]float64{4, 3}, 0)
	pad := 5.0
	pdf.Rect(x+pad, y+pad, w-2*pad, h-2*pad, "D")
	pdf.SetDashPattern([]float64{}, 0)
}

// drawRect fills a rectangle with fillHex and draws its border with borderHex.
func drawRect(pdf *fpdf.Fpdf, x, y, w, h float64, fillHex, borderHex string) {
	setFillColor(pdf, fillHex)
	setDrawColor(pdf, borderHex)
	pdf.Rect(x, y, w, h, "F")
}

// setFillColor parses a CSS hex color and sets it as the PDF fill color.
func setFillColor(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexToRGB(hex)
	pdf.SetFillColor(r, g, b)
}

// setDrawColor parses a CSS hex color and sets it as the PDF draw color.
func setDrawColor(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexToRGB(hex)
	pdf.SetDrawColor(r, g, b)
}

// setTextColor parses a CSS hex color and sets it as the PDF text color.
func setTextColor(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexToRGB(hex)
	pdf.SetTextColor(r, g, b)
}

// hexToRGB converts a "#RRGGBB" string to integer RGB components.
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	var r, g, b int
	fmt.Sscanf(hex[:2], "%x", &r)
	fmt.Sscanf(hex[2:4], "%x", &g)
	fmt.Sscanf(hex[4:6], "%x", &b)
	return r, g, b
}
