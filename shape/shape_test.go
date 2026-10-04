package shape

import (
	"math"
	"testing"
)

func TestMatchFindsTableEntries(t *testing.T) {
	for _, r := range `/\|_` {
		for _, e := range ASCII {
			if e.R == r {
				if got := Match(ASCII, e.V); got != r {
					t.Errorf("Match(vector of %q) = %q", r, got)
				}
			}
		}
	}
}

func TestEnhance(t *testing.T) {
	in := Vector{0.2, 0.8, 0.2, 0.8, 0.2, 0.8}
	var ext [10]float64

	// Exponent 1 changes nothing.
	if got := Enhance(in, ext, Options{1, 1}); got != in {
		t.Errorf("identity enhance changed %v to %v", in, got)
	}
	// Global contrast keeps the maximum and pulls the rest toward 0.
	got := Enhance(in, ext, Options{1, 2})
	if got[1] != 0.8 || got[0] >= 0.2 {
		t.Errorf("global enhance gave %v", got)
	}
	// A bright neighbor above pulls the top components down; the bottom ones,
	// which it does not affect, stay put.
	ext[0], ext[1] = 1, 1
	got = Enhance(in, ext, Options{3, 1})
	if got[0] >= 0.2 || got[1] >= 0.8 || got[4] != 0.2 || got[5] != 0.8 {
		t.Errorf("directional enhance gave %v", got)
	}
}

func TestPickUniform(t *testing.T) {
	if got := Pick(func(float64, float64) float64 { return 0 }, Default); got != ' ' {
		t.Errorf("empty cell picked %q", got)
	}
}

// TestKernelMatchesSample checks the precomputed kernel against Sample with
// the same bilinear interpolation done point by point.
func TestKernelMatchesSample(t *testing.T) {
	const sx, sy, cols, rows = 3, 6, 4, 3
	w, h := sx*cols, sy*rows
	ink := make([]float64, w*h)
	for i := range ink {
		ink[i] = float64((i*7919)%101) / 100
	}
	at := func(x, y int) float64 {
		x = min(max(x, 0), w-1)
		y = min(max(y, 0), h-1)
		return ink[y*w+x]
	}
	k := NewKernel(sx, sy)
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			gotIn, gotExt := k.Sample(ink, w, h, cx, cy)
			wantIn, wantExt := Sample(func(u, v float64) float64 {
				x := (float64(cx)+u)*sx - 0.5
				y := (float64(cy)+v)*sy - 0.5
				x0, y0 := math.Floor(x), math.Floor(y)
				fx, fy := x-x0, y-y0
				ix, iy := int(x0), int(y0)
				return at(ix, iy)*(1-fx)*(1-fy) + at(ix+1, iy)*fx*(1-fy) +
					at(ix, iy+1)*(1-fx)*fy + at(ix+1, iy+1)*fx*fy
			})
			for i := range gotIn {
				if math.Abs(gotIn[i]-wantIn[i]) > 1e-9 {
					t.Fatalf("cell %d,%d internal %d: kernel %f, sample %f", cx, cy, i, gotIn[i], wantIn[i])
				}
			}
			for i := range gotExt {
				if math.Abs(gotExt[i]-wantExt[i]) > 1e-9 {
					t.Fatalf("cell %d,%d external %d: kernel %f, sample %f", cx, cy, i, gotExt[i], wantExt[i])
				}
			}
		}
	}
}

// TestMatcherDrawsALine draws a continuous two-pixel diagonal across a grid
// of 4x8 cells and expects the matcher, cache and all, to draw the cells it
// crosses as slashes.
func TestMatcherDrawsALine(t *testing.T) {
	const sx, sy, n = 4, 8, 6
	const w, h = sx * n, sy * n
	for _, c := range []struct {
		want rune
		col  func(row int) int
		on   func(x, y float64) bool
	}{
		{'\\', func(r int) int { return r },
			func(x, y float64) bool { return math.Abs(x-y*sx/sy) < 1 }},
		{'/', func(r int) int { return n - 1 - r },
			func(x, y float64) bool { return math.Abs((w-x)-y*sx/sy) < 1 }},
	} {
		ink := make([]float64, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if c.on(float64(x)+0.5, float64(y)+0.5) {
					ink[y*w+x] = 1
				}
			}
		}
		k, m := NewKernel(sx, sy), NewMatcher(ASCII, Default)
		for pass := 0; pass < 2; pass++ { // the second pass is served from the cache
			for r := 1; r < n-1; r++ {
				if got := m.Pick(k.Sample(ink, w, h, c.col(r), r)); got != c.want {
					t.Errorf("pass %d row %d: %q, want %q", pass, r, got, c.want)
				}
			}
		}
	}
}
