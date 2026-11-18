// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imagediff

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

const (
	// DefaultColorThreshold is the default YIQ perceptual sensitivity (0.0 strict .. 1.0 loose).
	DefaultColorThreshold = 0.1
	// DefaultMaxDiffPixelRatio is the default maximum fraction of differing pixels allowed before flagging a visual diff.
	DefaultMaxDiffPixelRatio = 0.001

	maxYIQDelta         = 35215.0
	compositeStripePx   = 6
	compositeDividerPx  = 4
	aaMinSiblingCount   = 3
	backgroundFadeAlpha = 0.15
)

var (
	diffHighlightColor = color.NRGBA{R: 255, G: 0, B: 140, A: 255}
	aaHighlightColor   = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
	oobHighlightColor  = color.NRGBA{R: 255, G: 110, B: 0, A: 255}
	dividerColor       = color.NRGBA{R: 210, G: 214, B: 220, A: 255}
	beforeStripeColor  = color.NRGBA{R: 100, G: 116, B: 139, A: 255}
	diffStripeColor    = color.NRGBA{R: 219, G: 39, B: 119, A: 255}
	afterStripeColor   = color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	canvasBackdrop     = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
)

// Options configures perceptual PNG comparison sensitivity and output generation.
type Options struct {
	// ColorThreshold sets YIQ color distance sensitivity in [0, 1] (0 uses DefaultColorThreshold).
	ColorThreshold float64
	// IncludeAntialiasing counts anti-aliased edge pixels as diffs when true; by default anti-aliased fringes are ignored.
	IncludeAntialiasing bool
	// MaxDiffPixelRatio sets the maximum ratio of differing pixels allowed while still reporting Matched=true (negative disables tolerance).
	MaxDiffPixelRatio float64
}

// Result summarizes the outcome of comparing two PNG screenshots.
type Result struct {
	// Matched reports whether the images match within the configured threshold.
	Matched bool
	// IdenticalHash reports whether both PNG byte slices had identical SHA-256 digests.
	IdenticalHash bool
	// SizeMismatch reports whether the two images differed in width or height.
	SizeMismatch bool
	// DiffPixels is the count of pixels exceeding the perceptual threshold.
	DiffPixels int
	// TotalPixels is the total pixel count in the comparison bounding box.
	TotalPixels int
	// DiffRatio is DiffPixels divided by TotalPixels.
	DiffRatio float64
	// BeforeWidth is the width of the baseline image in pixels.
	BeforeWidth int
	// BeforeHeight is the height of the baseline image in pixels.
	BeforeHeight int
	// AfterWidth is the width of the candidate image in pixels.
	AfterWidth int
	// AfterHeight is the height of the candidate image in pixels.
	AfterHeight int
	// DiffPNG contains the encoded pixel-diff visualization when Matched is false.
	DiffPNG []byte
	// CompositePNG contains the 3-panel Before | Diff | After strip PNG when Matched is false.
	CompositePNG []byte
}

type pixelDiffConfig struct {
	maxDelta float64
	opts     Options
}

type neighborhoodBox struct {
	x0, x2 int
	y0, y2 int
	border int
}

type extremaPoint struct {
	delta float64
	x, y  int
}

// ComparePNGs compares beforePNG and afterPNG using YIQ perceptual distance and generates diff visuals when mismatched.
func ComparePNGs(beforePNG []byte, afterPNG []byte, opts Options) (Result, error) {
	if sha256.Sum256(beforePNG) == sha256.Sum256(afterPNG) {
		cfg, err := png.DecodeConfig(bytes.NewReader(afterPNG))
		if err != nil {
			return Result{}, fmt.Errorf("failed to decode identical PNG header: %w", err)
		}
		return Result{
			Matched:       true,
			IdenticalHash: true,
			TotalPixels:   cfg.Width * cfg.Height,
			BeforeWidth:   cfg.Width,
			BeforeHeight:  cfg.Height,
			AfterWidth:    cfg.Width,
			AfterHeight:   cfg.Height,
		}, nil
	}

	beforeImg, err := decodeNRGBA(beforePNG)
	if err != nil {
		return Result{}, fmt.Errorf("failed to decode baseline PNG: %w", err)
	}
	afterImg, err := decodeNRGBA(afterPNG)
	if err != nil {
		return Result{}, fmt.Errorf("failed to decode candidate PNG: %w", err)
	}

	return compareImages(beforeImg, afterImg, resolveOptions(opts))
}

func resolveOptions(opts Options) Options {
	if opts.ColorThreshold <= 0 {
		opts.ColorThreshold = DefaultColorThreshold
	}
	if opts.MaxDiffPixelRatio < 0 {
		opts.MaxDiffPixelRatio = 0
	} else if opts.MaxDiffPixelRatio == 0 {
		opts.MaxDiffPixelRatio = DefaultMaxDiffPixelRatio
	}
	return opts
}

func decodeNRGBA(data []byte) (*image.NRGBA, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return toNRGBA(img), nil
}

