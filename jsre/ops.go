// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsre

import "github.com/validatorjs/validator/jsstr"

// Search returns where the pattern first matches, or -1, as the search method does.
func Search(s jsstr.String, re *Regexp) int {
	defer re.resetLastIndex()
	m := re.ExecAt(s, 0)
	if m == nil {
		return -1
	}
	return m.Index
}

// MatchAll returns every match of a global pattern, in order.
func MatchAll(s jsstr.String, re *Regexp) []*Match {
	defer re.resetLastIndex()
	out := []*Match{}
	for pos := 0; pos <= len(s); {
		m := re.ExecAt(s, pos)
		if m == nil {
			break
		}
		out = append(out, m)
		if m.Caps[1] == m.Caps[0] {
			pos = m.Caps[1] + 1
		} else {
			pos = m.Caps[1]
		}
	}
	return out
}

// Matches is the match method: a global pattern gives the text of every match, and any other pattern gives the
// first match, or nil when there is none.
func Matches(s jsstr.String, re *Regexp) ([]jsstr.String, *Match) {
	defer re.resetLastIndex()
	if !re.global {
		m := re.ExecAt(s, 0)
		return nil, m
	}
	all := MatchAll(s, re)
	if len(all) == 0 {
		return nil, nil
	}
	out := make([]jsstr.String, len(all))
	for i, m := range all {
		out[i] = m.Text()
	}
	return out, nil
}

// Replace replaces the first match, or every match when the pattern is global, putting the replacement text in
// place with its $ substitutions.
func Replace(s jsstr.String, re *Regexp, replacement jsstr.String) jsstr.String {
	return ReplaceFunc(s, re, func(m *Match) jsstr.String { return expand(replacement, m) })
}

// ReplaceFunc replaces the first match, or every match when the pattern is global, with what the function returns
// for it.
func ReplaceFunc(s jsstr.String, re *Regexp, by func(*Match) jsstr.String) jsstr.String {
	defer re.resetLastIndex()
	matches := []*Match{}
	if re.global {
		matches = MatchAll(s, re)
	} else if m := re.ExecAt(s, 0); m != nil {
		matches = append(matches, m)
	}
	out := jsstr.String{}
	last := 0
	for _, m := range matches {
		out = append(out, s[last:m.Caps[0]]...)
		out = append(out, by(m)...)
		last = m.Caps[1]
	}
	return append(out, s[last:]...)
}

// expand carries out the $ substitutions of a replacement text.
func expand(replacement jsstr.String, m *Match) jsstr.String {
	out := jsstr.String{}
	for i := 0; i < len(replacement); i++ {
		if replacement[i] != '$' || i+1 >= len(replacement) {
			out = append(out, replacement[i])
			continue
		}
		switch c := replacement[i+1]; {
		case c == '$':
			out = append(out, '$')
			i++
		case c == '&':
			out = append(out, m.Text()...)
			i++
		case c == '`':
			out = append(out, m.Input[:m.Caps[0]]...)
			i++
		case c == '\'':
			out = append(out, m.Input[m.Caps[1]:]...)
			i++
		case c >= '0' && c <= '9':
			index := int(c - '0')
			width := 1
			if i+2 < len(replacement) && replacement[i+2] >= '0' && replacement[i+2] <= '9' {
				two := index*10 + int(replacement[i+2]-'0')
				if two >= 1 && two < m.Count() {
					index, width = two, 2
				}
			}
			if index < 1 || index >= m.Count() {
				out = append(out, replacement[i])
				continue
			}
			if text, ok := m.Group(index); ok {
				out = append(out, text...)
			}
			i += width
		default:
			out = append(out, replacement[i])
		}
	}
	return out
}

// Split splits the text on the pattern, keeping what the groups captured, as the split method does.
func Split(s jsstr.String, re *Regexp) []jsstr.String {
	defer re.resetLastIndex()
	out := []jsstr.String{}
	if len(s) == 0 {
		if re.ExecAt(s, 0) != nil {
			return out
		}
		return append(out, s)
	}
	p, q := 0, 0
	for q < len(s) {
		m := re.ExecAt(s, q)
		if m == nil {
			break
		}
		if m.Index != q {
			q = m.Index
			continue
		}
		end := m.Caps[1]
		if end == p {
			q++
			continue
		}
		out = append(out, s[p:q])
		for i := 1; i < m.Count(); i++ {
			text, ok := m.Group(i)
			if !ok {
				text = nil
			}
			out = append(out, text)
		}
		p = end
		q = p
	}
	return append(out, s[p:])
}
