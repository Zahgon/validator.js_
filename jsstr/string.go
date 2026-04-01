// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

// Package jsstr holds the JavaScript string the validators are written against: a sequence of UTF-16 code units,
// with the String methods and the number conversions the library relies on. Lengths, indexes and slices count code
// units, not code points or bytes, so a character outside the Basic Multilingual Plane counts as two.
package jsstr

import (
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// String is a JavaScript string: a sequence of UTF-16 code units, which may include unpaired surrogates.
type String []uint16

// Line terminators and the characters String.prototype.trim removes.
var (
	// LineTerminators are the code points the specification treats as ending a line.
	LineTerminators = []uint16{0x000a, 0x000d, 0x2028, 0x2029}
	// Whitespace is the WhiteSpace production: the characters trim removes besides the line terminators.
	Whitespace = []uint16{
		0x0009, 0x000b, 0x000c, 0x0020, 0x00a0, 0xfeff, 0x1680,
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200a,
		0x202f, 0x205f, 0x3000,
	}
)

// Of converts UTF-8 text to a string, as a JavaScript runtime does when it decodes UTF-8 input: a byte sequence
// that is not valid UTF-8 becomes the replacement character.
func Of(s string) String {
	out := make(String, 0, len(s))
	for _, r := range s {
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			out = append(out, uint16(hi), uint16(lo))
			continue
		}
		out = append(out, uint16(r))
	}
	return out
}

// Go converts the string to UTF-8, replacing every unpaired surrogate with the replacement character, as writing
// the string to a UTF-8 stream does.
func (s String) Go() string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := rune(s[i])
		switch {
		case c >= 0xd800 && c <= 0xdbff && i+1 < len(s) && s[i+1] >= 0xdc00 && s[i+1] <= 0xdfff:
			b.WriteRune(utf16.DecodeRune(c, rune(s[i+1])))
			i++
		case c >= 0xd800 && c <= 0xdfff:
			b.WriteRune(utf8.RuneError)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// Len is the number of code units, the value of the string's length property.
func (s String) Len() int { return len(s) }

// Clone returns a copy that shares no storage with s.
func (s String) Clone() String { return append(String(nil), s...) }

// CodeUnit returns the code unit at i, and false when i is outside the string.
func (s String) CodeUnit(i int) (uint16, bool) {
	if i < 0 || i >= len(s) {
		return 0, false
	}
	return s[i], true
}

// CodePointAt returns the code point starting at i and the number of code units it occupies, as codePointAt does.
func (s String) CodePointAt(i int) (rune, int) {
	if i < 0 || i >= len(s) {
		return -1, 0
	}
	c := rune(s[i])
	if c >= 0xd800 && c <= 0xdbff && i+1 < len(s) && s[i+1] >= 0xdc00 && s[i+1] <= 0xdfff {
		return utf16.DecodeRune(c, rune(s[i+1])), 2
	}
	return c, 1
}

// Equal reports whether the two strings hold the same code units.
func (s String) Equal(t String) bool {
	if len(s) != len(t) {
		return false
	}
	for i := range s {
		if s[i] != t[i] {
			return false
		}
	}
	return true
}

// Is reports whether the string equals the UTF-8 literal.
func (s String) Is(lit string) bool { return s.Equal(Of(lit)) }

// Slice returns the half-open range of code units, with the negative and out-of-range indexes of slice.
func (s String) Slice(start, end int) String {
	n := len(s)
	start = relative(start, n)
	end = relative(end, n)
	if start >= end {
		return String{}
	}
	return s[start:end:end]
}

// From returns the code units from start on, with the negative and out-of-range indexes of slice.
func (s String) From(start int) String { return s.Slice(start, len(s)) }

func relative(i, n int) int {
	if i < 0 {
		if i += n; i < 0 {
			return 0
		}
		return i
	}
	if i > n {
		return n
	}
	return i
}

// Substring returns the code units between the two indexes, clamping them into the string and swapping them when
// they are the wrong way round, as substring does.
func (s String) Substring(start, end int) String {
	n := len(s)
	start = clamp(start, n)
	end = clamp(end, n)
	if start > end {
		start, end = end, start
	}
	return s[start:end:end]
}

func clamp(i, n int) int {
	if i < 0 || i != i {
		return 0
	}
	if i > n {
		return n
	}
	return i
}

// CharAt returns the one code unit at i, or the empty string when i is outside the string.
func (s String) CharAt(i int) String {
	if i < 0 || i >= len(s) {
		return String{}
	}
	return s[i : i+1 : i+1]
}

// IndexOf returns the first index at or after from where sub occurs, or -1.
func (s String) IndexOf(sub String, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i : i+len(sub)].Equal(sub) {
			return i
		}
	}
	return -1
}

// Index returns the first index where the UTF-8 literal occurs, or -1.
func (s String) Index(lit string) int { return s.IndexOf(Of(lit), 0) }

// LastIndexOf returns the last index where sub occurs, or -1.
func (s String) LastIndexOf(sub String) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i : i+len(sub)].Equal(sub) {
			return i
		}
	}
	return -1
}