func toNRGBA(img image.Image) *image.NRGBA {
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba
	}
	bounds := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)
	return dst
}

func compareImages(beforeImg *image.NRGBA, afterImg *image.NRGBA, opts Options) (Result, error) {
	bw, bh := beforeImg.Bounds().Dx(), beforeImg.Bounds().Dy()
	aw, ah := afterImg.Bounds().Dx(), afterImg.Bounds().Dy()
	width := max(bw, aw)
	height := max(bh, ah)
	totalPixels := width * height
	sizeMismatch := bw != aw || bh != ah

	diffImg := image.NewNRGBA(image.Rect(0, 0, width, height))
	maxDelta := maxYIQDelta * opts.ColorThreshold * opts.ColorThreshold
	diffPixels := scanPixelDiffs(beforeImg, afterImg, diffImg, maxDelta, opts)

	diffRatio := 0.0
	if totalPixels > 0 {
		diffRatio = float64(diffPixels) / float64(totalPixels)
	}
	matched := !sizeMismatch && diffRatio <= opts.MaxDiffPixelRatio

	res := Result{
		Matched:      matched,
		SizeMismatch: sizeMismatch,
		DiffPixels:   diffPixels,
		TotalPixels:  totalPixels,
		DiffRatio:    diffRatio,
		BeforeWidth:  bw,
		BeforeHeight: bh,
		AfterWidth:   aw,
		AfterHeight:  ah,
	}
	if matched {
		return res, nil
	}

	diffBytes, err := encodePNG(diffImg)
	if err != nil {
		return Result{}, fmt.Errorf("failed to encode diff PNG: %w", err)
	}
	compositeImg := renderCompositeStrip(beforeImg, diffImg, afterImg)
	compositeBytes, err := encodePNG(compositeImg)
	if err != nil {
		return Result{}, fmt.Errorf("failed to encode composite PNG: %w", err)
	}
	res.DiffPNG = diffBytes
	res.CompositePNG = compositeBytes
	return res, nil
}

func scanPixelDiffs(beforeImg *image.NRGBA, afterImg *image.NRGBA, diffImg *image.NRGBA, maxDelta float64, opts Options) int {
	width := diffImg.Bounds().Dx()
	height := diffImg.Bounds().Dy()
	cfg := pixelDiffConfig{maxDelta: maxDelta, opts: opts}
	diffPixels := 0

	for y := range height {
		for x := range width {
			px, isDiff := classifyPixelDiff(beforeImg, afterImg, image.Pt(x, y), cfg)
			diffImg.SetNRGBA(x, y, px)
			if isDiff {
				diffPixels++
			}
		}
	}
	return diffPixels
}

func classifyPixelDiff(beforeImg *image.NRGBA, afterImg *image.NRGBA, pt image.Point, cfg pixelDiffConfig) (color.NRGBA, bool) {
	if !pixelInBounds(beforeImg, pt.X, pt.Y) || !pixelInBounds(afterImg, pt.X, pt.Y) {
		return oobHighlightColor, true
	}
	c1 := beforeImg.NRGBAAt(pt.X, pt.Y)
	c2 := afterImg.NRGBAAt(pt.X, pt.Y)
	if yiqDelta(c1, c2) <= cfg.maxDelta {
		return fadeGrayscale(c1), false
	}
	if !cfg.opts.IncludeAntialiasing && (isAntialiased(beforeImg, pt.X, pt.Y, afterImg) || isAntialiased(afterImg, pt.X, pt.Y, beforeImg)) {
		return aaHighlightColor, false
	}
	return diffHighlightColor, true
}

func pixelInBounds(img *image.NRGBA, x int, y int) bool {
	b := img.Bounds()
	return x >= 0 && y >= 0 && x < b.Dx() && y < b.Dy()
}

func yiqDelta(c1 color.NRGBA, c2 color.NRGBA) float64 {
	if c1 == c2 {
		return 0
	}
	r1, g1, b1 := blendOnWhite(c1)
	r2, g2, b2 := blendOnWhite(c2)
	dr := r1 - r2
	dg := g1 - g2
	db := b1 - b2
	y := dr*0.29889531 + dg*0.58662247 + db*0.11448223
	i := dr*0.59597799 - dg*0.27417610 - db*0.32180189
	q := dr*0.21147017 - dg*0.52261711 + db*0.31114694
	return 0.5053*y*y + 0.299*i*i + 0.1957*q*q
}

func yOnlyDelta(c1 color.NRGBA, c2 color.NRGBA) float64 {
	if c1 == c2 {
		return 0
	}
	r1, g1, b1 := blendOnWhite(c1)
	r2, g2, b2 := blendOnWhite(c2)
	return (r1-r2)*0.29889531 + (g1-g2)*0.58662247 + (b1-b2)*0.11448223
}

func blendOnWhite(c color.NRGBA) (float64, float64, float64) {
	if c.A == 255 {
		return float64(c.R), float64(c.G), float64(c.B)
	}
	alpha := float64(c.A) / 255.0
	inv := 255.0 * (1.0 - alpha)
	return inv + float64(c.R)*alpha, inv + float64(c.G)*alpha, inv + float64(c.B)*alpha
}

