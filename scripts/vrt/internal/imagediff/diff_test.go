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
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func makeSolidPNG(t *testing.T, w int, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	return buf.Bytes()
}

func makeRectDiffPNG(t *testing.T, w int, h int, bg color.NRGBA, fg color.NRGBA, rect image.Rectangle) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if image.Pt(x, y).In(rect) {
				img.SetNRGBA(x, y, fg)
			} else {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	return buf.Bytes()
}

func TestComparePNGs(t *testing.T) {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	nearWhite := color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	red := color.NRGBA{R: 220, G: 30, B: 30, A: 255}
	translucentRed := color.NRGBA{R: 220, G: 30, B: 30, A: 128}

	basePNG := makeSolidPNG(t, 20, 20, white)
	nearPNG := makeSolidPNG(t, 20, 20, nearWhite)
	redBoxPNG := makeRectDiffPNG(t, 20, 20, white, red, image.Rect(4, 4, 14, 14))
	alphaBoxPNG := makeRectDiffPNG(t, 20, 20, white, translucentRed, image.Rect(4, 4, 14, 14))
	biggerPNG := makeSolidPNG(t, 24, 20, white)

	type summary struct {
		Matched        bool
		IdenticalHash  bool
		SizeMismatch   bool
		DiffPixels     int
		TotalPixels    int
		BeforeWidth    int
		BeforeHeight   int
		AfterWidth     int
		AfterHeight    int
		HasDiffPNG     bool
		HasComposite   bool
		CompositeWidth int
	}

	testCases := []struct {
		name   string
		before []byte
		after  []byte
		opts   Options
		want   summary
	}{
		{
			name:   "identical bytes take fast hash path",
			before: basePNG,
			after:  basePNG,
			want: summary{
				Matched:       true,
				IdenticalHash: true,
				TotalPixels:   400,
				BeforeWidth:   20,
				BeforeHeight:  20,
				AfterWidth:    20,
				AfterHeight:   20,
			},
		},
		{
			name:   "imperceptible 1-step rgb change matches within threshold",
			before: basePNG,
			after:  nearPNG,
			want: summary{
				Matched:      true,
				TotalPixels:  400,
				BeforeWidth:  20,
				BeforeHeight: 20,
				AfterWidth:   20,
				AfterHeight:  20,
			},
		},
		{
			name:   "significant color block reports diff and generates diff and composite PNGs",
			before: basePNG,
			after:  redBoxPNG,
			opts:   Options{IncludeAntialiasing: true},
			want: summary{
				Matched:        false,
				DiffPixels:     100,
				TotalPixels:    400,
				BeforeWidth:    20,
				BeforeHeight:   20,
				AfterWidth:     20,
				AfterHeight:    20,
				HasDiffPNG:     true,
				HasComposite:   true,
				CompositeWidth: 20*3 + compositeDividerPx*2,
			},
		},
		{
			name:   "semi-transparent pixels blend on white before perceptual comparison",
			before: basePNG,
			after:  alphaBoxPNG,
			opts:   Options{IncludeAntialiasing: true, MaxDiffPixelRatio: -1},
			want: summary{
				Matched:        false,
				DiffPixels:     100,
				TotalPixels:    400,
				BeforeWidth:    20,
				BeforeHeight:   20,
				AfterWidth:     20,
				AfterHeight:    20,
				HasDiffPNG:     true,
				HasComposite:   true,
				CompositeWidth: 20*3 + compositeDividerPx*2,
			},
		},
		{
			name:   "different dimensions always mismatch and count out-of-bounds pixels",
			before: basePNG,
			after:  biggerPNG,
			want: summary{
				Matched:        false,
				SizeMismatch:   true,
				DiffPixels:     80,
				TotalPixels:    480,
				BeforeWidth:    20,
				BeforeHeight:   20,
				AfterWidth:     24,
				AfterHeight:    20,
				HasDiffPNG:     true,
				HasComposite:   true,
				CompositeWidth: 24*3 + compositeDividerPx*2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ComparePNGs(tc.before, tc.after, tc.opts)
			if err != nil {
				t.Fatalf("ComparePNGs() unexpected error = %v", err)
			}
			compositeWidth := 0
			if len(res.CompositePNG) > 0 {
				cfg, err := png.DecodeConfig(bytes.NewReader(res.CompositePNG))
				if err != nil {
					t.Fatalf("failed to decode composite PNG header: %v", err)
				}
				compositeWidth = cfg.Width
			}
			got := summary{
				Matched:        res.Matched,
				IdenticalHash:  res.IdenticalHash,
				SizeMismatch:   res.SizeMismatch,
				DiffPixels:     res.DiffPixels,
				TotalPixels:    res.TotalPixels,
				BeforeWidth:    res.BeforeWidth,
				BeforeHeight:   res.BeforeHeight,
				AfterWidth:     res.AfterWidth,
				AfterHeight:    res.AfterHeight,
				HasDiffPNG:     len(res.DiffPNG) > 0,
				HasComposite:   len(res.CompositePNG) > 0,
				CompositeWidth: compositeWidth,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ComparePNGs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestComparePNGs_InvalidInput(t *testing.T) {
	valid := makeSolidPNG(t, 4, 4, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	invalid := []byte("not-a-png")

	testCases := []struct {
		name   string
		before []byte
		after  []byte
	}{
		{name: "identical invalid bytes", before: invalid, after: invalid},
		{name: "invalid baseline", before: invalid, after: valid},
		{name: "invalid candidate", before: valid, after: invalid},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ComparePNGs(tc.before, tc.after, Options{}); err == nil {
				t.Errorf("ComparePNGs() expected error, got nil")
			}
		})
	}
}

func TestComparePNGs_AntialiasingAndTolerance(t *testing.T) {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	midGray := color.NRGBA{R: 128, G: 128, B: 128, A: 255}

	// Build a sharp diagonal edge where a single intermediate anti-aliased gray pixel shifts.
	makeEdge := func(aaColor color.NRGBA) []byte {
		img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
		for y := range 10 {
			for x := range 10 {
				if x < y {
					img.SetNRGBA(x, y, black)
				} else {
					img.SetNRGBA(x, y, white)
				}
			}
		}
		img.SetNRGBA(5, 5, aaColor)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("png.Encode failed: %v", err)
		}
		return buf.Bytes()
	}

	beforeEdge := makeEdge(midGray)
	afterEdge := makeEdge(white)

	// With default IncludeAntialiasing=false, the anti-aliased edge pixel is ignored.
	resIgnoreAA, err := ComparePNGs(beforeEdge, afterEdge, Options{MaxDiffPixelRatio: -1})
	if err != nil {
		t.Fatalf("ComparePNGs(ignoreAA) error = %v", err)
	}
	if !resIgnoreAA.Matched || resIgnoreAA.DiffPixels != 0 {
		t.Errorf("ComparePNGs(ignoreAA) = (matched=%v, diff=%d), want (true, 0)", resIgnoreAA.Matched, resIgnoreAA.DiffPixels)
	}

	// With IncludeAntialiasing=true and strict tolerance, the pixel is flagged.
	resStrict, err := ComparePNGs(beforeEdge, afterEdge, Options{IncludeAntialiasing: true, MaxDiffPixelRatio: -1})
	if err != nil {
		t.Fatalf("ComparePNGs(strict) error = %v", err)
	}
	if resStrict.Matched || resStrict.DiffPixels != 1 {
		t.Errorf("ComparePNGs(strict) = (matched=%v, diff=%d), want (false, 1)", resStrict.Matched, resStrict.DiffPixels)
	}

	// With explicit MaxDiffPixelRatio=0.05 (5%), 1/100 differing pixel passes.
	resTolerant, err := ComparePNGs(beforeEdge, afterEdge, Options{IncludeAntialiasing: true, MaxDiffPixelRatio: 0.05})
	if err != nil {
		t.Fatalf("ComparePNGs(tolerant) error = %v", err)
	}
	if !resTolerant.Matched {
		t.Errorf("ComparePNGs(tolerant) Matched = false, want true (ratio=%f)", resTolerant.DiffRatio)
	}

	// Also exercise dark-to-light yiqDelta (y < 0) and out-of-bounds hasManySiblings guard.
	darkPNG := makeSolidPNG(t, 4, 4, black)
	lightPNG := makeSolidPNG(t, 4, 4, white)
	resDarkToLight, err := ComparePNGs(darkPNG, lightPNG, Options{MaxDiffPixelRatio: -1})
	if err != nil || resDarkToLight.Matched || resDarkToLight.DiffPixels != 16 {
		t.Errorf("ComparePNGs(dark->light) = (matched=%v, diff=%d, err=%v), want (false, 16, nil)",
			resDarkToLight.Matched, resDarkToLight.DiffPixels, err)
	}
	if hasManySiblings(image.NewNRGBA(image.Rect(0, 0, 2, 2)), -1, 0) {
		t.Error("hasManySiblings(out-of-bounds) = true, want false")
	}
}

func makeFilledNRGBA(w int, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestPerceptualColorMathAndOptions(t *testing.T) {
	defOpts := resolveOptions(Options{})
	if defOpts.ColorThreshold != DefaultColorThreshold || defOpts.MaxDiffPixelRatio != DefaultMaxDiffPixelRatio {
		t.Errorf("resolveOptions(empty) = %+v, want defaults", defOpts)
	}
	zeroOpts := resolveOptions(Options{MaxDiffPixelRatio: -1})
	if zeroOpts.MaxDiffPixelRatio != 0 {
		t.Errorf("resolveOptions(negative).MaxDiffPixelRatio = %v, want 0", zeroOpts.MaxDiffPixelRatio)
	}

	nrgbaSrc := makeFilledNRGBA(2, 2, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	if got := toNRGBA(nrgbaSrc); got != nrgbaSrc {
		t.Error("toNRGBA(*image.NRGBA) allocated a copy instead of returning input directly")
	}
	rgbaSrc := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rgbaSrc.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if got := toNRGBA(rgbaSrc).NRGBAAt(0, 0); got != (color.NRGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Errorf("toNRGBA(*image.RGBA).NRGBAAt(0,0) = %+v", got)
	}

	emptyRes, err := compareImages(image.NewNRGBA(image.Rect(0, 0, 0, 0)), image.NewNRGBA(image.Rect(0, 0, 0, 0)), resolveOptions(Options{}))
	if err != nil || math.IsNaN(emptyRes.DiffRatio) || emptyRes.DiffRatio != 0 {
		t.Errorf("compareImages(0x0) = (ratio=%v, err=%v), want (0, nil)", emptyRes.DiffRatio, err)
	}

	if pixelInBounds(nrgbaSrc, 0, 2) || pixelInBounds(nrgbaSrc, 2, 0) || !pixelInBounds(nrgbaSrc, 1, 1) {
		t.Error("pixelInBounds boundary check failed")
	}

	c1 := color.NRGBA{R: 200, G: 40, B: 180, A: 255}
	c2 := color.NRGBA{R: 100, G: 120, B: 120, A: 255}
	gotDelta := yiqDelta(c1, c2)
	if math.Abs(gotDelta-2513.829155) > 1e-3 {
		t.Errorf("yiqDelta() = %f, want ~2513.829155", gotDelta)
	}
	if _, isDiff := classifyPixelDiff(makeFilledNRGBA(1, 1, c1), makeFilledNRGBA(1, 1, c2), image.Pt(0, 0), pixelDiffConfig{maxDelta: gotDelta}); isDiff {
		t.Error("classifyPixelDiff at exact maxDelta = diff, want matched")
	}

	gotY := yOnlyDelta(c1, c2)
	if math.Abs(gotY-(-10.171333)) > 1e-4 {
		t.Errorf("yOnlyDelta() = %f, want ~-10.171333", gotY)
	}

	r, g, b := blendOnWhite(color.NRGBA{R: 100, G: 150, B: 200, A: 128})
	if math.Abs(r-177.196078) > 1e-3 || math.Abs(g-202.294118) > 1e-3 || math.Abs(b-227.392157) > 1e-3 {
		t.Errorf("blendOnWhite(half-alpha) = (%f, %f, %f)", r, g, b)
	}
	or, og, ob := blendOnWhite(color.NRGBA{R: 100, G: 150, B: 200, A: 255})
	if or != 100 || og != 150 || ob != 200 {
		t.Errorf("blendOnWhite(opaque) = (%f, %f, %f), want (100, 150, 200)", or, og, ob)
	}

	if got := fadeGrayscale(color.NRGBA{R: 200, G: 100, B: 250, A: 255}); got != (color.NRGBA{R: 239, G: 239, B: 239, A: 255}) {
		t.Errorf("fadeGrayscale() = %+v, want {239 239 239 255}", got)
	}
}

func TestNeighborhoodAndAntialiasingHelpers(t *testing.T) {
	img5 := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	boxCases := []struct {
		name string
		x, y int
		want neighborhoodBox
	}{
		{name: "interior", x: 2, y: 2, want: neighborhoodBox{x0: 1, x2: 3, y0: 1, y2: 3, border: 0}},
		{name: "left edge", x: 0, y: 2, want: neighborhoodBox{x0: 0, x2: 1, y0: 1, y2: 3, border: 1}},
		{name: "right edge", x: 4, y: 2, want: neighborhoodBox{x0: 3, x2: 4, y0: 1, y2: 3, border: 1}},
		{name: "top edge", x: 2, y: 0, want: neighborhoodBox{x0: 1, x2: 3, y0: 0, y2: 1, border: 1}},
		{name: "bottom edge", x: 2, y: 4, want: neighborhoodBox{x0: 1, x2: 3, y0: 3, y2: 4, border: 1}},
	}
	for _, tc := range boxCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeNeighborhood(img5, tc.x, tc.y); got != tc.want {
				t.Errorf("computeNeighborhood(%d,%d) = %+v, want %+v", tc.x, tc.y, got, tc.want)
			}
		})
	}

	mid := color.NRGBA{R: 120, G: 120, B: 120, A: 255}
	dark := color.NRGBA{R: 20, G: 20, B: 20, A: 255}
	bright := color.NRGBA{R: 230, G: 230, B: 230, A: 255}
	slightDark := color.NRGBA{R: 90, G: 90, B: 90, A: 255}

	extImg := makeFilledNRGBA(3, 3, slightDark)
	extImg.SetNRGBA(1, 1, mid)
	extImg.SetNRGBA(0, 0, mid)
	extImg.SetNRGBA(0, 1, mid) // 2 zero-delta neighbors (allowed: <= 2)
	extImg.SetNRGBA(2, 1, bright)
	extImg.SetNRGBA(1, 2, dark)

	minPt, maxPt, ok := findNeighborLuminanceExtrema(extImg, 1, 1)
	if !ok || minPt.x != 2 || minPt.y != 1 || maxPt.x != 1 || maxPt.y != 2 {
		t.Errorf("findNeighborLuminanceExtrema(2 zeros) = (min=%+v, max=%+v, ok=%v), want (min=(2,1), max=(1,2), true)", minPt, maxPt, ok)
	}
	extImg.SetNRGBA(0, 2, mid) // 3rd zero-delta neighbor -> must reject (> 2)
	if _, _, ok := findNeighborLuminanceExtrema(extImg, 1, 1); ok {
		t.Error("findNeighborLuminanceExtrema(3 zeros) ok = true, want false")
	}

	sibRowImg := makeFilledNRGBA(3, 3, dark)
	sibRowImg.SetNRGBA(1, 1, mid)
	sibRowImg.SetNRGBA(0, 1, mid)
	sibRowImg.SetNRGBA(2, 1, mid)
	sibRowImg.SetNRGBA(2, 2, mid) // exact 3 siblings (2 on same row)
	if !hasManySiblings(sibRowImg, 1, 1) {
		t.Error("hasManySiblings(3 siblings, row pair) = false, want true")
	}
	sibRowImg.SetNRGBA(2, 2, bright) // down to 2 siblings
	if hasManySiblings(sibRowImg, 1, 1) {
		t.Error("hasManySiblings(2 siblings) = true, want false")
	}

	sibColImg := makeFilledNRGBA(3, 3, dark)
	sibColImg.SetNRGBA(1, 1, mid)
	sibColImg.SetNRGBA(1, 0, mid)
	sibColImg.SetNRGBA(1, 2, mid)
	sibColImg.SetNRGBA(2, 2, mid) // exact 3 siblings (2 on same column)
	if !hasManySiblings(sibColImg, 1, 1) {
		t.Error("hasManySiblings(3 siblings, col pair) = false, want true")
	}

	// Verify isAntialiased branches: monotonic extrema return false, and min-extrema must have siblings in BOTH images.
	onlyBrighter := makeFilledNRGBA(3, 3, bright)
	onlyBrighter.SetNRGBA(1, 1, dark)
	if isAntialiased(onlyBrighter, 1, 1, onlyBrighter) {
		t.Error("isAntialiased(only brighter) = true, want false")
	}
	onlyDarker := makeFilledNRGBA(3, 3, dark)
	onlyDarker.SetNRGBA(1, 1, bright)
	if isAntialiased(onlyDarker, 1, 1, onlyDarker) {
		t.Error("isAntialiased(only darker) = true, want false")
	}

	// Build a 5x5 image where (2,2) has dark maxPt at (1,1) surrounded by >=3 dark siblings, and isolated bright minPt at (3,3).
	aaBase := makeFilledNRGBA(5, 5, slightDark)
	for _, p := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		aaBase.SetNRGBA(p[0], p[1], dark)
	}
	aaBase.SetNRGBA(2, 2, mid)
	aaBase.SetNRGBA(3, 3, bright)
	aaOtherMissing := makeFilledNRGBA(5, 5, slightDark)
	aaOtherMissing.SetNRGBA(1, 1, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	aaOtherMissing.SetNRGBA(3, 3, color.NRGBA{R: 4, G: 5, B: 6, A: 255})
	if isAntialiased(aaBase, 2, 2, aaOtherMissing) {
		t.Error("isAntialiased(siblings only in img, not other) = true, want false")
	}
	if !isAntialiased(aaBase, 2, 2, aaBase) {
		t.Error("isAntialiased(maxPt has siblings in both, minPt does not) = false, want true")
	}

	// Also verify minPt-only siblings case (bright cluster at (3,3), isolated dark at (1,1)).
	aaMinOnly := makeFilledNRGBA(5, 5, slightDark)
	for _, p := range [][2]int{{3, 3}, {4, 3}, {3, 4}, {4, 4}} {
		aaMinOnly.SetNRGBA(p[0], p[1], bright)
	}
	aaMinOnly.SetNRGBA(2, 2, mid)
	aaMinOnly.SetNRGBA(1, 1, dark)
	if isAntialiased(aaMinOnly, 2, 2, aaOtherMissing) {
		t.Error("isAntialiased(minPt siblings only in img, not other) = true, want false")
	}
	if !isAntialiased(aaMinOnly, 2, 2, aaMinOnly) {
		t.Error("isAntialiased(minPt has siblings in both, maxPt does not) = false, want true")
	}
}

func TestRenderCompositeStripLayout(t *testing.T) {
	red := color.NRGBA{R: 200, G: 10, B: 10, A: 255}
	green := color.NRGBA{R: 10, G: 200, B: 10, A: 255}
	blue := color.NRGBA{R: 10, G: 10, B: 200, A: 255}

	strip := renderCompositeStrip(makeFilledNRGBA(6, 4, red), makeFilledNRGBA(6, 4, green), makeFilledNRGBA(6, 4, blue))
	if gotW, gotH := strip.Bounds().Dx(), strip.Bounds().Dy(); gotW != 26 || gotH != 10 {
		t.Fatalf("strip size = %dx%d, want 26x10", gotW, gotH)
	}

	wantPixels := map[image.Point]color.NRGBA{
		image.Pt(0, 0):  beforeStripeColor,
		image.Pt(5, 5):  beforeStripeColor,
		image.Pt(6, 0):  dividerColor,
		image.Pt(9, 9):  dividerColor,
		image.Pt(10, 0): diffStripeColor,
		image.Pt(15, 5): diffStripeColor,
		image.Pt(16, 0): dividerColor,
		image.Pt(19, 9): dividerColor,
		image.Pt(20, 0): afterStripeColor,
		image.Pt(25, 5): afterStripeColor,
		image.Pt(0, 6):  red,
		image.Pt(5, 9):  red,
		image.Pt(10, 6): green,
		image.Pt(15, 9): green,
		image.Pt(20, 6): blue,
		image.Pt(25, 9): blue,
	}
	gotPixels := make(map[image.Point]color.NRGBA, len(wantPixels))
	for pt := range wantPixels {
		gotPixels[pt] = strip.NRGBAAt(pt.X, pt.Y)
	}
	if diff := cmp.Diff(wantPixels, gotPixels); diff != "" {
		t.Errorf("composite strip pixels mismatch (-want +got):\n%s", diff)
	}
}
