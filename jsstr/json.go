// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsstr

import (
	"fmt"
	"strings"
)

// Quote writes the string as a JSON string, the way JSON.stringify writes one: the quote, the backslash and the
// control characters are escaped, an unpaired surrogate is escaped so that the result is well formed, and every
// other character is written as it is.
func (s String) Quote() string {
	var b strings.Builder
	b.WriteByte('"')
	run := String{}
	flush := func() {
		if len(run) > 0 {
			b.WriteString(run.Go())
			run = run[:0]
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			flush()
			b.WriteString(`\"`)
		case c == '\\':
			flush()
			b.WriteString(`\\`)
		case c == 0x08:
			flush()
			b.WriteString(`\b`)
		case c == 0x09:
			flush()
			b.WriteString(`\t`)
		case c == 0x0a:
			flush()
			b.WriteString(`\n`)
		case c == 0x0c:
			flush()
			b.WriteString(`\f`)
		case c == 0x0d:
			flush()
			b.WriteString(`\r`)
		case c < 0x20:
			flush()
			fmt.Fprintf(&b, `\u%04x`, c)
		case c >= 0xd800 && c <= 0xdbff:
			if i+1 < len(s) && s[i+1] >= 0xdc00 && s[i+1] <= 0xdfff {
				run = append(run, c, s[i+1])
				i++
				continue
			}
			flush()
			fmt.Fprintf(&b, `\u%04x`, c)
		case c >= 0xdc00 && c <= 0xdfff:
			flush()
			fmt.Fprintf(&b, `\u%04x`, c)
		default:
			run = append(run, c)
		}
	}
	flush()
	b.WriteByte('"')
	return b.String()
}
