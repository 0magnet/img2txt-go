package caca

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0magnet/img2txt-go/shape"
)

var update = flag.Bool("update", false, "rewrite the shape-mode golden files")

// renderFile runs the img2txt pipeline on testdata/shapes.png 40 columns
// wide, the way cmd/img2txt sizes it, and exports it.
func renderFile(t *testing.T, algo string, shaped bool, format string) []byte {
	t.Helper()
	im, err := LoadImage(filepath.Join("testdata", "shapes.png"))
	if err != nil {
		t.Fatal(err)
	}
	const cols = 40
	lines := cols * im.H * 6 / im.W / 10
	im.Dither.SetShape(shaped)
	cv := render(t, im, cols, lines, algo)
	out, ok := cv.Export(format)
	if !ok {
		t.Fatalf("export %s", format)
	}
	return out
}

// TestDefaultMatchesLibcaca pins the default output to the system img2txt
// (libcaca 0.99.beta20): the libcaca-* files are its output for the same
// command line, and the TGA hash is the SHA-256 of `img2txt -W 40 -f tga`.
func TestDefaultMatchesLibcaca(t *testing.T) {
	for _, algo := range []string{"fstein", "none", "ordered4"} {
		for _, format := range []string{"ansi", "utf8"} {
			want, err := os.ReadFile(filepath.Join("testdata", "libcaca-"+algo+"."+format)) //nolint:gosec // a test fixture
			if err != nil {
				t.Fatal(err)
			}
			if got := renderFile(t, algo, false, format); !bytes.Equal(got, want) {
				t.Errorf("-d %s -f %s differs from libcaca", algo, format)
			}
		}
	}
	sum := sha256.Sum256(renderFile(t, "fstein", false, "tga"))
	if got := hex.EncodeToString(sum[:]); got != "0c782f1cddbf2e03c637939ba628621d34899aa804db4d5c215eb934abca40dd" {
		t.Errorf("tga sha256 %s differs from libcaca", got)
	}
}

// TestShapeTableMatchesFont proves the committed table is what the font
// measures to, so it cannot drift from the generator.
func TestShapeTableMatchesFont(t *testing.T) {
	f, err := BuiltinFont("Monospace 9")
	if err != nil {
		t.Fatal(err)
	}
	got := ShapeEntries(f)
	if len(got) != len(shape.ASCII) || len(got) != 95 {
		t.Fatalf("%d glyphs measured, %d in the table, want 95", len(got), len(shape.ASCII))
	}
	for i, e := range got {
		w := shape.ASCII[i]
		if e.R != w.R {
			t.Fatalf("entry %d: %q, table has %q", i, e.R, w.R)
		}
		for j := range e.V {
			if math.Abs(e.V[j]-w.V[j]) > 1e-6 {
				t.Errorf("%q component %d: measured %f, table %f; run go generate ./shape", e.R, j, e.V[j], w.V[j])
			}
		}
	}
}

// synth draws white on black at 120x200, which a 20x20 canvas divides into
// 6x10 cells, and decodes it as an Image.
func synth(t *testing.T, on func(x, y float64) bool) *Image {
	t.Helper()
	src := image.NewNRGBA(image.Rect(0, 0, 120, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 120; x++ {
			c := color.NRGBA{0, 0, 0, 255}
			if on(float64(x)+0.5, float64(y)+0.5) {
				c = color.NRGBA{255, 255, 255, 255}
			}
			src.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	im, err := DecodeImage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	return im
}

// TestShapeFollowsEdges checks that shape mode draws edges with glyphs of
// the matching orientation where brightness mode, given the same cells,
// cannot: it only knows how much ink a cell needs, not where.
func TestShapeFollowsEdges(t *testing.T) {
	cases := []struct {
		name   string
		on     func(x, y float64) bool
		cell   func(row int) int // the column the edge crosses in this row
		family string
	}{
		{"diagonal line \\",
			func(x, y float64) bool { return math.Abs(y*0.6-x) < 1.2 },
			func(r int) int { return r }, `\`},
		{"diagonal line /",
			func(x, y float64) bool { return math.Abs(y*0.6-(120-x)) < 1.2 },
			func(r int) int { return 19 - r }, `/`},
		{"vertical line",
			func(x, _ float64) bool { return x > 62 && x < 64 },
			func(int) int { return 10 }, `|!lI1`},
	}
	for _, c := range cases {
		for _, algo := range []string{"none", "fstein"} {
			for _, shaped := range []bool{true, false} {
				im := synth(t, c.on)
				im.Dither.SetShape(shaped)
				cv := render(t, im, 20, 20, algo)
				hits := 0
				for r := 2; r < 18; r++ {
					if strings.ContainsRune(c.family, cv.Chars[r*20+c.cell(r)]) {
						hits++
					}
				}
				switch {
				case shaped && hits < 16:
					t.Errorf("%s, -d %s: shape mode drew %d of 16 cells from %q", c.name, algo, hits, c.family)
				case !shaped && hits > 0:
					t.Errorf("%s, -d %s: brightness mode drew %d cells from %q", c.name, algo, hits, c.family)
				}
			}
		}
	}

	// A horizontal edge 70% of the way down a row of cells: the light part
	// sits at the bottom of each cell, which shape mode draws as '_'.
	im := synth(t, func(_, y float64) bool { return y > 107 })
	im.Dither.SetShape(true)
	cv := render(t, im, 20, 20, "none")
	if row := string(cv.Chars[10*20 : 11*20]); row != strings.Repeat("_", 20) {
		t.Errorf("horizontal edge drew %q", row)
	}
}

// TestShapeGolden pins shape mode's output for one image, so a change to
// the geometry, the table or the enhancement shows up as a diff. Rewrite it
// with go test ./caca -run TestShapeGolden -update.
func TestShapeGolden(t *testing.T) {
	got := renderFile(t, "fstein", true, "utf8")
	path := filepath.Join("testdata", "shape-fstein.utf8")
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // a test fixture
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path) //nolint:gosec // a test fixture
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("shape output differs from %s", path)
	}
	// And twice in a row is the same, so the golden is not luck.
	if again := renderFile(t, "fstein", true, "utf8"); !bytes.Equal(got, again) {
		t.Error("shape mode is not deterministic")
	}
}
