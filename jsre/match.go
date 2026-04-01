// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsre

import "github.com/validatorjs/validator/jsstr"

// context is the state of one search: the text, where every group has matched so far, and a budget that stops a
// pattern whose backtracking would otherwise not end.
type context struct {
	input  jsstr.String
	caps   []int
	re     *Regexp
	steps  int
	goal   int
	frames [][]qframe
}

// qframe is one running repetition of one quantifier: how many iterations it has made and where the current one
// started, which is how an iteration that matches nothing is recognised.
type qframe struct {
	count int
	start int
}

// cont is what the engine does after a part of the pattern has matched, and also the type of a compiled part.
type cont func(pos int, ctx *context) bool

type matcher = cont

const stepBudget = 40_000_000

// Match is one result of a search: where it starts and what every group captured.
type Match struct {
	// Index is where the whole match starts, counted in code units.
	Index int
	// Input is the text that was searched.
	Input jsstr.String
	// Caps holds the start and the end of the whole match and of every group, -1 where a group did not take part.
	Caps []int
}

// Text is the text the whole match covers.
func (m *Match) Text() jsstr.String { return m.Input[m.Caps[0]:m.Caps[1]] }

// End is where the whole match ends.
func (m *Match) End() int { return m.Caps[1] }

// Count is the number of groups, the whole match included.
func (m *Match) Count() int { return len(m.Caps) / 2 }

// Group returns what group i captured; the second result is false when the group did not take part in the match.
func (m *Match) Group(i int) (jsstr.String, bool) {
	if i < 0 || 2*i+1 >= len(m.Caps) || m.Caps[2*i] < 0 {
		return nil, false
	}
	return m.Input[m.Caps[2*i]:m.Caps[2*i+1]], true
}

func (re *Regexp) compile(n node, k cont) matcher {
	switch t := n.(type) {
	case *seqNode:
		next := k
		for i := len(t.items) - 1; i >= 0; i-- {
			next = re.compile(t.items[i], next)
		}
		return next
	case *altNode:
		options := make([]matcher, len(t.options))
		for i, option := range t.options {
			options[i] = re.compile(option, k)
		}
		return func(pos int, ctx *context) bool {
			saved := append([]int(nil), ctx.caps...)
			for _, option := range options {
				if option(pos, ctx) {
					return true
				}
				copy(ctx.caps, saved)
			}
			return false
		}
	case *charNode:
		want := t.c
		return func(pos int, ctx *context) bool {
			if ctx.spend() || pos >= len(ctx.input) {
				return false
			}
			if !unitEqual(ctx, ctx.input[pos], want) {
				return false
			}
			return k(pos+1, ctx)
		}
	case *pointNode:
		want := t.r
		return func(pos int, ctx *context) bool {
			if ctx.spend() {
				return false
			}
			got, width := ctx.read(pos)
			if width == 0 || !pointEqual(ctx, got, want) {
				return false
			}
			return k(pos+width, ctx)
		}
	case *classNode:
		set := t.set
		return func(pos int, ctx *context) bool {
			if ctx.spend() {
				return false
			}
			got, width := ctx.read(pos)
			if width == 0 || !set.matches(got, ctx.re.ignoreCase, ctx.re.unicode) {
				return false
			}
			return k(pos+width, ctx)
		}
	case *dotNode:
		return func(pos int, ctx *context) bool {
			if ctx.spend() {
				return false
			}
			got, width := ctx.read(pos)
			if width == 0 {
				return false
			}
			if !ctx.re.dotAll && got <= 0xffff && isLineTerminator(uint16(got)) {
				return false
			}
			return k(pos+width, ctx)
		}
	case *assertNode:
		kind := t.kind
		return func(pos int, ctx *context) bool {
			if !ctx.assert(kind, pos) {
				return false
			}
			return k(pos, ctx)
		}
	case *groupNode:
		index := t.index
		inner := re.compile(t.inner, func(pos int, ctx *context) bool {
			end := ctx.caps[2*index+1]
			ctx.caps[2*index+1] = pos
			if k(pos, ctx) {
				return true
			}
			ctx.caps[2*index+1] = end
			return false
		})
		return func(pos int, ctx *context) bool {
			start := ctx.caps[2*index]
			ctx.caps[2*index] = pos
			if inner(pos, ctx) {
				return true
			}
			ctx.caps[2*index] = start
			return false
		}
	case *lookNode:
		return re.compileLook(t, k)
	case *quantNode:
		return re.compileQuant(t, k)
	case *backrefNode:
		index := t.index
		return func(pos int, ctx *context) bool {
			start, end := ctx.caps[2*index], ctx.caps[2*index+1]
			if start < 0 || end < 0 {
				return k(pos, ctx)
			}
			want := ctx.input[start:end]
			if pos+len(want) > len(ctx.input) {
				return false
			}
			for i := range want {
				if !unitEqual(ctx, ctx.input[pos+i], want[i]) {
					return false
				}
			}
			return k(pos+len(want), ctx)
		}
	}
	return k
}

