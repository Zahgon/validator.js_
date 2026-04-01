// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsre

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/validatorjs/validator/jsstr"
)

// The kinds of zero-width assertion.
const (
	assertStart = iota
	assertEnd
	assertWord
	assertNotWord
)

type node interface{ isNode() }

type seqNode struct{ items []node }

type altNode struct{ options []node }

// charNode is one code unit, how a pattern without the unicode flag reads a character.
type charNode struct{ c uint16 }

// pointNode is one code point, how a pattern with the unicode flag reads a character.
type pointNode struct{ r rune }

type classNode struct{ set *charSet }

type dotNode struct{}

type assertNode struct{ kind int }

type groupNode struct {
	index int
	inner node
}

type lookNode struct {
	inner          node
	negate, behind bool
}

type quantNode struct {
	inner       node
	min, max    int
	greedy      bool
	first, last int
}

type backrefNode struct{ index int }

func (*seqNode) isNode()     {}
func (*altNode) isNode()     {}
func (*charNode) isNode()    {}
func (*pointNode) isNode()   {}
func (*classNode) isNode()   {}
func (*dotNode) isNode()     {}
func (*assertNode) isNode()  {}
func (*groupNode) isNode()   {}
func (*lookNode) isNode()    {}
func (*quantNode) isNode()   {}
func (*backrefNode) isNode() {}

// crange is a range of code points, both ends included.
type crange struct{ lo, hi rune }

// classItem is one member of a character class: a set of ranges, or everything outside them.
type classItem struct {
	ranges []crange
	negate bool
}

// charSet is a character class: the union of its items, taken the other way round when it is negated.
type charSet struct {
	items  []classItem
	negate bool
}

func (s *charSet) add(lo, hi rune) { s.items = append(s.items, classItem{ranges: []crange{{lo, hi}}}) }

func (s *charSet) addItem(item classItem) { s.items = append(s.items, item) }

func (s *charSet) contains(r rune) bool {
	for _, item := range s.items {
		in := false
		for _, rg := range item.ranges {
			if r >= rg.lo && r <= rg.hi {
				in = true
				break
			}
		}
		if in != item.negate {
			return true
		}
	}
	return false
}

// matches applies the case folding of the pattern: a character matches when any character it is case equivalent
// to is in the set, and a negated class turns that answer round.
func (s *charSet) matches(r rune, fold, uni bool) bool {
	in := s.contains(r)
	if !in && fold {
		if uni {
			for _, f := range jsstr.FoldClass(r) {
				if s.contains(f) {
					in = true
					break
				}
			}
		} else if r <= 0xffff {
			for _, c := range jsstr.CaseClass(uint16(r)) {
				if s.contains(rune(c)) {
					in = true
					break
				}
			}
		}
	}
	return in != s.negate
}

var (
	digitRanges = []crange{{'0', '9'}}
	wordRanges  = []crange{{'0', '9'}, {'A', 'Z'}, {'_', '_'}, {'a', 'z'}}
	// The word characters of a unicode pattern that ignores case also fold onto an ASCII letter.
	wordRangesFold = append(append([]crange{}, wordRanges...), crange{0x017f, 0x017f}, crange{0x212a, 0x212a})
	spaceRanges    = buildSpaceRanges()
)

func buildSpaceRanges() []crange {
	out := []crange{}
	for _, c := range jsstr.Whitespace {
		out = append(out, crange{rune(c), rune(c)})
	}
	for _, c := range jsstr.LineTerminators {
		out = append(out, crange{rune(c), rune(c)})
	}
	return out
}

func (p *parser) wordItem(negate bool) classItem {
	if p.re.unicode && p.re.ignoreCase {
		return classItem{ranges: wordRangesFold, negate: negate}
	}
	return classItem{ranges: wordRanges, negate: negate}
}

// parseEscape reads a backslash escape outside a character class.
func (p *parser) parseEscape() (node, bool, error) {
	p.pos++ // the backslash
	if p.eof() {
		return nil, false, fmt.Errorf(`\ at end of pattern`)
	}
	switch c := p.peek(); c {
	case 'b':
		p.pos++
		return &assertNode{kind: assertWord}, false, nil
	case 'B':
		p.pos++
		return &assertNode{kind: assertNotWord}, false, nil
	case 'd', 'D', 's', 'S', 'w', 'W', 'p', 'P':
		item, ok, err := p.classEscapeItem()
		if err != nil {
			return nil, false, err
		}
		if ok {
			return &classNode{set: &charSet{items: []classItem{item}}}, true, nil
		}
	case 'k':
		if len(p.re.names) > 0 || p.re.unicode {
			p.pos++
			if !p.accept('<') {
				return nil, false, fmt.Errorf("Invalid named reference")
			}
			start := p.pos
			for !p.eof() && p.peek() != '>' {
				p.pos++
			}
			if p.eof() {
				return nil, false, fmt.Errorf("Invalid named reference")
			}
			name := string(p.src[start:p.pos])
			p.pos++
			index, ok := p.re.names[name]
			if !ok {
				return nil, false, fmt.Errorf("Invalid named capture referenced")
			}
			return &backrefNode{index: index}, true, nil
		}
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		save := p.pos
		value, _ := p.parseDecimal()
		if value <= p.re.groups {
			return &backrefNode{index: value}, true, nil
		}
		p.pos = save
		if p.re.unicode {
			return nil, false, fmt.Errorf("Invalid escape")
		}
		if p.peek() == '8' || p.peek() == '9' {
			return p.literal(p.next()), true, nil
		}
		return p.literal(p.legacyOctal()), true, nil
	}
	r, err := p.characterEscape()
	if err != nil {
		return nil, false, err
	}
	return p.literal(r), true, nil
}

