//go:build cgo

package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/tonydisco/claude-usage/internal/config"
)

// renderMenuBarIcon paints a small PNG with a horizontal battery-shape
// progress bar suitable for the macOS menu bar. The fill color follows
// the warn/alert bands (green/orange/red).
//
// We render at 2x (44px tall) so the image stays crisp on Retina; macOS
// scales it down to the menu bar height (~22px logical) automatically.
func renderMenuBarIcon(pct float64, cfg config.Config) []byte {
	const W, H = 88, 44 // 2x retina; logical ~44x22
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Battery body geometry.
	bodyX, bodyY := 4, 12
	bodyW, bodyH := 70, 20
	nipW, nipH := 4, 10
	nipX := bodyX + bodyW + 1
	nipY := bodyY + (bodyH-nipH)/2

	border := color.RGBA{60, 60, 60, 255}
	track := color.RGBA{210, 210, 210, 255}

	// Outline drawn as a slightly larger filled rect…
	rect(img, bodyX-2, bodyY-2, bodyX+bodyW+2, bodyY+bodyH+2, border)
	// …then the lighter unfilled track on top.
	rect(img, bodyX, bodyY, bodyX+bodyW, bodyY+bodyH, track)
	// Nipple on the right.
	rect(img, nipX, nipY, nipX+nipW, nipY+nipH, border)

	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fillW := int(float64(bodyW) * pct / 100.0)
	if fillW > 0 {
		rect(img, bodyX, bodyY, bodyX+fillW, bodyY+bodyH, bandColor(pct, cfg))
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func bandColor(pct float64, cfg config.Config) color.RGBA {
	switch {
	case pct >= float64(cfg.AlertThreshold):
		return color.RGBA{220, 53, 69, 255} // red
	case pct >= float64(cfg.WarnThreshold):
		return color.RGBA{255, 140, 0, 255} // orange
	default:
		return color.RGBA{40, 167, 69, 255} // green
	}
}

func rect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{C: c}, image.Point{}, draw.Src)
}
