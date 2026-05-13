//go:build cgo

package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/vector"

	"github.com/tonydisco/claude-usage/internal/config"
)

// renderMenuBarIcon paints an anti-aliased, rounded-rectangle battery
// icon. The fill follows the warn/alert color bands. The percentage
// text is intentionally omitted — the bar alone communicates state and
// keeps the menu-bar footprint tight.
//
// Rendered at 2x dimensions; macOS auto-downscales for the physical
// menu-bar height.
func renderMenuBarIcon(pct float64, cfg config.Config) []byte {
	// Aspect ratio matches the macOS battery: the body is roughly
	// half the canvas height with generous horizontal proportion, and
	// the canvas reserves vertical breathing room so the bar reads as
	// "short and wide" once the menu bar scales it to ~22pt height.
	const W, H = 88, 36
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Battery body — narrow horizontal pill.
	bodyX0, bodyY0 := 2.0, 10.0
	bodyX1, bodyY1 := 70.0, 26.0
	const bodyRadius = 3.5
	const borderW = 1.5

	// Nipple geometry — thin nub flush to the right.
	nipX0, nipY0 := bodyX1+1, 14.0
	nipX1, nipY1 := bodyX1+5, 22.0
	const nipRadius = 1.0

	border := color.RGBA{50, 50, 50, 255}
	track := color.RGBA{225, 225, 225, 255}

	// 1. Draw the outer body — render border + track via two rounded rects.
	rast := vector.NewRasterizer(W, H)
	addRoundedRect(rast, float32(bodyX0-borderW), float32(bodyY0-borderW),
		float32(bodyX1+borderW), float32(bodyY1+borderW), bodyRadius+borderW)
	rast.Draw(img, img.Bounds(), image.NewUniform(border), image.Point{})

	rast.Reset(W, H)
	addRoundedRect(rast, float32(bodyX0), float32(bodyY0),
		float32(bodyX1), float32(bodyY1), bodyRadius)
	rast.Draw(img, img.Bounds(), image.NewUniform(track), image.Point{})

	// 2. Nipple.
	rast.Reset(W, H)
	addRoundedRect(rast, float32(nipX0), float32(nipY0),
		float32(nipX1), float32(nipY1), nipRadius)
	rast.Draw(img, img.Bounds(), image.NewUniform(border), image.Point{})

	// 3. Fill the bar to `pct` of the body width.
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if pct > 0 {
		fillW := (bodyX1 - bodyX0) * pct / 100.0
		fillX1 := bodyX0 + fillW

		// Build the body shape as an alpha mask so the fill respects
		// the rounded outline even when fillX1 lands inside the body.
		mask := image.NewAlpha(image.Rect(0, 0, W, H))
		mrast := vector.NewRasterizer(W, H)
		addRoundedRect(mrast, float32(bodyX0), float32(bodyY0),
			float32(bodyX1), float32(bodyY1), bodyRadius)
		mrast.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})

		fillRect := image.Rect(int(bodyX0), int(bodyY0), int(fillX1)+1, int(bodyY1)+1)
		fillSrc := image.NewUniform(bandColor(pct, cfg))
		draw.DrawMask(img, fillRect, fillSrc, image.Point{}, mask, fillRect.Min, draw.Over)
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// addRoundedRect appends a rounded-rectangle subpath to rast.
func addRoundedRect(rast *vector.Rasterizer, x0, y0, x1, y1, r float32) {
	rast.MoveTo(x0+r, y0)
	rast.LineTo(x1-r, y0)
	rast.QuadTo(x1, y0, x1, y0+r)
	rast.LineTo(x1, y1-r)
	rast.QuadTo(x1, y1, x1-r, y1)
	rast.LineTo(x0+r, y1)
	rast.QuadTo(x0, y1, x0, y1-r)
	rast.LineTo(x0, y0+r)
	rast.QuadTo(x0, y0, x0+r, y0)
	rast.ClosePath()
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
