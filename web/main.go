//go:build js && wasm

// Command web is the img2txt-go demo: an image in, colored character art out.
//
// It runs the same pipeline cmd/img2txt does — decode, size the canvas to the
// image's aspect against a 6x10 font, dither, export — so what the page shows
// is what the binary prints. Drop in your own image or use the generated one;
// nothing leaves the tab, because there is no server to send it to.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"strconv"
	"syscall/js"

	"github.com/0magnet/img2txt-go/caca"
)

// The font img2txt assumes, and so the aspect correction it applies.
const fontWidth, fontHeight = 6, 10

var current *caca.Image

func main() {
	doc := js.Global().Get("document")
	file := doc.Call("getElementById", "file")
	cols := doc.Call("getElementById", "cols")
	colsOut := doc.Call("getElementById", "colsout")
	algo := doc.Call("getElementById", "algo")
	format := doc.Call("getElementById", "format")
	out := doc.Call("getElementById", "out")
	raw := doc.Call("getElementById", "raw")

	fill := func(sel js.Value, list [][2]string, def string) {
		for _, e := range list {
			o := doc.Call("createElement", "option")
			o.Set("value", e[0])
			o.Set("textContent", e[0]+" — "+e[1])
			if e[0] == def {
				o.Set("selected", true)
			}
			sel.Call("appendChild", o)
		}
	}
	fill(algo, caca.AlgorithmList(), "fstein")
	// Only the formats that mean something in a browser; the rest (tga, ps,
	// troff, caca) are files to save, not things to look at.
	fill(format, [][2]string{
		{"html", "HTML"},
		{"ansi", "ANSI"},
		{"utf8", "UTF-8 with ANSI escape codes"},
		{"irc", "IRC color codes"},
		{"bbfr", "BBCode"},
		{"svg", "SVG"},
	}, "html")

	render := func() {
		if current == nil {
			return
		}
		c, _ := strconv.Atoi(cols.Get("value").String())
		if c < 8 {
			c = 8
		}
		colsOut.Set("textContent", strconv.Itoa(c))
		lines := int(int64(c) * int64(current.H) * fontWidth / int64(current.W) / fontHeight)
		if lines < 1 {
			lines = 1
		}

		cv := caca.NewCanvas(c, lines)
		cv.SetColorANSI(caca.Default, caca.Transparent)
		cv.Clear()
		if !current.Dither.SetAlgorithm(algo.Get("value").String()) {
			current.Dither.SetAlgorithm("fstein") //nolint:errcheck
		}
		cv.DitherBitmap(0, 0, c, lines, current.Dither, current.Pixels)

		f := format.Get("value").String()
		b, ok := cv.Export(f)
		if !ok {
			raw.Set("textContent", "cannot export to "+f)
			return
		}
		raw.Set("textContent", string(b))
		// html and svg export whole documents, not fragments — a doctype and
		// an <html> element cannot be dropped into a div, so they are shown
		// in an iframe, which is the thing they actually are.
		if f == "html" || f == "svg" {
			out.Get("style").Set("display", "")
			out.Set("srcdoc", string(b))
		} else {
			out.Get("style").Set("display", "none")
		}
	}

	load := func(b []byte) {
		im, err := caca.DecodeImage(bytes.NewReader(b))
		if err != nil {
			raw.Set("textContent", "cannot decode image: "+err.Error())
			return
		}
		current = im
		render()
	}

	// A file the visitor picked. FileReader hands back an ArrayBuffer, which
	// copies into Go as bytes — the image never leaves the tab.
	file.Call("addEventListener", "change", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		files := file.Get("files")
		if files.Length() == 0 {
			return nil
		}
		fr := js.Global().Get("FileReader").New()
		fr.Call("addEventListener", "load", js.FuncOf(func(js.Value, []js.Value) any {
			buf := fr.Get("result")
			u8 := js.Global().Get("Uint8Array").New(buf)
			b := make([]byte, u8.Get("length").Int())
			js.CopyBytesToGo(b, u8)
			load(b)
			return nil
		}))
		fr.Call("readAsArrayBuffer", files.Index(0))
		return nil
	}))

	cb := js.FuncOf(func(js.Value, []js.Value) any { render(); return nil })
	cols.Call("addEventListener", "input", cb)
	algo.Call("addEventListener", "change", cb)
	format.Call("addEventListener", "change", cb)

	if msg := doc.Call("getElementById", "msg"); msg.Truthy() {
		msg.Call("remove")
	}
	load(sample())
	select {}
}

// sample draws the image the page starts with, rather than shipping one. A
// Mandelbrot set has smooth gradients and hard edges in the same frame, which
// is what makes the difference between dithering algorithms visible.
func sample() []byte {
	const w, h = 480, 360
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := complex(float64(x)/w*3.0-2.1, float64(y)/h*2.4-1.2)
			z := complex(0, 0)
			var i int
			for ; i < 60; i++ {
				z = z*z + c
				if cmplx.Abs(z) > 2 {
					break
				}
			}
			if i == 60 {
				img.Set(x, y, color.NRGBA{0, 0, 0, 255})
				continue
			}
			// Dark blue through cyan to yellow and white: four widely spaced
			// hues, so the sixteen-color palette has something to choose
			// between and the bands stay legible as characters.
			t := math.Sqrt(float64(i) / 60)
			switch {
			case t < 0.4:
				u := t / 0.4
				img.Set(x, y, color.NRGBA{0, uint8(120 * u), uint8(60 + 195*u), 255})
			case t < 0.75:
				u := (t - 0.4) / 0.35
				img.Set(x, y, color.NRGBA{uint8(230 * u), uint8(120 + 135*u), uint8(255 - 155*u), 255})
			default:
				u := (t - 0.75) / 0.25
				img.Set(x, y, color.NRGBA{255, 255, uint8(100 + 155*u), 255})
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img) //nolint:errcheck // an in-memory encode of a valid image
	return buf.Bytes()
}