// legacyOctal reads the octal escape a pattern without the unicode flag allows.
func (p *parser) legacyOctal() rune {
	value := rune(0)
	for n := 0; n < 3 && !p.eof() && p.src[p.pos] >= '0' && p.src[p.pos] <= '7'; n++ {
		if value*8+(p.src[p.pos]-'0') > 0o377 {
			break
		}
		value = value*8 + (p.src[p.pos] - '0')
		p.pos++
	}
	return value
}

// characterEscape reads the escapes that stand for one character.
func (p *parser) characterEscape() (rune, error) {
	c := p.next()
	switch c {
	case 'f':
		return 0x0c, nil
	case 'n':
		return 0x0a, nil
	case 'r':
		return 0x0d, nil
	case 't':
		return 0x09, nil
	case 'v':
		return 0x0b, nil
	case '0':
		if !p.eof() && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
			if p.re.unicode {
				return 0, fmt.Errorf("Invalid decimal escape")
			}
			p.pos--
			return p.legacyOctal(), nil
		}
		return 0, nil
	case '1', '2', '3', '4', '5', '6', '7':
		if p.re.unicode {
			return 0, fmt.Errorf("Invalid escape")
		}
		p.pos--
		return p.legacyOctal(), nil
	case '8', '9':
		if p.re.unicode {
			return 0, fmt.Errorf("Invalid escape")
		}
		return c, nil
	case 'c':
		if !p.eof() && isASCIILetter(p.peek()) {
			return p.next() % 32, nil
		}
		if p.re.unicode {
			return 0, fmt.Errorf("Invalid unicode escape")
		}
		p.pos--
		return '\\', nil
	case 'x':
		if value, ok := p.hex(2); ok {
			return value, nil
		}
		if p.re.unicode {
			return 0, fmt.Errorf("Invalid escape")
		}
		return 'x', nil
	case 'u':
		if value, ok := p.unicodeEscape(); ok {
			return value, nil
		}
		if p.re.unicode {
			return 0, fmt.Errorf("Invalid Unicode escape")
		}
		return 'u', nil
	}
	if p.re.unicode && (isASCIILetter(c) || c > 0x7f && c != 0x2028 && c != 0x2029 && !isSyntaxCharacter(c) && c != '/') {
		return 0, fmt.Errorf("Invalid escape")
	}
	return c, nil
}

func (p *parser) unicodeEscape() (rune, bool) {
	save := p.pos
	if p.re.unicode && p.accept('{') {
		start := p.pos
		for !p.eof() && p.peek() != '}' {
			p.pos++
		}
		if p.eof() {
			p.pos = save
			return 0, false
		}
		value, err := strconv.ParseInt(string(p.src[start:p.pos]), 16, 32)
		p.pos++
		if err != nil || value > 0x10ffff {
			p.pos = save
			return 0, false
		}
		return rune(value), true
	}
	high, ok := p.hex(4)
	if !ok {
		return 0, false
	}
	// With the unicode flag a surrogate pair written as two escapes is one code point.
	if p.re.unicode && high >= 0xd800 && high <= 0xdbff && p.pos+1 < len(p.src) && p.src[p.pos] == '\\' &&
		p.src[p.pos+1] == 'u' {
		mark := p.pos
		p.pos += 2
		if low, ok := p.hex(4); ok && low >= 0xdc00 && low <= 0xdfff {
			return (high-0xd800)*0x400 + (low - 0xdc00) + 0x10000, true
		}
		p.pos = mark
	}
	return high, true
}

func (p *parser) hex(n int) (rune, bool) {
	if p.pos+n > len(p.src) {
		return 0, false
	}
	value := rune(0)
	for i := 0; i < n; i++ {
		d := hexValue(p.src[p.pos+i])
		if d < 0 {
			return 0, false
		}
		value = value*16 + rune(d)
	}
	p.pos += n
	return value, true
}

func hexValue(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	}
	return -1
}

func isASCIILetter(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }

func isSyntaxCharacter(r rune) bool {
	switch r {
	case '^', '$', '\\', '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|':
		return true
	}
	return false
}

