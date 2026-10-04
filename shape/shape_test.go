package shape

import "testing"

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
