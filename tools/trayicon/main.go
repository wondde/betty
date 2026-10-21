// Draws the 64×64 macOS menu bar template icon (black + alpha; macOS tints it).
// A berry with a leaf. Run: go run ./tools/trayicon > assets/tray-template.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const size = 64
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	// coverage with 4×4 supersampling for smooth edges
	cover := func(inside func(x, y float64) bool, px, py int) float64 {
		n := 0
		for sy := 0; sy < 4; sy++ {
			for sx := 0; sx < 4; sx++ {
				if inside(float64(px)+(float64(sx)+0.5)/4, float64(py)+(float64(sy)+0.5)/4) {
					n++
				}
			}
		}
		return float64(n) / 16
	}
	berry := func(x, y float64) bool { return math.Hypot(x-32, y-37) <= 21 }
	hole := func(x, y float64) bool { return math.Hypot(x-25, y-31) <= 5.5 } // highlight cut-out
	leaf := func(x, y float64) bool {
		// ellipse rotated -35°, centred above-right of the berry
		cx, cy, a := 39.0, 12.0, -35*math.Pi/180
		dx, dy := x-cx, y-cy
		rx := dx*math.Cos(a) - dy*math.Sin(a)
		ry := dx*math.Sin(a) + dy*math.Cos(a)
		return (rx*rx)/(11*11)+(ry*ry)/(5*5) <= 1
	}
	stem := func(x, y float64) bool { return x >= 30 && x <= 34 && y >= 9 && y <= 18 }
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			c := math.Max(cover(func(x, y float64) bool { return berry(x, y) && !hole(x, y) }, px, py),
				math.Max(cover(leaf, px, py), cover(stem, px, py)))
			img.SetNRGBA(px, py, color.NRGBA{0, 0, 0, uint8(c * 255)})
		}
	}
	if err := png.Encode(os.Stdout, img); err != nil {
		panic(err)
	}
}
