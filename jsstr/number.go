// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsstr

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// ToNumber converts the string to a number the way the language does when a string is used where a number is
// wanted: surrounding white space is ignored, the empty string is zero, binary, octal and hexadecimal prefixes and
// Infinity are understood, and anything else is not a number.
func ToNumber(s String) float64 {
	t := s.Trim()
	if len(t) == 0 {
		return 0
	}
	text := t.Go()
	if len(t) > 2 && t[0] == '0' {
		var base int
		switch t[1] {
		case 'b', 'B':
			base = 2
		case 'o', 'O':
			base = 8
		case 'x', 'X':
			base = 16
		}
		if base != 0 {
			digits := text[2:]
			value, ok := new(big.Int).SetString(digits, base)
			if !ok || !validDigits(digits, base) {
				return math.NaN()
			}
			out, _ := new(big.Float).SetInt(value).Float64()
			return out
		}
	}
	if !decimalLiteral(text) {
		return math.NaN()
	}
	switch text {
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	out, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if num, ok := err.(*strconv.NumError); ok && num.Err == strconv.ErrRange {
			return out
		}
		return math.NaN()
	}
	return out
}

func validDigits(s string, base int) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if digitValue(uint16(c)) < 0 || digitValue(uint16(c)) >= base {
			return false
		}
	}
	return true
}

// decimalLiteral reports whether the text is a decimal numeric literal: the grammar accepts a leading sign,
// Infinity, digits with an optional fraction and an optional exponent, but no underscore separator and no
// trailing text.
func decimalLiteral(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	if strings.HasPrefix(s[i:], "Infinity") {
		return i+len("Infinity") == len(s)
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		exponent := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			exponent++
		}
		if exponent == 0 {
			return false
		}
	}
	return i == len(s)
}

func digitValue(c uint16) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

// ParseInt reads an integer from the front of the string in the given radix, ignoring whatever follows the digits,
// as the parseInt function does. A radix of zero means the default: sixteen after a 0x prefix and ten otherwise.
func ParseInt(s String, radix int) float64 {
	t := s.TrimStart()
	i := 0
	sign := 1
	if i < len(t) && (t[i] == '+' || t[i] == '-') {
		if t[i] == '-' {
			sign = -1
		}
		i++
	}
	stripPrefix := false
	switch {
	case radix == 0:
		radix = 10
		stripPrefix = true
	case radix == 16:
		stripPrefix = true
	case radix < 2 || radix > 36:
		return math.NaN()
	}
	if stripPrefix && i+1 < len(t) && t[i] == '0' && (t[i+1] == 'x' || t[i+1] == 'X') {
		i += 2
		radix = 16
	}
	start := i
	for i < len(t) {
		d := digitValue(t[i])
		if d < 0 || d >= radix {
			break
		}
		i++
	}
	if i == start {
		return math.NaN()
	}
	value, ok := new(big.Int).SetString(t[start:i].Go(), radix)
	if !ok {
		return math.NaN()
	}
	out, _ := new(big.Float).SetInt(value).Float64()
	return float64(sign) * out
}

// ParseFloat reads a number from the front of the string, ignoring whatever follows it, as the parseFloat function
// does.
func ParseFloat(s String) float64 {
	t := s.TrimStart().Go()
	i := 0
	if i < len(t) && (t[i] == '+' || t[i] == '-') {
		i++
	}
	if strings.HasPrefix(t[i:], "Infinity") {
		if t[0] == '-' {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	digits := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
		digits++
	}
	if i < len(t) && t[i] == '.' {
		i++
		for i < len(t) && t[i] >= '0' && t[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return math.NaN()
	}
	end := i
	if i < len(t) && (t[i] == 'e' || t[i] == 'E') {
		i++
		if i < len(t) && (t[i] == '+' || t[i] == '-') {
			i++
		}
		exponent := 0
		for i < len(t) && t[i] >= '0' && t[i] <= '9' {
			i++
			exponent++
		}
		if exponent > 0 {
			end = i
		}
	}
	out, err := strconv.ParseFloat(t[:end], 64)
	if err != nil {
		if num, ok := err.(*strconv.NumError); ok && num.Err == strconv.ErrRange {
			return out
		}
		return math.NaN()
	}
	return out
}

// NumberToString writes a number the way the language writes it: the shortest digits that read back as the same
// number, in plain form while the exponent is small and in exponential form outside that range. Negative zero
// writes as "0", as it does there.
func NumberToString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	case f < 0:
		return "-" + NumberToString(-f)
	}
	shortest := strconv.FormatFloat(f, 'e', -1, 64)
	mantissa, exponent, _ := strings.Cut(shortest, "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	power, _ := strconv.Atoi(exponent)
	k := len(digits)
	n := power + 1
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	sign := "+"
	if n-1 < 0 {
		sign = "-"
	}
	magnitude := strconv.Itoa(abs(n - 1))
	if k == 1 {
		return digits + "e" + sign + magnitude
	}
	return digits[:1] + "." + digits[1:] + "e" + sign + magnitude
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// NumberToStringRadix writes a number in the given radix, as Number.prototype.toString does for an integer.
func NumberToStringRadix(f float64, radix int) string {
	if radix == 10 {
		return NumberToString(f)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return NumberToString(f)
	}
	return strconv.FormatInt(int64(f), radix)
}

// Trunc removes the fractional part of a number, as Math.trunc does.
func Trunc(f float64) float64 { return math.Trunc(f) }

// ToInt32 converts a number to a signed 32 bit integer the way the language does when an integer is wanted.
func ToInt32(f float64) int32 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	n := math.Trunc(f)
	n = math.Mod(n, 1<<32)
	if n >= 1<<31 {
		n -= 1 << 32
	} else if n < -(1 << 31) {
		n += 1 << 32
	}
	return int32(n)
}
