// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsval

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/validatorjs/validator/jsstr"
)

// ParseJSON reads JSON text the way JSON.parse does, returning the value it holds or an error whose message is
// the one the source runtime gives.
func ParseJSON(s jsstr.String) (Value, error) {
	p := &jsonParser{text: s}
	p.space()
	value, err := p.value()
	if err != nil {
		return nil, err
	}
	p.space()
	if p.pos != len(p.text) {
		switch c := p.text[p.pos]; {
		case c == '-' || c >= '0' && c <= '9':
			return nil, p.expected("Unexpected number")
		case c == '"':
			return nil, p.expected("Unexpected string")
		}
		return nil, fmt.Errorf("Unexpected non-whitespace character after JSON at position %d (line 1 column %d)",
			p.pos, p.pos+1)
	}
	return value, nil
}

type jsonParser struct {
	text jsstr.String
	pos  int
}

func (p *jsonParser) space() {
	for p.pos < len(p.text) {
		switch p.text[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// at writes a position the way the reader's messages carry it, with the line and the column it falls on.
func (p *jsonParser) at(pos int) string {
	line, column := 1, pos+1
	for i := 0; i < pos && i < len(p.text); i++ {
		if p.text[i] == '\n' {
			line++
			column = pos - i
		}
	}
	return fmt.Sprintf(" in JSON at position %d (line %d column %d)", pos, line, column)
}

// expected is one of the messages the reader gives for a token that is not the one the text needs there.
func (p *jsonParser) expected(what string) error {
	return fmt.Errorf("%s%s", what, p.at(p.pos))
}

// snippet is the window of the text the reader shows around the token it stopped at: the whole text when it is
// short, and ten code units on either side of the token otherwise, with dots where it was cut.
func (p *jsonParser) snippet() string {
	if len(p.text) <= 20 {
		return "\"" + p.text.Go() + "\""
	}
	start, end := p.pos-10, p.pos+10
	if start < 0 {
		start = 0
	}
	if end > len(p.text) {
		end = len(p.text)
	}
	out := "\"" + p.text.Slice(start, end).Go() + "\""
	if start > 0 {
		out = "..." + out
	}
	if end < len(p.text) {
		out += "..."
	}
	return out
}

func (p *jsonParser) unexpected() error {
	if p.pos >= len(p.text) {
		return fmt.Errorf("Unexpected end of JSON input")
	}
	if p.pos == 0 {
		switch p.text.Go() {
		case "NaN", "Infinity", "undefined":
			return fmt.Errorf("%s is not valid JSON", p.snippet())
		}
	}
	return fmt.Errorf("Unexpected token '%s', %s is not valid JSON",
		jsstr.String{p.text[p.pos]}.Go(), p.snippet())
}

func (p *jsonParser) literal(word string, value Value) (Value, error) {
	if p.text.StartsWith(jsstr.Of(word), p.pos) {
		p.pos += len(word)
		return value, nil
	}
	for i := 0; i < len(word); i++ {
		if p.pos+i >= len(p.text) || p.text[p.pos+i] != uint16(word[i]) {
			p.pos += i
			break
		}
	}
	return nil, p.unexpected()
}

func (p *jsonParser) value() (Value, error) {
	if p.pos >= len(p.text) {
		return nil, p.unexpected()
	}
	switch c := p.text[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		return p.string()
	case c == 't':
		return p.literal("true", true)
	case c == 'f':
		return p.literal("false", false)
	case c == 'n':
		return p.literal("null", Null)
	case c == '-' || c >= '0' && c <= '9':
		return p.number()
	}
	return nil, p.unexpected()
}

func (p *jsonParser) object() (Value, error) {
	p.pos++
	out := NewObject()
	p.space()
	if p.pos < len(p.text) && p.text[p.pos] == '}' {
		p.pos++
		return out, nil
	}
	first := true
	for {
		p.space()
		if p.pos >= len(p.text) || p.text[p.pos] != '"' {
			if first {
				return nil, p.expected("Expected property name or '}'")
			}
			return nil, p.expected("Expected double-quoted property name")
		}
		first = false
		key, err := p.string()
		if err != nil {
			return nil, err
		}
		p.space()
		if p.pos >= len(p.text) || p.text[p.pos] != ':' {
			return nil, p.expected("Expected ':' after property name")
		}
		p.pos++
		p.space()
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		out.Set(Str(key).Go(), value)
		p.space()
		if p.pos < len(p.text) && p.text[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.pos < len(p.text) && p.text[p.pos] == '}' {
			p.pos++
			return out, nil
		}
		return nil, p.expected("Expected ',' or '}' after property value")
	}
}

func (p *jsonParser) array() (Value, error) {
	p.pos++
	out := []Value{}
	p.space()
	if p.pos < len(p.text) && p.text[p.pos] == ']' {
		p.pos++
		return out, nil
	}
	for {
		p.space()
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
		p.space()
		if p.pos < len(p.text) && p.text[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.pos < len(p.text) && p.text[p.pos] == ']' {
			p.pos++
			return out, nil
		}
		return nil, p.expected("Expected ',' or ']' after array element")
	}
}

func (p *jsonParser) string() (Value, error) {
	p.pos++
	out := jsstr.String{}
	for {
		if p.pos >= len(p.text) {
			return nil, p.expected("Unterminated string")
		}
		c := p.text[p.pos]
		switch {
		case c == '"':
			p.pos++
			return out, nil
		case c == '\\':
			p.pos++
			if p.pos >= len(p.text) {
				return nil, p.unexpected()
			}
			switch e := p.text[p.pos]; e {
			case '"', '\\', '/':
				out = append(out, e)
			case 'b':
				out = append(out, 0x08)
			case 'f':
				out = append(out, 0x0c)
			case 'n':
				out = append(out, 0x0a)
			case 'r':
				out = append(out, 0x0d)
			case 't':
				out = append(out, 0x09)
			case 'u':
				// The escape needs four hexadecimal digits; the message names the first place where one
				// is missing.
				digits := 0
				for digits < 4 && p.pos+1+digits < len(p.text) && isHexDigit(p.text[p.pos+1+digits]) {
					digits++
				}
				if digits < 4 {
					p.pos += 1 + digits
					return nil, p.expected("Bad Unicode escape")
				}
				value, err := strconv.ParseUint(p.text[p.pos+1:p.pos+5].Go(), 16, 32)
				if err != nil {
					return nil, p.expected("Bad Unicode escape")
				}
				out = append(out, uint16(value))
				p.pos += 4
			default:
				return nil, p.expected("Bad escaped character")
			}
			p.pos++
		case c < 0x20:
			return nil, p.expected("Bad control character in string literal")
		default:
			out = append(out, c)
			p.pos++
		}
	}
}

func (p *jsonParser) number() (Value, error) {
	start := p.pos
	if p.pos < len(p.text) && p.text[p.pos] == '-' {
		p.pos++
	}
	digits := 0
	for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
		p.pos++
		digits++
	}
	if digits == 0 {
		if p.text[start] == '-' {
			return nil, p.expected("No number after minus sign")
		}
		return nil, p.unexpected()
	}
	if digits > 1 && p.text[start] == '0' || digits > 1 && p.text[start] == '-' && p.text[start+1] == '0' {
		p.pos = start + 1
		if p.text[start] == '-' {
			p.pos++
		}
	}
	if p.pos < len(p.text) && p.text[p.pos] == '.' {
		p.pos++
		fracDigits := 0
		for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
			p.pos++
			fracDigits++
		}
		if fracDigits == 0 {
			return nil, p.expected("Unterminated fractional number")
		}
	}
	if p.pos < len(p.text) && (p.text[p.pos] == 'e' || p.text[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.text) && (p.text[p.pos] == '+' || p.text[p.pos] == '-') {
			p.pos++
		}
		expDigits := 0
		for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
			p.pos++
			expDigits++
		}
		if expDigits == 0 {
			return nil, p.expected("Exponent part is missing a number")
		}
	}
	value, err := strconv.ParseFloat(p.text[start:p.pos].Go(), 64)
	if err != nil {
		if num, ok := err.(*strconv.NumError); !ok || num.Err != strconv.ErrRange {
			return nil, p.unexpected()
		}
	}
	return value, nil
}

const uriUnescaped = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"

const uriReserved = ";/?:@&=+$,#"

// EncodeURI percent-encodes the string the way encodeURI does, leaving the unreserved and the reserved
// characters alone. An unpaired surrogate makes it raise the error that function raises.
func EncodeURI(s jsstr.String) string { return encodeWith(s, uriUnescaped+uriReserved) }

// EncodeURIComponent percent-encodes the string the way encodeURIComponent does, leaving only the unreserved
// characters alone.
func EncodeURIComponent(s jsstr.String) string { return encodeWith(s, uriUnescaped) }

func encodeWith(s jsstr.String, keep string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x80 && strings.ContainsRune(keep, rune(c)) {
			b.WriteByte(byte(c))
			continue
		}
		r, width := s.CodePointAt(i)
		if r >= 0xd800 && r <= 0xdfff {
			Throw("URIError", "URI malformed")
		}
		for _, octet := range []byte(string(r)) {
			fmt.Fprintf(&b, "%%%02X", octet)
		}
		i += width - 1
	}
	return b.String()
}

// isHexDigit reports whether the code unit is a hexadecimal digit.
func isHexDigit(c uint16) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
