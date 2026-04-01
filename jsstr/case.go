// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsstr

import (
	"sync"
	"unicode"
	"unicode/utf16"
)

const (
	capitalSigma = 0x03a3
	smallSigma   = 0x03c3
	finalSigma   = 0x03c2
)

// ToUpper returns the string with every code point replaced by its upper-case mapping, which may be more than one
// code point, as toUpperCase does.
func (s String) ToUpper() String { return s.mapCase(true) }

// ToLower returns the string with every code point replaced by its lower-case mapping, as toLowerCase does. A
// capital sigma at the end of a word becomes a final sigma.
func (s String) ToLower() String { return s.mapCase(false) }

func (s String) mapCase(upper bool) String {
	out := make(String, 0, len(s))
	for i := 0; i < len(s); {
		r, width := s.CodePointAt(i)
		if !upper && r == capitalSigma {
			out = append(out, uint16(sigmaFor(s, i)))
			i += width
			continue
		}
		if over, ok := caseOverrides[r]; ok {
			which := over[1]
			if upper {
				which = over[0]
			}
			out = append(out, FromCodePoint(which...)...)
			i += width
			continue
		}
		mapped := unicode.ToLower(r)
		if upper {
			mapped = unicode.ToUpper(r)
		}
		if r >= 0xd800 && r <= 0xdfff {
			mapped = r // an unpaired surrogate is left alone
		}
		out = append(out, FromCodePoint(mapped)...)
		i += width
	}
	return out
}

// sigmaFor decides between the final and the ordinary small sigma: a capital sigma preceded by a cased letter and
// not followed by one becomes the final form.
func sigmaFor(s String, i int) rune {
	before := false
	for j := i; j > 0; {
		r, width := previousCodePoint(s, j)
		j -= width
		if isCaseIgnorable(r) {
			continue
		}
		before = isCased(r)
		break
	}
	if !before {
		return smallSigma
	}
	for j := i + 1; j < len(s); {
		r, width := s.CodePointAt(j)
		j += width
		if isCaseIgnorable(r) {
			continue
		}
		if isCased(r) {
			return smallSigma
		}
		break
	}
	return finalSigma
}

func previousCodePoint(s String, i int) (rune, int) {
	if i <= 0 {
		return -1, 0
	}
	c := rune(s[i-1])
	if c >= 0xdc00 && c <= 0xdfff && i >= 2 && s[i-2] >= 0xd800 && s[i-2] <= 0xdbff {
		return utf16.DecodeRune(rune(s[i-2]), c), 2
	}
	return c, 1
}

func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.Is(unicode.Lt, r) ||
		unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}

func isCaseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', '^', '`', 0x00a8, 0x00ad, 0x00af, 0x00b4, 0x00b7, 0x00b8, 0x2018, 0x2019, 0x2024, 0x2027,
		0xfe13, 0xfe52, 0xfe55, 0xff07, 0xff0e, 0xff1a:
		return true
	}
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) ||
		unicode.Is(unicode.Lm, r) || unicode.Is(unicode.Sk, r)
}

// Canonicalize is the case folding a regular expression without the unicode flag applies to one code unit: the
// upper-case mapping, unless it is not a single code unit, or unless it turns a character outside Latin-1 into an
// ASCII one.
func Canonicalize(c uint16) uint16 {
	up := String{c}.ToUpper()
	if len(up) != 1 {
		return c
	}
	if c >= 128 && up[0] < 128 {
		return c
	}
	return up[0]
}

var (
	classOnce  sync.Once
	caseClass  map[uint16][]uint16
	foldedOnce sync.Once
)

// CaseClass returns every code unit that canonicalizes to the same value as c, including c itself.
func CaseClass(c uint16) []uint16 {
	classOnce.Do(func() {
		caseClass = make(map[uint16][]uint16, 1<<11)
		for u := 0; u <= 0xffff; u++ {
			canon := Canonicalize(uint16(u))
			if canon != uint16(u) {
				caseClass[canon] = append(caseClass[canon], uint16(u))
			}
		}
	})
	canon := Canonicalize(c)
	out := []uint16{c}
	if canon != c {
		out = append(out, canon)
	}
	for _, other := range caseClass[canon] {
		if other != c {
			out = append(out, other)
		}
	}
	return out
}

// FoldClass returns every code point equivalent to r under simple case folding, the folding a regular expression
// with the unicode flag applies.
func FoldClass(r rune) []rune {
	foldedOnce.Do(func() {})
	out := []rune{r}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		out = append(out, f)
	}
	return out
}
