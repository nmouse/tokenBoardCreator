// gentokens generates simple PNG token images for embedding in the binary.
// Run once with: go run ./cmd/gentokens
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

const size = 64

func main() {
	write("internal/assets/tokens/star.png", drawStar())
	write("internal/assets/tokens/smiley.png", drawSmiley())
	write("internal/assets/tokens/thumbsup.png", drawThumbsup())
}

func write(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func newRGBA() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	// transparent background
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)
	return img
}

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	if x >= 0 && x < size && y >= 0 && y < size {
		img.SetNRGBA(x, y, c)
	}
}

func drawFilledCircle(img *image.NRGBA, cx, cy, r int, c color.NRGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			if dx*dx+dy*dy <= float64(r*r) {
				setPixel(img, x, y, c)
			}
		}
	}
}

func drawLine(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		setPixel(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawStar() image.Image {
	img := newRGBA()
	gold := color.NRGBA{255, 215, 0, 255}
	cx, cy := size/2, size/2
	outer := float64(cx - 4)
	inner := outer * 0.4
	points := 5

	// Fill star polygon by converting to triangles from center.
	pts := make([][2]float64, points*2)
	for i := 0; i < points*2; i++ {
		angle := math.Pi/2 + float64(i)*math.Pi/float64(points)
		r := outer
		if i%2 == 1 {
			r = inner
		}
		pts[i] = [2]float64{
			float64(cx) + r*math.Cos(angle),
			float64(cy) - r*math.Sin(angle),
		}
	}

	// Scan-line fill: for each row, find intersections.
	for y := 0; y < size; y++ {
		var xs []float64
		n := len(pts)
		for i := 0; i < n; i++ {
			p1 := pts[i]
			p2 := pts[(i+1)%n]
			if (p1[1] <= float64(y) && p2[1] > float64(y)) ||
				(p2[1] <= float64(y) && p1[1] > float64(y)) {
				t := (float64(y) - p1[1]) / (p2[1] - p1[1])
				xs = append(xs, p1[0]+t*(p2[0]-p1[0]))
			}
		}
		// Sort xs (simple insertion sort — small slice).
		for i := 1; i < len(xs); i++ {
			for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
				xs[j-1], xs[j] = xs[j], xs[j-1]
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			for x := int(xs[i]); x <= int(xs[i+1]); x++ {
				setPixel(img, x, y, gold)
			}
		}
	}

	// Draw outline.
	outline := color.NRGBA{180, 150, 0, 255}
	n := len(pts)
	for i := 0; i < n; i++ {
		p1 := pts[i]
		p2 := pts[(i+1)%n]
		drawLine(img, int(p1[0]), int(p1[1]), int(p2[0]), int(p2[1]), outline)
	}
	return img
}

func drawSmiley() image.Image {
	img := newRGBA()
	yellow := color.NRGBA{255, 220, 50, 255}
	dark := color.NRGBA{60, 40, 0, 255}
	cx, cy := size/2, size/2
	r := size/2 - 3

	drawFilledCircle(img, cx, cy, r, yellow)

	// Eyes.
	drawFilledCircle(img, cx-8, cy-8, 4, dark)
	drawFilledCircle(img, cx+8, cy-8, 4, dark)

	// Smile arc: draw pixels along an arc.
	for deg := -150; deg <= -30; deg++ {
		rad := float64(deg) * math.Pi / 180.0
		smileR := float64(r) * 0.55
		x := cx + int(smileR*math.Cos(rad))
		y := cy + int(smileR*math.Sin(rad))*(-1) + 6
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				setPixel(img, x+dx, y+dy, dark)
			}
		}
	}
	return img
}

func drawThumbsup() image.Image {
	img := newRGBA()
	skin := color.NRGBA{255, 220, 150, 255}
	dark := color.NRGBA{180, 130, 60, 255}

	// Palm: filled rectangle.
	for y := 28; y < 56; y++ {
		for x := 16; x < 48; x++ {
			setPixel(img, x, y, skin)
		}
	}
	// Thumb: angled rectangle.
	for y := 12; y < 34; y++ {
		for x := 30; x < 46; x++ {
			setPixel(img, x, y, skin)
		}
	}
	// Thumb tip rounded.
	drawFilledCircle(img, 38, 12, 7, skin)

	// Outline.
	for y := 28; y < 56; y++ {
		setPixel(img, 16, y, dark)
		setPixel(img, 47, y, dark)
	}
	for x := 16; x < 48; x++ {
		setPixel(img, x, 55, dark)
	}
	for y := 12; y < 34; y++ {
		setPixel(img, 30, y, dark)
	}
	return img
}