func (re *Regexp) compileLook(t *lookNode, k cont) matcher {
	if t.behind {
		inner := re.compile(t.inner, func(pos int, ctx *context) bool { return pos == ctx.goal })
		return func(pos int, ctx *context) bool {
			saved := append([]int(nil), ctx.caps...)
			goal := ctx.goal
			ctx.goal = pos
			found := false
			for start := pos; start >= 0 && !found; start-- {
				found = inner(start, ctx)
			}
			ctx.goal = goal
			if found == t.negate {
				copy(ctx.caps, saved)
				return false
			}
			if t.negate {
				copy(ctx.caps, saved)
			}
			return k(pos, ctx)
		}
	}
	inner := re.compile(t.inner, func(int, *context) bool { return true })
	return func(pos int, ctx *context) bool {
		saved := append([]int(nil), ctx.caps...)
		found := inner(pos, ctx)
		if found == t.negate {
			copy(ctx.caps, saved)
			return false
		}
		if t.negate {
			copy(ctx.caps, saved)
		}
		if k(pos, ctx) {
			return true
		}
		copy(ctx.caps, saved)
		return false
	}
}

func (re *Regexp) compileQuant(t *quantNode, k cont) matcher {
	min, max, greedy := t.min, t.max, t.greedy
	first, last := t.first, t.last
	id := re.quantifiers
	re.quantifiers++
	var repeat func(count, pos int, ctx *context) bool
	inner := re.compile(t.inner, func(pos int, ctx *context) bool {
		frame := ctx.frames[id][len(ctx.frames[id])-1]
		if frame.count >= min && pos == frame.start {
			return false // an iteration that matched nothing ends the repetition
		}
		return repeat(frame.count+1, pos, ctx)
	})
	repeat = func(count, pos int, ctx *context) bool {
		if ctx.spend() {
			return false
		}
		tryMore := func() bool {
			saved := make([]int, 0, 2*(last-first+1))
			for g := first; g <= last; g++ {
				saved = append(saved, ctx.caps[2*g], ctx.caps[2*g+1])
				ctx.caps[2*g], ctx.caps[2*g+1] = -1, -1
			}
			ctx.frames[id] = append(ctx.frames[id], qframe{count: count, start: pos})
			ok := inner(pos, ctx)
			ctx.frames[id] = ctx.frames[id][:len(ctx.frames[id])-1]
			if !ok {
				for g := first; g <= last; g++ {
					ctx.caps[2*g], ctx.caps[2*g+1] = saved[2*(g-first)], saved[2*(g-first)+1]
				}
			}
			return ok
		}
		if count < max {
			if greedy {
				if tryMore() {
					return true
				}
				if count >= min {
					return k(pos, ctx)
				}
				return false
			}
			if count >= min && k(pos, ctx) {
				return true
			}
			return tryMore()
		}
		if count >= min {
			return k(pos, ctx)
		}
		return false
	}
	return func(pos int, ctx *context) bool { return repeat(0, pos, ctx) }
}

