package caca

import "math/rand"

// UTF32ToCP437 converts a Unicode code point to its CP437 byte, or '?' when it
// has no CP437 representation.
func UTF32ToCP437(ch rune) byte {
	if ch < 0x00000020 {
		return '?'
	}
	if ch < 0x00000080 {
		return byte(ch)
	}
	for i, r := range cp437Lookup1 {
		if r == ch {
			return byte(0x01 + i)
		}
	}
	for i, r := range cp437Lookup2 {
		if r == ch {
			return byte(0x7f + i)
		}
	}
	return '?'
}

// CP437ToUTF32 converts a CP437 byte to Unicode, or zero for control codes
// with no representation.
func CP437ToUTF32(ch byte) rune {
	if ch > 0x7f {
		return cp437Lookup2[int(ch)-0x7f]
	}
	if ch >= 0x20 {
		return rune(ch)
	}
	if ch > 0 {
		return cp437Lookup1[int(ch)-1]
	}
	return 0
}

// UTF32ToUTF8 appends the UTF-8 encoding of ch to dst. It follows libcaca's
// encoder, which allows the historical five and six byte forms.
func UTF32ToUTF8(dst []byte, ch rune) []byte {
	u := uint32(ch)
	switch {
	case u < 0x80:
		return append(dst, byte(u))
	case u < 0x800:
		return append(dst, byte(0xc0|(u>>6)), byte(0x80|(u&0x3f)))
	case u < 0x10000:
		return append(dst, byte(0xe0|(u>>12)), byte(0x80|((u>>6)&0x3f)),
			byte(0x80|(u&0x3f)))
	case u < 0x200000:
		return append(dst, byte(0xf0|(u>>18)), byte(0x80|((u>>12)&0x3f)),
			byte(0x80|((u>>6)&0x3f)), byte(0x80|(u&0x3f)))
	case u < 0x4000000:
		return append(dst, byte(0xf8|(u>>24)), byte(0x80|((u>>18)&0x3f)),
			byte(0x80|((u>>12)&0x3f)), byte(0x80|((u>>6)&0x3f)),
			byte(0x80|(u&0x3f)))
	default:
		return append(dst, byte(0xfc|(u>>30)), byte(0x80|((u>>24)&0x3f)),
			byte(0x80|((u>>18)&0x3f)), byte(0x80|((u>>12)&0x3f)),
			byte(0x80|((u>>6)&0x3f)), byte(0x80|(u&0x3f)))
	}
}

// IsFullwidth reports whether a code point occupies two cells. The ranges are
// those libcaca uses.
func IsFullwidth(ch rune) bool {
	return (ch >= 0x1100 && ch <= 0x115f) || // Hangul Jamo
		(ch >= 0x2e80 && ch <= 0xa4cf && ch != 0x303f) || // CJK, Yi
		(ch >= 0xac00 && ch <= 0xd7a3) || // Hangul syllables
		(ch >= 0xf900 && ch <= 0xfaff) || // CJK compatibility ideographs
		(ch >= 0xfe30 && ch <= 0xfe6f) || // CJK compatibility forms
		(ch >= 0xff00 && ch <= 0xff60) || // fullwidth forms
		(ch >= 0xffe0 && ch <= 0xffe6) ||
		(ch >= 0x20000 && ch <= 0x2fffd) ||
		(ch >= 0x30000 && ch <= 0x3fffd)
}

// rngState backs the "random" dithering algorithm. libcaca seeds its generator
// from the process id and a timer, so random dithering is not reproducible
// there either.
type rngState struct{ r *rand.Rand }

func newRNG() *rngState {
	return &rngState{r: rand.New(rand.NewSource(rand.Int63()))}
}

// rand returns a value in [min, max).
func (s *rngState) rand(min, max int32) int32 {
	if max <= min {
		return min
	}
	return min + int32(float64(max-min)*s.r.Float64())
}
