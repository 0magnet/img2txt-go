// Command img2txt converts an image to any text based format libcaca supports.
//
// It is a Go port of libcaca's img2txt and accepts the same options.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/0magnet/img2txt-go/caca"
)

const version = "0.99.beta20"

func usage() {
	prog := os.Args[0]
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]... <IMAGE>\n", prog)
	fmt.Fprintf(os.Stderr, "Convert IMAGE to any text based available format.\n")
	fmt.Fprintf(os.Stderr, "Example : %s -W 80 -f ansi ./caca.png\n\n", prog)
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -h, --help\t\t\tThis help\n")
	fmt.Fprintf(os.Stderr, "  -v, --version\t\t\tVersion of the program\n")
	fmt.Fprintf(os.Stderr, "  -W, --width=WIDTH\t\tWidth of resulting image\n")
	fmt.Fprintf(os.Stderr, "  -H, --height=HEIGHT\t\tHeight of resulting image\n")
	fmt.Fprintf(os.Stderr, "  -x, --font-width=WIDTH\t\tWidth of output font\n")
	fmt.Fprintf(os.Stderr, "  -y, --font-height=HEIGHT\t\tHeight of output font\n")
	fmt.Fprintf(os.Stderr, "  -b, --brightness=BRIGHTNESS\tBrightness of resulting image\n")
	fmt.Fprintf(os.Stderr, "  -c, --contrast=CONTRAST\tContrast of resulting image\n")
	fmt.Fprintf(os.Stderr, "  -g, --gamma=GAMMA\t\tGamma of resulting image\n")
	fmt.Fprintf(os.Stderr, "  -d, --dither=DITHER\t\tDithering algorithm to use :\n")
	for _, e := range caca.AlgorithmList() {
		fmt.Fprintf(os.Stderr, "\t\t\t%s: %s\n", e[0], e[1])
	}
	fmt.Fprintf(os.Stderr, "  -f, --format=FORMAT\t\tFormat of the resulting image :\n")
	for _, e := range caca.ExportList() {
		fmt.Fprintf(os.Stderr, "\t\t\t%s: %s\n", e[0], e[1])
	}
}

func printVersion() {
	fmt.Printf(
		"img2txt Copyright 2006-2007 Sam Hocevar and Jean-Yves Lamoureux\n"+
			"Internet: <sam@hocevar.net> <jylam@lnxscene.org> Version: %s\n"+
			"\n"+
			"img2txt, along with its documentation, may be freely copied and distributed.\n"+
			"\n"+
			"The latest version of img2txt is available from the web site,\n"+
			"        http://caca.zoy.org/wiki/libcaca in the libcaca package.\n"+
			"\n", version)
}

// atoiC parses a leading integer, returning 0 when there is none, like atoi.
func atoiC(s string) int {
	s = strings.TrimSpace(s)
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return 0
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0
	}
	return v
}

// atofC parses a leading float, returning 0 when there is none, like atof.
func atofC(s string) float64 {
	s = strings.TrimSpace(s)
	end := 0
	seenDigit, seenDot, seenExp := false, false, false
	for end < len(s) {
		c := s[end]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
		case (c == '-' || c == '+') && (end == 0 || (seenExp && (s[end-1] == 'e' || s[end-1] == 'E'))):
		case c == '.' && !seenDot && !seenExp:
			seenDot = true
		case (c == 'e' || c == 'E') && seenDigit && !seenExp:
			seenExp = true
		default:
			goto done
		}
		end++
	}
done:
	if !seenDigit {
		return 0
	}
	v, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return 0
	}
	return v
}

// shortOpts maps single-letter options to whether they take an argument.
var shortOpts = map[byte]bool{
	'W': true, 'H': true, 'f': true, 'd': true, 'g': true,
	'b': true, 'c': true, 'h': false, 'v': false, 'x': true, 'y': true,
}

// longOpts maps long option names to their short equivalent.
var longOpts = map[string]byte{
	"width": 'W', "height": 'H', "font-width": 'x', "font-height": 'y',
	"format": 'f', "dither": 'd', "gamma": 'g', "brightness": 'b',
	"contrast": 'c', "help": 'h', "version": 'v',
}

type parsedOpt struct {
	code byte
	arg  string
}

