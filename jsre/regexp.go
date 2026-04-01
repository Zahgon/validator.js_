// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

// Package jsre is the regular expression dialect the validators are written in. The standard library's regular
// expressions are a different language: they have no look-around and no back references, they match over UTF-8
// bytes rather than UTF-16 code units, and their case folding and their character classes are not the ones the
// source dialect defines. The patterns of the library are kept here as they are written and matched by this
// engine instead.
package jsre

import (
	"fmt"
	"strconv"
	"strings"
)

// Regexp is a compiled pattern together with the flags it was compiled with.
type Regexp struct {
	// Source is the pattern text, the value of the source property.
	Source string
	// Flags is the flag letters in the order the flags property reports them.
	Flags string
	// LastIndex is where the next sticky or global search starts, the value of the lastIndex property.
	LastIndex int

	global     bool
	ignoreCase bool
	multiline  bool
	dotAll     bool
	unicode    bool
	sticky     bool
	hasIndices bool

	root        node
	groups      int
	quantifiers int
	names       map[string]int
	matcher     matcher
}

// SyntaxError is the error a pattern that is not a regular expression produces, with the message text of the
// source runtime.
type SyntaxError struct {
	// Message is the whole message, including the pattern and the flags.
	Message string
}

// Error returns the syntax error's message, as the runtime reports it.
func (e *SyntaxError) Error() string { return e.Message }

// Name is the error's name property, which the command prints before the message.
func (e *SyntaxError) Name() string { return "SyntaxError" }

const flagOrder = "dgimsuvy"

// Compile builds a pattern, reporting a SyntaxError when the pattern or the flags are not valid.
func Compile(source, flags string) (*Regexp, error) {
	re := &Regexp{Source: source, names: map[string]int{}}
	seen := map[rune]bool{}
	for _, f := range flags {
		if seen[f] || !strings.ContainsRune(flagOrder, f) || f == 'v' {
			return nil, &SyntaxError{Message: fmt.Sprintf("Invalid flags supplied to RegExp constructor '%s'", flags)}
		}
		seen[f] = true
		switch f {
		case 'd':
			re.hasIndices = true
		case 'g':
			re.global = true
		case 'i':
			re.ignoreCase = true
		case 'm':
			re.multiline = true
		case 's':
			re.dotAll = true
		case 'u':
			re.unicode = true
		case 'y':
			re.sticky = true
		}
	}
	for _, f := range flagOrder {
		if seen[f] {
			re.Flags += string(f)
		}
	}
	p := &parser{src: []rune(source), re: re}
	if err := p.countGroups(); err != nil {
		return nil, p.fail(err.Error())
	}
	root, err := p.parseDisjunction()
	if err != nil {
		return nil, p.fail(err.Error())
	}
	if p.pos < len(p.src) {
		if p.src[p.pos] == ')' {
			return nil, p.fail("Unmatched ')'")
		}
		return nil, p.fail("Invalid regular expression")
	}
	re.root = root
	re.matcher = re.compile(root, func(pos int, ctx *context) bool {
		ctx.caps[1] = pos
		return true
	})
	return re, nil
}

// MustCompile builds a pattern that is known to be valid, and panics when it is not.
func MustCompile(source, flags string) *Regexp {
	re, err := Compile(source, flags)
	if err != nil {
		panic(err)
	}
	return re
}

// resetLastIndex puts the search position back at the start, which the string methods that take a pattern do:
// replace, match, matchAll, search and split all start at the beginning and leave lastIndex at zero.
func (re *Regexp) resetLastIndex() {
	if re.global || re.sticky {
		re.LastIndex = 0
	}
}

// Global reports whether the pattern carries the g flag.
func (re *Regexp) Global() bool { return re.global }

// Sticky reports whether the pattern carries the y flag.
func (re *Regexp) Sticky() bool { return re.sticky }

// GroupCount is the number of capturing groups in the pattern.
func (re *Regexp) GroupCount() int { return re.groups }

// GroupName returns the index of a named group, and false when there is no group of that name.
func (re *Regexp) GroupName(name string) (int, bool) {
	i, ok := re.names[name]
	return i, ok
}