func (ctx *context) spend() bool {
	ctx.steps++
	return ctx.steps > stepBudget
}

// read returns the character at pos and how many code units it takes: one code unit, or one code point when the
// pattern carries the unicode flag.
func (ctx *context) read(pos int) (rune, int) {
	if pos < 0 || pos >= len(ctx.input) {
		return 0, 0
	}
	if ctx.re.unicode {
		return ctx.input.CodePointAt(pos)
	}
	return rune(ctx.input[pos]), 1
}

func (ctx *context) assert(kind, pos int) bool {
	switch kind {
	case assertStart:
		return pos == 0 || ctx.re.multiline && isLineTerminator(ctx.input[pos-1])
	case assertEnd:
		return pos == len(ctx.input) || ctx.re.multiline && isLineTerminator(ctx.input[pos])
	case assertWord:
		return ctx.wordAt(pos-1) != ctx.wordAt(pos)
	default:
		return ctx.wordAt(pos-1) == ctx.wordAt(pos)
	}
}

func (ctx *context) wordAt(pos int) bool {
	if pos < 0 || pos >= len(ctx.input) {
		return false
	}
	c := ctx.input[pos]
	if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	return ctx.re.unicode && ctx.re.ignoreCase && (c == 0x017f || c == 0x212a)
}

func isLineTerminator(c uint16) bool { return jsstr.IsLineTerminator(c) }

func unitEqual(ctx *context, got, want uint16) bool {
	if got == want {
		return true
	}
	if !ctx.re.ignoreCase {
		return false
	}
	if ctx.re.unicode {
		return pointEqual(ctx, rune(got), rune(want))
	}
	return jsstr.Canonicalize(got) == jsstr.Canonicalize(want)
}

func pointEqual(ctx *context, got, want rune) bool {
	if got == want {
		return true
	}
	if !ctx.re.ignoreCase {
		return false
	}
	if !ctx.re.unicode {
		if got > 0xffff || want > 0xffff {
			return false
		}
		return jsstr.Canonicalize(uint16(got)) == jsstr.Canonicalize(uint16(want))
	}
	for _, f := range jsstr.FoldClass(got) {
		if f == want {
			return true
		}
	}
	return false
}

// ExecAt searches from start and returns the first match at or after it, or nil. A sticky pattern only matches at
// start itself.
func (re *Regexp) ExecAt(input jsstr.String, start int) *Match {
	if start < 0 {
		start = 0
	}
	for pos := start; pos <= len(input); pos++ {
		ctx := &context{input: input, re: re, caps: make([]int, 2*(re.groups+1)),
			frames: make([][]qframe, re.quantifiers)}
		for i := range ctx.caps {
			ctx.caps[i] = -1
		}
		ctx.caps[0] = pos
		if re.matcher(pos, ctx) {
			return &Match{Index: pos, Input: input, Caps: ctx.caps}
		}
		if re.sticky {
			return nil
		}
		if re.unicode && pos < len(input) {
			if _, width := input.CodePointAt(pos); width == 2 {
				pos++
			}
		}
	}
	return nil
}

// Exec searches the text the way the exec method does, starting at lastIndex for a global or sticky pattern and
// moving lastIndex on.
func (re *Regexp) Exec(input jsstr.String) *Match {
	start := 0
	if re.global || re.sticky {
		start = re.LastIndex
		if start > len(input) {
			re.LastIndex = 0
			return nil
		}
	}
	m := re.ExecAt(input, start)
	if m == nil {
		if re.global || re.sticky {
			re.LastIndex = 0
		}
		return nil
	}
	if re.global || re.sticky {
		re.LastIndex = m.Caps[1]
	}
	return m
}

// Test reports whether the pattern matches anywhere in the text, moving lastIndex on for a global or sticky
// pattern as the test method does.
func (re *Regexp) Test(input jsstr.String) bool { return re.Exec(input) != nil }

// TestString reports whether the pattern matches the UTF-8 text.
func (re *Regexp) TestString(input string) bool { return re.Test(jsstr.Of(input)) }