func computeNeighborhood(img *image.NRGBA, x int, y int) neighborhoodBox {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	box := neighborhoodBox{
		x0: max(x-1, 0),
		x2: min(x+1, width-1),
		y0: max(y-1, 0),
		y2: min(y+1, height-1),
	}
	if x == box.x0 || x == box.x2 || y == box.y0 || y == box.y2 {
		box.border = 1
	}
	return box
}

func isAntialiased(img *image.NRGBA, x int, y int, other *image.NRGBA) bool {
	minPt, maxPt, ok := findNeighborLuminanceExtrema(img, x, y)
	if !ok || minPt.delta == 0 || maxPt.delta == 0 {
		return false
	}
	return (hasManySiblings(img, minPt.x, minPt.y) && hasManySiblings(other, minPt.x, minPt.y)) ||
		(hasManySiblings(img, maxPt.x, maxPt.y) && hasManySiblings(other, maxPt.x, maxPt.y))
}

func findNeighborLuminanceExtrema(img *image.NRGBA, x int, y int) (extremaPoint, extremaPoint, bool) {
	box := computeNeighborhood(img, x, y)
	center := img.NRGBAAt(x, y)
	zeroes := box.border
	var minPt, maxPt extremaPoint

	for ny := box.y0; ny <= box.y2; ny++ {
		for nx := box.x0; nx <= box.x2; nx++ {
			if nx == x && ny == y {
				continue
			}
			delta := yOnlyDelta(center, img.NRGBAAt(nx, ny))
			switch {
			case delta == 0:
				zeroes++
				if zeroes > 2 {
					return minPt, maxPt, false
				}
			case delta < minPt.delta:
				minPt = extremaPoint{delta: delta, x: nx, y: ny}
			case delta > maxPt.delta:
				maxPt = extremaPoint{delta: delta, x: nx, y: ny}
			}
		}
	}
	return minPt, maxPt, true
}

func hasManySiblings(img *image.NRGBA, x int, y int) bool {
	if !pixelInBounds(img, x, y) {
		return false
	}
	box := computeNeighborhood(img, x, y)
	target := img.NRGBAAt(x, y)
	zeroes := box.border
	for ny := box.y0; ny <= box.y2; ny++ {
		for nx := box.x0; nx <= box.x2; nx++ {
			if nx == x && ny == y {
				continue
			}
			if img.NRGBAAt(nx, ny) == target {
				zeroes++
			}
		}
	}
	return zeroes >= aaMinSiblingCount
}

func fadeGrayscale(c color.NRGBA) color.NRGBA {
	r, g, b := blendOnWhite(c)
	luma := 0.29889531*r + 0.58662247*g + 0.11448223*b
	val := uint8(math.Round(255.0 + (luma-255.0)*backgroundFadeAlpha))
	return color.NRGBA{R: val, G: val, B: val, A: 255}
}

func renderCompositeStrip(beforeImg *image.NRGBA, diffImg *image.NRGBA, afterImg *image.NRGBA) *image.NRGBA {
	panelW := max(beforeImg.Bounds().Dx(), afterImg.Bounds().Dx(), diffImg.Bounds().Dx())
	panelH := max(beforeImg.Bounds().Dy(), afterImg.Bounds().Dy(), diffImg.Bounds().Dy())
	totalW := panelW*3 + compositeDividerPx*2
	totalH := panelH + compositeStripePx

	out := image.NewNRGBA(image.Rect(0, 0, totalW, totalH))
	draw.Draw(out, out.Bounds(), image.NewUniform(canvasBackdrop), image.Point{}, draw.Src)

	panels := []struct {
		img    *image.NRGBA
		stripe color.NRGBA
		x0     int
	}{
		{img: beforeImg, stripe: beforeStripeColor, x0: 0},
		{img: diffImg, stripe: diffStripeColor, x0: panelW + compositeDividerPx},
		{img: afterImg, stripe: afterStripeColor, x0: panelW*2 + compositeDividerPx*2},
	}

	for _, p := range panels {
		stripeRect := image.Rect(p.x0, 0, p.x0+panelW, compositeStripePx)
		draw.Draw(out, stripeRect, image.NewUniform(p.stripe), image.Point{}, draw.Src)
		dstRect := image.Rect(p.x0, compositeStripePx, p.x0+p.img.Bounds().Dx(), compositeStripePx+p.img.Bounds().Dy())
		draw.Draw(out, dstRect, p.img, image.Point{}, draw.Src)
	}

	div1 := image.Rect(panelW, 0, panelW+compositeDividerPx, totalH)
	div2 := image.Rect(panelW*2+compositeDividerPx, 0, panelW*2+compositeDividerPx*2, totalH)
	draw.Draw(out, div1, image.NewUniform(dividerColor), image.Point{}, draw.Src)
	draw.Draw(out, div2, image.NewUniform(dividerColor), image.Point{}, draw.Src)
	return out
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