type parser struct {
	src    []rune
	pos    int
	re     *Regexp
	groups int
	// afterLook records that the atom just read was a look-around a quantifier may not follow, which the
	// dialect reports differently from a quantifier with nothing in front of it.
	afterLook bool
	// classDepth is how many character classes are being read, which a message raised inside one names.
	classDepth int
}

func (p *parser) fail(reason string) *SyntaxError {
	return &SyntaxError{Message: fmt.Sprintf("Invalid regular expression: /%s/%s: %s", p.re.Source, p.re.Flags, reason)}
}

// countGroups counts the capturing groups and records their names before parsing, because a back reference may
// come before the group it names. It reports a name two groups share, which the dialect refuses whatever
// alternative each group is in.
func (p *parser) countGroups() error {
	inClass := false
	for i := 0; i < len(p.src); i++ {
		switch p.src[i] {
		case '\\':
			i++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '(':
			if inClass {
				continue
			}
			if i+1 < len(p.src) && p.src[i+1] == '?' {
				if i+2 < len(p.src) && p.src[i+2] == '<' && i+3 < len(p.src) && p.src[i+3] != '=' && p.src[i+3] != '!' {
					p.groups++
					end := i + 3
					for end < len(p.src) && p.src[end] != '>' {
						end++
					}
					name := string(p.src[i+3 : end])
					if _, taken := p.re.names[name]; taken {
						return fmt.Errorf("Duplicate capture group name")
					}
					p.re.names[name] = p.groups
				}
				continue
			}
			p.groups++
		}
	}
	p.re.groups = p.groups
	p.groups = 0
	return nil
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) peek() rune {
	if p.eof() {
		return -1
	}
	return p.src[p.pos]
}

func (p *parser) next() rune {
	r := p.peek()
	p.pos++
	return r
}

func (p *parser) accept(r rune) bool {
	if p.peek() == r {
		p.pos++
		return true
	}
	return false
}