// classEscapeItem reads the escapes that stand for a set of characters; the second result is false when the
// escape at hand is not one of them.
func (p *parser) classEscapeItem() (classItem, bool, error) {
	switch p.peek() {
	case 'd':
		p.pos++
		return classItem{ranges: digitRanges}, true, nil
	case 'D':
		p.pos++
		return classItem{ranges: digitRanges, negate: true}, true, nil
	case 's':
		p.pos++
		return classItem{ranges: spaceRanges}, true, nil
	case 'S':
		p.pos++
		return classItem{ranges: spaceRanges, negate: true}, true, nil
	case 'w':
		p.pos++
		return p.wordItem(false), true, nil
	case 'W':
		p.pos++
		return p.wordItem(true), true, nil
	case 'p', 'P':
		if !p.re.unicode {
			return classItem{}, false, nil
		}
		negate := p.peek() == 'P'
		p.pos++
		if !p.accept('{') {
			return classItem{}, false, p.inClassError(fmt.Errorf("Invalid property name"))
		}
		start := p.pos
		for !p.eof() && p.peek() != '}' {
			p.pos++
		}
		if p.eof() {
			return classItem{}, false, p.inClassError(fmt.Errorf("Invalid property name"))
		}
		name := string(p.src[start:p.pos])
		p.pos++
		ranges, err := propertyRanges(name)
		if err != nil {
			return classItem{}, false, p.inClassError(err)
		}
		return classItem{ranges: ranges, negate: negate}, true, nil
	}
	return classItem{}, false, nil
}

// inClassError names the character class in a message raised while one is being read.
func (p *parser) inClassError(err error) error {
	if p.classDepth > 0 {
		return fmt.Errorf("%s in character class", err)
	}
	return err
}

func propertyRanges(name string) ([]crange, error) {
	if ranges, ok := ucdProperty(name); ok {
		return ranges, nil
	}
	return nil, fmt.Errorf("Invalid property name")
}

func trimPrefix(s, prefix string) (string, bool) {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}

func tableRanges(table *unicode.RangeTable) []crange {
	out := make([]crange, 0, len(table.R16)+len(table.R32))
	for _, r := range table.R16 {
		for c := rune(r.Lo); c <= rune(r.Hi); c += rune(r.Stride) {
			out = append(out, crange{c, c})
			if r.Stride == 1 {
				out[len(out)-1] = crange{rune(r.Lo), rune(r.Hi)}
				break
			}
		}
	}
	for _, r := range table.R32 {
		for c := rune(r.Lo); c <= rune(r.Hi); c += rune(r.Stride) {
			out = append(out, crange{c, c})
			if r.Stride == 1 {
				out[len(out)-1] = crange{rune(r.Lo), rune(r.Hi)}
				break
			}
		}
	}
	return out
}

// parseClass reads a bracketed character class.
func (p *parser) parseClass() (node, error) {
	p.pos++ // '['
	p.classDepth++
	defer func() { p.classDepth-- }()
	set := &charSet{}
	if p.accept('^') {
		set.negate = true
	}
	for {
		if p.eof() {
			return nil, fmt.Errorf("Unterminated character class")
		}
		if p.accept(']') {
			return &classNode{set: set}, nil
		}
		lo, loItem, err := p.classAtom()
		if err != nil {
			return nil, err
		}
		if p.peek() == '-' && p.pos+1 < len(p.src) && p.src[p.pos+1] != ']' {
			mark := p.pos
			p.pos++
			hi, hiItem, err := p.classAtom()
			if err != nil {
				return nil, err
			}
			if loItem != nil || hiItem != nil {
				if p.re.unicode {
					return nil, fmt.Errorf("Invalid character class")
				}
				// Annex B keeps the dash as an ordinary character next to a class escape.
				p.pos = mark
				p.addAtom(set, lo, loItem)
				continue
			}
			if lo > hi {
				return nil, fmt.Errorf("Range out of order in character class")
			}
			set.add(lo, hi)
			continue
		}
		p.addAtom(set, lo, loItem)
	}
}

func (p *parser) addAtom(set *charSet, r rune, item *classItem) {
	if item != nil {
		set.addItem(*item)
		return
	}
	set.add(r, r)
}

// classAtom reads one member of a character class: either a single character or a whole set.
func (p *parser) classAtom() (rune, *classItem, error) {
	if p.peek() == '\\' {
		p.pos++
		if p.eof() {
			return 0, nil, fmt.Errorf(`\ at end of pattern`)
		}
		if p.peek() == 'b' {
			p.pos++
			return 0x08, nil, nil
		}
		if p.peek() == '-' {
			p.pos++
			return '-', nil, nil
		}
		if item, ok, err := p.classEscapeItem(); err != nil {
			return 0, nil, err
		} else if ok {
			return 0, &item, nil
		}
		r, err := p.characterEscape()
		return r, nil, err
	}
	r := p.next()
	if !p.re.unicode && r > 0xffff {
		// Without the unicode flag the pattern itself is read as code units.
		high, low := surrogates(r)
		p.src = append(p.src[:p.pos-1], append([]rune{high, low}, p.src[p.pos:]...)...)
		r = high
	}
	return r, nil, nil
}

func surrogates(r rune) (rune, rune) {
	r -= 0x10000
	return 0xd800 + (r >> 10), 0xdc00 + (r & 0x3ff)
}
