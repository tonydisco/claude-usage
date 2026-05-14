//go:build cgo

package cli

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/tonydisco/claude-usage/internal/config"
)

// iconFont is reused across renders. Bold so the digits stay legible
// at the 24px-tall body height we're working with.
var iconFont font.Face

func init() {
	tt, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return
	}
	iconFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16, // image pixels at 72 DPI
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// renderMenuBarIcon paints a battery-shape progress indicator suitable
// for the macOS menu bar.
//
// When template is true the image is monochrome (pure black on
// transparent) so macOS can tint it to match the active menu-bar
// appearance (light/dark mode) via NSImage.isTemplate. This is the
// "default" look that matches stock macOS icons.
//
// When template is false the bar is filled with the warn/alert band
// color (green/orange/red) — used as a fallback on non-darwin systray
// implementations and, optionally, to break out of template tinting
// when usage crosses the warn threshold.
//
// Geometry mirrors the proportions of the macOS battery indicator:
// short and wide body, slim right-hand nipple, 1.5pt stroke outline,
// 3.5pt corner radius.
func renderMenuBarIcon(pct float64, cfg config.Config, template bool) []byte {
	// 2x-retina canvas. Image height matches the macOS menu-bar height
	// (~22pt → 44px) so the icon does not get downscaled and shrunk
	// out of visual parity with native indicators.
	const W, H = 64, 44
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Battery body. Slightly thicker outline than the native battery
	// so claude-usage stays distinguishable when it sits next to the
	// system indicator.
	bodyX0, bodyY0 := 5.0, 10.0
	bodyX1, bodyY1 := 55.0, 34.0
	const bodyRadius = 4.0
	const borderW = 3.0

	nipX0, nipY0 := bodyX1+2, 16.0
	nipX1, nipY1 := bodyX1+6, 28.0
	const nipRadius = 1.5

	var outline, fill color.Color
	if template {
		// Pure black; macOS template renderer tints both pixels and
		// alpha to the menu-bar foreground color.
		outline = color.Black
		fill = color.Black
	} else {
		outline = color.RGBA{50, 50, 50, 255}
		fill = bandColor(pct, cfg)
	}

	rast := vector.NewRasterizer(W, H)

	// 1. Outer rounded rect filled with the outline color.
	addRoundedRect(rast, float32(bodyX0-borderW), float32(bodyY0-borderW),
		float32(bodyX1+borderW), float32(bodyY1+borderW), bodyRadius+borderW)
	rast.Draw(img, img.Bounds(), image.NewUniform(outline), image.Point{})

	// 2. Carve the body interior out — replaces the inner pixels with
	//    fully-transparent so we get a stroked outline, not a filled
	//    block. The mask is the inner body shape.
	innerMask := image.NewAlpha(image.Rect(0, 0, W, H))
	mrast := vector.NewRasterizer(W, H)
	addRoundedRect(mrast, float32(bodyX0), float32(bodyY0),
		float32(bodyX1), float32(bodyY1), bodyRadius)
	mrast.Draw(innerMask, innerMask.Bounds(), image.Opaque, image.Point{})
	draw.DrawMask(img, img.Bounds(), image.Transparent, image.Point{},
		innerMask, image.Point{}, draw.Src)

	// 3. Nipple — solid rounded nub on the right.
	rast.Reset(W, H)
	addRoundedRect(rast, float32(nipX0), float32(nipY0),
		float32(nipX1), float32(nipY1), nipRadius)
	rast.Draw(img, img.Bounds(), image.NewUniform(outline), image.Point{})

	// 4. Proportional fill, masked to the inner body shape so it
	//    follows the rounded outline.
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if pct > 0 {
		fillW := (bodyX1 - bodyX0) * pct / 100.0
		fillX1 := bodyX0 + fillW
		fillRect := image.Rect(int(bodyX0), int(bodyY0), int(fillX1)+1, int(bodyY1)+1)
		draw.DrawMask(img, fillRect, image.NewUniform(fill), image.Point{},
			innerMask, fillRect.Min, draw.Over)
	}

	// 5. Centered % digits inside the body. We drop the "%" character
	//    so "100" still fits. In template mode the digits tint with
	//    everything else; in colored mode we draw them white so they
	//    stand out on the orange/red fill.
	drawCenteredDigits(img, pct, image.Rect(int(bodyX0), int(bodyY0), int(bodyX1), int(bodyY1)), template)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func drawCenteredDigits(dst *image.RGBA, pct float64, body image.Rectangle, template bool) {
	if iconFont == nil {
		return
	}
	var textSrc image.Image
	if template {
		// Same color as the outline so macOS template tinting applies
		// uniformly across the icon.
		textSrc = image.Black
	} else {
		// White on orange/red reads at any size.
		textSrc = image.White
	}
	drawer := &font.Drawer{
		Dst:  dst,
		Src:  textSrc,
		Face: iconFont,
	}
	text := fmt.Sprintf("%.0f", pct)
	w := drawer.MeasureString(text).Round()
	m := iconFont.Metrics()
	textH := (m.Ascent + m.Descent).Round()

	cx := (body.Min.X + body.Max.X) / 2
	cy := (body.Min.Y + body.Max.Y) / 2
	x := cx - w/2
	y := cy + textH/2 - m.Descent.Round()
	drawer.Dot = fixed.P(x, y)
	drawer.DrawString(text)
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