func (p *parser) parseDisjunction() (node, error) {
	options := []node{}
	for {
		alt, err := p.parseAlternative()
		if err != nil {
			return nil, err
		}
		options = append(options, alt)
		if !p.accept('|') {
			break
		}
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return &altNode{options: options}, nil
}

func (p *parser) parseAlternative() (node, error) {
	items := []node{}
	for !p.eof() && p.peek() != '|' && p.peek() != ')' {
		term, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		if term != nil {
			items = append(items, term)
		}
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &seqNode{items: items}, nil
}

func (p *parser) parseTerm() (node, error) {
	start := p.groups
	p.afterLook = false
	atom, quantifiable, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	afterLook := p.afterLook
	min, max, greedy, ok, err := p.parseQuantifier()
	if err != nil {
		return nil, err
	}
	if !ok {
		// A brace that follows an atom and is not a quantifier is an unfinished quantifier, not the lone
		// bracket that a brace with nothing in front of it is.
		if p.re.unicode && p.peek() == '{' {
			return nil, fmt.Errorf("Incomplete quantifier")
		}
		return atom, nil
	}
	if !quantifiable {
		if afterLook {
			return nil, fmt.Errorf("Invalid quantifier")
		}
		return nil, fmt.Errorf("Nothing to repeat")
	}
	return &quantNode{inner: atom, min: min, max: max, greedy: greedy, first: start + 1, last: p.groups}, nil
}

const unbounded = 1 << 30

func (p *parser) parseQuantifier() (min, max int, greedy, ok bool, err error) {
	switch p.peek() {
	case '*':
		p.pos++
		min, max = 0, unbounded
	case '+':
		p.pos++
		min, max = 1, unbounded
	case '?':
		p.pos++
		min, max = 0, 1
	case '{':
		save := p.pos
		p.pos++
		lo, haveLo := p.parseDecimal()
		if !haveLo {
			p.pos = save
			return 0, 0, false, false, nil
		}
		hi := lo
		if p.accept(',') {
			var haveHi bool
			hi, haveHi = p.parseDecimal()
			if !haveHi {
				hi = unbounded
			}
		}
		if !p.accept('}') {
			p.pos = save
			return 0, 0, false, false, nil
		}
		if hi < lo {
			return 0, 0, false, false, fmt.Errorf("numbers out of order in {} quantifier")
		}
		min, max = lo, hi
	default:
		return 0, 0, false, false, nil
	}
	greedy = !p.accept('?')
	return min, max, greedy, true, nil
}

func (p *parser) parseDecimal() (int, bool) {
	start := p.pos
	for !p.eof() && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		p.pos++
	}
	if p.pos == start {
		return 0, false
	}
	value, err := strconv.Atoi(string(p.src[start:p.pos]))
	if err != nil {
		return unbounded, true
	}
	return value, true
}

// parseAtom reads one atom or assertion; the second result says whether a quantifier may follow it.
func (p *parser) parseAtom() (node, bool, error) {
	switch c := p.peek(); c {
	case '^':
		p.pos++
		return &assertNode{kind: assertStart}, false, nil
	case '$':
		p.pos++
		return &assertNode{kind: assertEnd}, false, nil
	case '.':
		p.pos++
		return &dotNode{}, true, nil
	case '*', '+', '?':
		return nil, false, fmt.Errorf("Nothing to repeat")
	case '{':
		save := p.pos
		if _, _, _, ok, err := p.parseQuantifier(); err != nil || ok {
			if err != nil {
				return nil, false, err
			}
			return nil, false, fmt.Errorf("Nothing to repeat")
		}
		p.pos = save
		if p.re.unicode {
			return nil, false, fmt.Errorf("Lone quantifier brackets")
		}
		p.pos++
		return p.literal('{'), true, nil
	case '}', ']':
		p.pos++
		if p.re.unicode {
			return nil, false, fmt.Errorf("Lone quantifier brackets")
		}
		return p.literal(c), true, nil
	case '[':
		set, err := p.parseClass()
		if err != nil {
			return nil, false, err
		}
		return set, true, nil
	case '(':
		return p.parseGroup()
	case '\\':
		return p.parseEscape()
	default:
		p.pos++
		return p.literal(c), true, nil
	}
}

func (p *parser) literal(r rune) node {
	if p.re.unicode || r > 0xffff {
		return &pointNode{r: r}
	}
	return &charNode{c: uint16(r)}
}

func (p *parser) parseGroup() (node, bool, error) {
	p.pos++ // '('
	if p.accept('?') {
		switch {
		case p.accept(':'):
			inner, err := p.parseDisjunction()
			if err != nil {
				return nil, false, err
			}
			if !p.accept(')') {
				return nil, false, fmt.Errorf("Unterminated group")
			}
			return inner, true, nil
		case p.accept('='):
			return p.lookAround(false, false)
		case p.accept('!'):
			return p.lookAround(true, false)
		case p.peek() == '<':
			p.pos++
			if p.accept('=') {
				return p.lookAround(false, true)
			}
			if p.accept('!') {
				return p.lookAround(true, true)
			}
			start := p.pos
			for !p.eof() && p.peek() != '>' {
				p.pos++
			}
			if p.eof() {
				return nil, false, fmt.Errorf("Invalid capture group name")
			}
			name := string(p.src[start:p.pos])
			p.pos++
			p.groups++
			index := p.groups
			inner, err := p.parseDisjunction()
			if err != nil {
				return nil, false, err
			}
			if !p.accept(')') {
				return nil, false, fmt.Errorf("Unterminated group")
			}
			p.re.names[name] = index
			return &groupNode{index: index, inner: inner}, true, nil
		default:
			return nil, false, fmt.Errorf("Invalid group")
		}
	}
	p.groups++
	index := p.groups
	inner, err := p.parseDisjunction()
	if err != nil {
		return nil, false, err
	}
	if !p.accept(')') {
		return nil, false, fmt.Errorf("Unterminated group")
	}
	return &groupNode{index: index, inner: inner}, true, nil
}

func (p *parser) lookAround(negate, behind bool) (node, bool, error) {
	inner, err := p.parseDisjunction()
	if err != nil {
		return nil, false, err
	}
	if !p.accept(')') {
		return nil, false, fmt.Errorf("Unterminated group")
	}
	// Without the unicode flag a look-ahead may carry a quantifier, which the unicode grammar forbids.
	p.afterLook = behind || p.re.unicode
	return &lookNode{inner: inner, negate: negate, behind: behind}, !behind && !p.re.unicode, nil
}