// Contains reports whether the UTF-8 literal occurs in the string.
func (s String) Contains(lit string) bool { return s.Index(lit) >= 0 }

// Includes reports whether sub occurs in the string.
func (s String) Includes(sub String) bool { return s.IndexOf(sub, 0) >= 0 }

// HasPrefix reports whether the string starts with the UTF-8 literal.
func (s String) HasPrefix(lit string) bool {
	p := Of(lit)
	return len(s) >= len(p) && s[:len(p)].Equal(p)
}

// HasSuffix reports whether the string ends with the UTF-8 literal.
func (s String) HasSuffix(lit string) bool {
	p := Of(lit)
	return len(s) >= len(p) && s[len(s)-len(p):].Equal(p)
}

// StartsWith reports whether the code units at from are sub.
func (s String) StartsWith(sub String, from int) bool {
	if from < 0 {
		from = 0
	}
	return from+len(sub) <= len(s) && s[from:from+len(sub)].Equal(sub)
}

// EndsWith reports whether the string ends with sub.
func (s String) EndsWith(sub String) bool {
	return len(s) >= len(sub) && s[len(s)-len(sub):].Equal(sub)
}

// Concat joins the strings, as the + operator does.
func Concat(parts ...String) String {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make(String, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// Repeat returns the string repeated n times.
func (s String) Repeat(n int) String {
	out := make(String, 0, len(s)*max(n, 0))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return out
}

// PadStart pads the string to length n with pad, as padStart does.
func (s String) PadStart(n int, pad String) String {
	if len(s) >= n || len(pad) == 0 {
		return s
	}
	fill := make(String, 0, n-len(s))
	for len(fill) < n-len(s) {
		fill = append(fill, pad...)
	}
	return Concat(fill[:n-len(s)], s)
}

// Split splits the string on the literal separator. The empty separator splits it into its code units; a separator
// that does not occur gives the whole string back as the only part.
func (s String) Split(sep string) []String {
	return s.SplitStr(Of(sep))
}

// SplitStr splits the string on sep, which may be any string.
func (s String) SplitStr(sep String) []String {
	if len(sep) == 0 {
		out := make([]String, len(s))
		for i := range s {
			out[i] = s[i : i+1 : i+1]
		}
		return out
	}
	out := []String{}
	start := 0
	for {
		i := s.IndexOf(sep, start)
		if i < 0 {
			break
		}
		out = append(out, s[start:i:i])
		start = i + len(sep)
	}
	return append(out, s[start:])
}

// Join joins the parts with the UTF-8 literal between them.
func Join(parts []String, sep string) String {
	s := Of(sep)
	out := String{}
	for i, p := range parts {
		if i > 0 {
			out = append(out, s...)
		}
		out = append(out, p...)
	}
	return out
}

// Reverse returns the code units in the opposite order, as splitting, reversing and joining does.
func (s String) Reverse() String {
	out := make(String, len(s))
	for i, c := range s {
		out[len(s)-1-i] = c
	}
	return out
}

// FromCharCode builds a string out of code units.
func FromCharCode(units ...uint16) String { return String(units) }

// FromCodePoint builds a string out of code points.
func FromCodePoint(points ...rune) String {
	out := String{}
	for _, r := range points {
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			out = append(out, uint16(hi), uint16(lo))
			continue
		}
		out = append(out, uint16(r))
	}
	return out
}

// IsTrimmable reports whether the code unit is white space or a line terminator, the characters trim removes.
func IsTrimmable(c uint16) bool {
	return contains(Whitespace, c) || contains(LineTerminators, c)
}

// IsLineTerminator reports whether the code unit ends a line.
func IsLineTerminator(c uint16) bool { return contains(LineTerminators, c) }

func contains(set []uint16, c uint16) bool {
	for _, u := range set {
		if u == c {
			return true
		}
	}
	return false
}

// Trim removes white space and line terminators from both ends.
func (s String) Trim() String { return s.TrimStart().TrimEnd() }

// TrimStart removes white space and line terminators from the front.
func (s String) TrimStart() String {
	i := 0
	for i < len(s) && IsTrimmable(s[i]) {
		i++
	}
	return s[i:]
}

// TrimEnd removes white space and line terminators from the end.
func (s String) TrimEnd() String {
	i := len(s)
	for i > 0 && IsTrimmable(s[i-1]) {
		i--
	}
	return s[:i:i]
}

// Chars returns the code units as one-unit strings, as split("") does.
func (s String) Chars() []String { return s.Split("") }

// NaN is the not-a-number value the conversions return for text that is not a number.
var NaN = math.NaN()

// ReplaceText replaces the first occurrence of the search text, as replace does when it is handed a string
// rather than a pattern. A dollar sign in the replacement keeps its meaning.
func (s String) ReplaceText(search, replacement String) String {
	i := s.IndexOf(search, 0)
	if i < 0 {
		return s
	}
	out := Concat(s[:i], replacement, s[i+len(search):])
	return out
}
