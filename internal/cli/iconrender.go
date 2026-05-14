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

// Font for the percentage digits inside the body. Bold and tightly
// hinted so it stays sharp once macOS downscales the icon to the
// menu-bar height.
var iconFont font.Face

func init() {
	tt, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return
	}
	iconFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    15,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// Apple-system color palette. Slightly more saturated than the
// previous flat web colors so the alert states pop and the warning
// state stays unmistakable against the menu bar.
var (
	colorGreen  = color.RGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF} // systemGreen
	colorOrange = color.RGBA{R: 0xFF, G: 0x95, B: 0x00, A: 0xFF} // systemOrange
	colorRed    = color.RGBA{R: 0xFF, G: 0x3B, B: 0x30, A: 0xFF} // systemRed
)

// renderMenuBarIcon paints a polished battery-shape progress indicator
// for the macOS menu bar.
//
// template=true produces a pure-black-on-transparent image. macOS tints
// it to match the active appearance (light/dark mode) via
// NSImage.isTemplate, and the percentage digits are punched out as
// transparent holes so they stay legible even when the fill bar covers
// them.
//
// template=false renders the icon with the band fill color (green /
// orange / red). The digits are drawn in white so they're crisp on top
// of the colored fill.
func renderMenuBarIcon(pct float64, cfg config.Config, template bool) []byte {
	// 2x-retina canvas matched to the menu-bar height (~22pt → 44px)
	// so the icon is not downscaled out of visual parity with native
	// indicators.
	const W, H = 64, 44
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Geometry — proportions tuned to read like the SF Symbol battery
	// glyph: short body, slim nipple, fine stroke.
	bodyX0, bodyY0 := 5.0, 11.0
	bodyX1, bodyY1 := 55.0, 33.0
	const bodyRadius = 3.5
	const borderW = 2.5

	nipX0, nipY0 := bodyX1+1.5, 17.0
	nipX1, nipY1 := bodyX1+5.5, 27.0
	const nipRadius = 1.5

	var outline, fill color.Color
	if template {
		outline = color.Black
		fill = color.Black
	} else {
		outline = color.RGBA{0x33, 0x33, 0x33, 0xFF}
		fill = bandColor(pct, cfg)
	}

	rast := vector.NewRasterizer(W, H)

	// 1. Outer rounded rect filled with the outline color, then carve
	//    out the inner body so we end up with a clean ring.
	addRoundedRect(rast, float32(bodyX0-borderW), float32(bodyY0-borderW),
		float32(bodyX1+borderW), float32(bodyY1+borderW), bodyRadius+borderW)
	rast.Draw(img, img.Bounds(), image.NewUniform(outline), image.Point{})

	innerMask := image.NewAlpha(image.Rect(0, 0, W, H))
	mrast := vector.NewRasterizer(W, H)
	addRoundedRect(mrast, float32(bodyX0), float32(bodyY0),
		float32(bodyX1), float32(bodyY1), bodyRadius)
	mrast.Draw(innerMask, innerMask.Bounds(), image.Opaque, image.Point{})
	draw.DrawMask(img, img.Bounds(), image.Transparent, image.Point{},
		innerMask, image.Point{}, draw.Src)

	// 2. Nipple — small rounded nub.
	rast.Reset(W, H)
	addRoundedRect(rast, float32(nipX0), float32(nipY0),
		float32(nipX1), float32(nipY1), nipRadius)
	rast.Draw(img, img.Bounds(), image.NewUniform(outline), image.Point{})

	// 3. Proportional fill, masked to the inner body shape so it
	//    follows the rounded outline cleanly.
	//
	// We display the REMAINING capacity, mirroring how the macOS
	// battery indicator works: a full bar means "lots of headroom"
	// and the bar drains as the user burns their plan. The color
	// band is still decided from the raw usage percentage so the
	// warn/alert thresholds keep their meaning.
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	remaining := 100.0 - pct
	if remaining > 0 {
		fillW := (bodyX1 - bodyX0) * remaining / 100.0
		fillX1 := bodyX0 + fillW
		fillRect := image.Rect(int(bodyX0), int(bodyY0), int(fillX1)+1, int(bodyY1)+1)
		draw.DrawMask(img, fillRect, image.NewUniform(fill), image.Point{},
			innerMask, fillRect.Min, draw.Over)
	}

	// 4. Centered remaining-percentage digits.
	drawCenteredDigits(img, remaining, image.Rect(int(bodyX0), int(bodyY0), int(bodyX1), int(bodyY1)), template)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func drawCenteredDigits(dst *image.RGBA, pct float64, body image.Rectangle, template bool) {
	if iconFont == nil {
		return
	}
	text := fmt.Sprintf("%.0f", pct)

	// Measure once with a throwaway drawer.
	measure := &font.Drawer{Face: iconFont}
	w := measure.MeasureString(text).Round()
	m := iconFont.Metrics()
	textH := (m.Ascent + m.Descent).Round()
	cx := (body.Min.X + body.Max.X) / 2
	cy := (body.Min.Y + body.Max.Y) / 2
	x := cx - w/2
	y := cy + textH/2 - m.Descent.Round()

	if !template {
		// Solid white digits on the colored fill.
		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.White,
			Face: iconFont,
		}
		drawer.Dot = fixed.P(x, y)
		drawer.DrawString(text)
		return
	}

	// Template path: render the text shape into an alpha mask, then
	// erase those pixels from the icon. macOS tints the rest of the
	// black image to the foreground color; the punched-out digits stay
	// transparent and show the menu bar background, which keeps them
	// crisp regardless of how full the bar is.
	bounds := dst.Bounds()
	textMask := image.NewAlpha(bounds)
	maskDrawer := &font.Drawer{
		Dst:  textMask,
		Src:  image.Opaque,
		Face: iconFont,
	}
	maskDrawer.Dot = fixed.P(x, y)
	maskDrawer.DrawString(text)

	draw.DrawMask(dst, bounds, image.Transparent, image.Point{},
		textMask, image.Point{}, draw.Src)
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
		return colorRed
	case pct >= float64(cfg.WarnThreshold):
		return colorOrange
	default:
		return colorGreen
	}
}
