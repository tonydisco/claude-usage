//go:build cgo

package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/tonydisco/claude-usage/internal/config"
	"github.com/tonydisco/claude-usage/internal/fetcher"
)

// renderDockIcon paints a 256x256 PNG with a battery-shape progress bar
// filled to the worst bucket's percent. The fill color follows the
// warn/alert bands (green/orange/red).
//
// 256x256 is the macOS Dock-icon design size; macOS auto-scales for
// retina and the various tile sizes.
func renderDockIcon(u *fetcher.Usage, cfg config.Config) []byte {
	const W, H = 256, 256
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Battery body geometry (centered horizontal pill with a nip on
	// the right, like the macOS battery icon).
	bodyX, bodyY := 24, 80
	bodyW, bodyH := 196, 96
	nipW, nipH := 14, 36
	nipX := bodyX + bodyW + 4
	nipY := bodyY + (bodyH-nipH)/2

	border := color.RGBA{40, 40, 40, 255}
	inside := color.RGBA{240, 240, 240, 255}

	// Outer outline drawn as a slightly larger filled rect…
	rect(img, bodyX-4, bodyY-4, bodyX+bodyW+4, bodyY+bodyH+4, border)
	// …then the lighter interior on top.
	rect(img, bodyX, bodyY, bodyX+bodyW, bodyY+bodyH, inside)
	// Nipple on the right side.
	rect(img, nipX, nipY, nipX+nipW, nipY+nipH, border)

	worst := worstBucket(u).PercentUsed
	if worst < 0 {
		worst = 0
	}
	if worst > 100 {
		worst = 100
	}
	fillW := int(float64(bodyW) * worst / 100.0)
	if fillW > 0 {
		rect(img, bodyX, bodyY, bodyX+fillW, bodyY+bodyH, bandColor(worst, cfg))
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