// parseArgs parses the command line the way caca_getopt does.
func parseArgs(argv []string) ([]parsedOpt, bool) {
	var out []parsedOpt
	i := 0
	for i < len(argv) {
		a := argv[i]
		switch {
		case a == "--":
			return out, true
		case strings.HasPrefix(a, "--"):
			body := a[2:]
			name, val := body, ""
			hasVal := false
			if eq := strings.IndexByte(body, '='); eq >= 0 {
				name, val, hasVal = body[:eq], body[eq+1:], true
			}
			code, ok := longOpts[name]
			if !ok {
				return out, false
			}
			if shortOpts[code] && !hasVal {
				i++
				if i >= len(argv) {
					return out, false
				}
				val = argv[i]
			}
			out = append(out, parsedOpt{code, val})
			i++
		case len(a) > 1 && a[0] == '-':
			j := 1
			for j < len(a) {
				c := a[j]
				takesArg, known := shortOpts[c]
				if !known {
					return out, false
				}
				if !takesArg {
					out = append(out, parsedOpt{c, ""})
					j++
					continue
				}
				if j+1 < len(a) {
					out = append(out, parsedOpt{c, a[j+1:]})
				} else {
					i++
					if i >= len(argv) {
						return out, false
					}
					out = append(out, parsedOpt{c, argv[i]})
				}
				j = len(a)
			}
			i++
		default:
			// Operand; img2txt only uses the final argument as the file name.
			i++
		}
	}
	return out, true
}

func run() int {
	argv := os.Args[1:]
	if len(argv) < 1 {
		fmt.Fprintf(os.Stderr, "%s: wrong argument count\n", os.Args[0])
		usage()
		return 1
	}

	cols, lines := 0, 0
	fontWidth, fontHeight := 6, 10
	format := "ansi"
	dither := ""
	gamma, brightness, contrast := -1.0, -1.0, -1.0

	opts, ok := parseArgs(argv)
	if !ok {
		return 1
	}
	for _, o := range opts {
		switch o.code {
		case 'W':
			cols = atoiC(o.arg)
		case 'H':
			lines = atoiC(o.arg)
		case 'x':
			fontWidth = atoiC(o.arg)
		case 'y':
			fontHeight = atoiC(o.arg)
		case 'f':
			format = o.arg
		case 'd':
			dither = o.arg
		case 'g':
			gamma = atofC(o.arg)
		case 'b':
			brightness = atofC(o.arg)
		case 'c':
			contrast = atofC(o.arg)
		case 'h':
			usage()
			return 0
		case 'v':
			printVersion()
			return 0
		}
	}

	if fontHeight == 0 || fontWidth == 0 {
		fmt.Fprintf(os.Stderr, "%s: invalid font size %dx%d\n", os.Args[0], fontWidth, fontHeight)
		return 1
	}

	// img2txt always treats the last argument as the file name.
	name := os.Args[len(os.Args)-1]
	im, err := caca.LoadImage(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to load %s\n", os.Args[0], name)
		return 1
	}
	if im.W == 0 || im.H == 0 {
		fmt.Fprintf(os.Stderr, "%s: image %s has invalid dimensions %dx%d\n",
			os.Args[0], name, im.W, im.H)
		return 1
	}

	// Assume a 6x10 font.
	switch {
	case cols == 0 && lines == 0:
		cols = 60
		lines = int(int64(cols) * int64(im.H) * int64(fontWidth) / int64(im.W) / int64(fontHeight))
	case cols != 0 && lines == 0:
		lines = int(int64(cols) * int64(im.H) * int64(fontWidth) / int64(im.W) / int64(fontHeight))
	case cols == 0 && lines != 0:
		cols = int(int64(lines) * int64(im.W) * int64(fontHeight) / int64(im.H) / int64(fontWidth))
	}

	cv := caca.NewCanvas(cols, lines)
	cv.SetColorANSI(caca.Default, caca.Transparent)
	cv.Clear()

	algo := dither
	if algo == "" {
		algo = "fstein"
	}
	if !im.Dither.SetAlgorithm(algo) {
		fmt.Fprintf(os.Stderr, "%s: Can't dither image with algorithm '%s'\n", os.Args[0], dither)
		return -1
	}

	if brightness != -1 {
		im.Dither.SetBrightness(float32(brightness))
	}
	if contrast != -1 {
		im.Dither.SetContrast(float32(contrast))
	}
	if gamma != -1 {
		im.Dither.SetGamma(float32(gamma))
	}

	cv.DitherBitmap(0, 0, cols, lines, im.Dither, im.Pixels)

	out, ok := cv.Export(format)
	if !ok {
		fmt.Fprintf(os.Stderr, "%s: Can't export to format '%s'\n", os.Args[0], format)
		return 0
	}
	_, _ = os.Stdout.Write(out)
	return 0
}

func main() {
	os.Exit(run())
}
