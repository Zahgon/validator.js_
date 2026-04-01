// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsstr

import (
	"math"
	"testing"
)

func TestOfIsCodeUnits(t *testing.T) {
	// A string counts its UTF-16 code units, so an astral character counts as two.
	cases := []struct {
		in   string
		want int
	}{{"", 0}, {"a", 1}, {"é", 1}, {"中", 1}, {"\U0001f600", 2}, {"a\U0001f600b", 4}}
	for _, c := range cases {
		if got := len(Of(c.in)); got != c.want {
			t.Errorf("len(Of(%q)) = %d, want %d", c.in, got, c.want)
		}
	}
}

// sum adds at run time: Go folds a constant sum exactly, which is not how the source runtime adds.
func sum(a, b float64) float64 { return a + b }

func TestNumberToString(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{1, "1"},
		{-1, "-1"},
		{0.5, "0.5"},
		{0.1, "0.1"},
		{sum(0.1, 0.2), "0.30000000000000004"},
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},
		{1.1e21, "1.1e+21"},
		{1e-6, "0.000001"},
		{1e-7, "1e-7"},
		{5e-324, "5e-324"},
		{1.7976931348623157e308, "1.7976931348623157e+308"},
		{9007199254740993, "9007199254740992"},
		{1000000000000000128, "1000000000000000100"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, c := range cases {
		if got := NumberToString(c.in); got != c.want {
			t.Errorf("NumberToString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToNumber(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"", 0},
		{" ", 0},
		{"\t\n", 0},
		{"1", 1},
		{" 12 ", 12},
		{"0x1f", 31},
		{"0X1F", 31},
		{"0b1011", 11},
		{"0o17", 15},
		{"1_000", math.NaN()},
		{"Infinity", math.Inf(1)},
		{"-Infinity", math.Inf(-1)},
		{"+Infinity", math.Inf(1)},
		{"1e500", math.Inf(1)},
		{".5", 0.5},
		{"5.", 5},
		{"1.5e-3", 0.0015},
		{"١٢٣", math.NaN()},
		{"1,5", math.NaN()},
		{"1-", math.NaN()},
		{"0xg", math.NaN()},
	}
	for _, c := range cases {
		got := ToNumber(Of(c.in))
		if math.IsNaN(c.want) {
			if !math.IsNaN(got) {
				t.Errorf("ToNumber(%q) = %v, want NaN", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("ToNumber(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseIntAndFloat(t *testing.T) {
	intCases := []struct {
		in    string
		radix int
		want  float64
	}{
		{"12abc", 10, 12},
		{" 12", 10, 12},
		{"0x1f", 0, 31},
		{"0x1f", 16, 31},
		{"1f", 16, 31},
		{"1f", 10, 1},
		{"z", 36, 35},
		{"z", 37, math.NaN()},
		{"z", 1, math.NaN()},
		{"", 10, math.NaN()},
		{"+-1", 10, math.NaN()},
		{"-12.9", 10, -12},
		{"1e3", 10, 1},
	}
	for _, c := range intCases {
		got := ParseInt(Of(c.in), c.radix)
		if math.IsNaN(c.want) != math.IsNaN(got) || (!math.IsNaN(c.want) && got != c.want) {
			t.Errorf("ParseInt(%q, %d) = %v, want %v", c.in, c.radix, got, c.want)
		}
	}
	floatCases := []struct {
		in   string
		want float64
	}{
		{"1.5abc", 1.5},
		{" .5", 0.5},
		{"1e3", 1000},
		{"Infinity!", math.Inf(1)},
		{"", math.NaN()},
		{"abc", math.NaN()},
		{"0x10", 0},
	}
	for _, c := range floatCases {
		got := ParseFloat(Of(c.in))
		if math.IsNaN(c.want) != math.IsNaN(got) || (!math.IsNaN(c.want) && got != c.want) {
			t.Errorf("ParseFloat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNumberToStringRadix(t *testing.T) {
	cases := []struct {
		in    float64
		radix int
		want  string
	}{{255, 16, "ff"}, {255, 2, "11111111"}, {-255, 16, "-ff"}, {35, 36, "z"}, {0, 16, "0"}}
	for _, c := range cases {
		if got := NumberToStringRadix(c.in, c.radix); got != c.want {
			t.Errorf("NumberToStringRadix(%v, %d) = %q, want %q", c.in, c.radix, got, c.want)
		}
	}
}

func TestCase(t *testing.T) {
	cases := []struct{ in, lower, upper string }{
		{"abc", "abc", "ABC"},
		{"ABC", "abc", "ABC"},
		{"ß", "ß", "SS"},
		{"ﬁ", "ﬁ", "FI"},
		{"İ", "i̇", "İ"},
		{"ı", "ı", "I"},
		{"ΑΣ", "ας", "ΑΣ"},
		{"ΑΣΑ", "ασα", "ΑΣΑ"},
		{"ς", "ς", "Σ"},
		{"K", "k", "K"},
		{"\U0001f600", "\U0001f600", "\U0001f600"},
		{"\U00010428", "\U00010428", "\U00010400"},
	}
	for _, c := range cases {
		if got := Of(c.in).ToLower().Go(); got != c.lower {
			t.Errorf("ToLower(%q) = %q, want %q", c.in, got, c.lower)
		}
		if got := Of(c.in).ToUpper().Go(); got != c.upper {
			t.Errorf("ToUpper(%q) = %q, want %q", c.in, got, c.upper)
		}
	}
}

func TestTrimmable(t *testing.T) {
	for _, c := range []uint16{' ', '\t', '\n', '\r', 0x0b, 0x0c, 0xa0, 0x1680, 0x2000, 0x3000, 0xfeff} {
		if !IsTrimmable(c) {
			t.Errorf("IsTrimmable(%#04x) = false, want true", c)
		}
	}
	for _, c := range []uint16{'a', 0x200b, 0x180e} {
		if IsTrimmable(c) {
			t.Errorf("IsTrimmable(%#04x) = true, want false", c)
		}
	}
}

func TestToInt32(t *testing.T) {
	cases := []struct {
		in   float64
		want int32
	}{{0, 0}, {1.9, 1}, {-1.9, -1}, {4294967296, 0}, {4294967297, 1}, {2147483648, -2147483648},
		{math.NaN(), 0}, {math.Inf(1), 0}}
	for _, c := range cases {
		if got := ToInt32(c.in); got != c.want {
			t.Errorf("ToInt32(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
