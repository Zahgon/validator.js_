// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsre

import (
	"testing"

	"github.com/validatorjs/validator/jsstr"
)

func mustCompile(t *testing.T, source, flags string) *Regexp {
	t.Helper()
	re, err := Compile(source, flags)
	if err != nil {
		t.Fatalf("Compile(%q, %q): %v", source, flags, err)
	}
	return re
}

func TestTest(t *testing.T) {
	cases := []struct {
		source, flags, input string
		want                 bool
	}{
		{"^a+$", "", "aaa", true},
		{"^a+$", "", "aab", false},
		{"a", "i", "A", true},
		{"ß", "i", "SS", false},
		{"^\\u00df$", "i", "ẞ", false},
		{"^k$", "i", "K", false},
		{"^K$", "iu", "k", true},
		{"^.$", "", "\U0001f600", false},
		{"^.$", "u", "\U0001f600", true},
		{"^.$", "s", "\n", true},
		{"^.$", "", "\n", false},
		{"^a$", "m", "b\na", true},
		{"(?<=a)b", "", "ab", true},
		{"(?<!a)b", "", "ab", false},
		{"(?=ab)a", "", "ab", true},
		{"(?!ab)a", "", "ab", false},
		{"(a)\\1", "", "aa", true},
		{"(a)\\1", "", "ab", false},
		{"(?<n>a)\\k<n>", "", "aa", true},
		{"^\\w+$", "", "a_1", true},
		{"^\\w$", "iu", "ſ", true},
		{"^\\w$", "i", "ſ", false},
		{"^\\d$", "", "٣", false},
		{"^\\s$", "", " ", true},
		{"^\\s$", "", "​", false},
		{"^\\b$", "", "", false},
		{"a\\b", "", "a b", true},
		{"^[]$", "", "", false},
		{"^[^]$", "", "a", true},
		{"^(a|ab)c$", "", "abc", true},
		{"^a{2,}$", "", "aaa", true},
		{"^a{2,3}$", "", "aaaa", false},
		{"^(a+?)(a+)$", "", "aaa", true},
	}
	for _, c := range cases {
		re := mustCompile(t, c.source, c.flags)
		if got := re.Test(jsstr.Of(c.input)); got != c.want {
			t.Errorf("/%s/%s.test(%q) = %v, want %v", c.source, c.flags, c.input, got, c.want)
		}
	}
}

func TestPropertyEscapes(t *testing.T) {
	cases := []struct {
		source, input string
		want          bool
	}{
		{"^\\p{L}$", "a", true},
		{"^\\p{L}$", "1", false},
		{"^\\p{Lu}$", "A", true},
		{"^\\p{General_Category=Lu}$", "A", true},
		{"^\\p{gc=Lu}$", "A", true},
		{"^\\p{Uppercase_Letter}$", "A", true},
		{"^\\P{L}$", "1", true},
		{"^\\p{Script=Greek}$", "α", true},
		{"^\\p{sc=Grek}$", "α", true},
		{"^\\p{Script=Greek}$", "a", false},
		// A combining mark the Greek script borrows: it belongs to the inherited script, and to Greek only
		// as an extension.
		{"^\\p{Script=Greek}$", "͂", false},
		{"^\\p{Script=Inherited}$", "͂", true},
		{"^\\p{Script_Extensions=Greek}$", "͂", true},
		{"^\\p{Script=Common}$", "͵", false},
		{"^\\p{Script_Extensions=Greek}$", "͵", true},
		{"^\\p{scx=Grek}$", "͵", true},
		// A script of the most recent version of the standard, which the Go standard library has no table for.
		{"^\\p{Script=Garay}$", "\U00010d50", true},
		{"^\\p{Script=Todhri}$", "\U000105c0", true},
		{"^\\p{Alphabetic}$", "ʰ", true},
		{"^\\p{Cased}$", "ª", true},
		{"^\\p{Changes_When_Uppercased}$", "ß", true},
		{"^\\p{Changes_When_NFKC_Casefolded}$", "µ", true},
		{"^\\p{Emoji}$", "\U0001f600", true},
		{"^\\p{Emoji_Presentation}$", "©", false},
		{"^\\p{Extended_Pictographic}$", "©", true},
		{"^\\p{Grapheme_Extend}$", "́", true},
		{"^\\p{ID_Start}$", "_", false},
		{"^\\p{ID_Continue}$", "_", true},
		{"^\\p{Default_Ignorable_Code_Point}$", "​", true},
		{"^\\p{ASCII}$", "a", true},
		{"^\\p{Any}$", "\U0010ffff", true},
		{"^\\p{Assigned}$", "﷐", false},
		{"^[\\p{L}\\p{N}]$", "٣", true},
		{"^[^\\p{L}]$", "1", true},
	}
	for _, c := range cases {
		re := mustCompile(t, c.source, "u")
		if got := re.Test(jsstr.Of(c.input)); got != c.want {
			t.Errorf("/%s/u.test(%q) = %v, want %v", c.source, c.input, got, c.want)
		}
	}
}

func TestSyntaxErrors(t *testing.T) {
	cases := []struct{ source, flags, want string }{
		{"(", "", "Invalid regular expression: /(/: Unterminated group"},
		{")", "", "Invalid regular expression: /)/: Unmatched ')'"},
		{"[", "", "Invalid regular expression: /[/: Unterminated character class"},
		{"*", "", "Invalid regular expression: /*/: Nothing to repeat"},
		{"a{2,1}", "", "Invalid regular expression: /a{2,1}/: numbers out of order in {} quantifier"},
		{"[z-a]", "", "Invalid regular expression: /[z-a]/: Range out of order in character class"},
		{"\\", "", "Invalid regular expression: /\\/: \\ at end of pattern"},
		{"a{x}", "u", "Invalid regular expression: /a{x}/u: Incomplete quantifier"},
		{"{x}", "u", "Invalid regular expression: /{x}/u: Lone quantifier brackets"},
		{"(?<=a){2}", "", "Invalid regular expression: /(?<=a){2}/: Invalid quantifier"},
		{"^{2}", "", "Invalid regular expression: /^{2}/: Nothing to repeat"},
		{"\\p{Nope}", "u", "Invalid regular expression: /\\p{Nope}/u: Invalid property name"},
		{"[\\p{Nope}]", "u", "Invalid regular expression: /[\\p{Nope}]/u: Invalid property name in character class"},
		{"\\p{Latin}", "u", "Invalid regular expression: /\\p{Latin}/u: Invalid property name"},
		{"\\p{Script=Nope}", "u", "Invalid regular expression: /\\p{Script=Nope}/u: Invalid property name"},
		{"(?<n>a)(?<n>b)", "", "Invalid regular expression: /(?<n>a)(?<n>b)/: Duplicate capture group name"},
	}
	for _, c := range cases {
		_, err := Compile(c.source, c.flags)
		if err == nil {
			t.Errorf("Compile(%q, %q): no error, want %q", c.source, c.flags, c.want)
			continue
		}
		if err.Error() != c.want {
			t.Errorf("Compile(%q, %q) = %q, want %q", c.source, c.flags, err.Error(), c.want)
		}
	}
}

func TestInvalidFlags(t *testing.T) {
	for _, flags := range []string{"x", "gg", "uu", "a", "U", "v"} {
		if _, err := Compile("a", flags); err == nil {
			t.Errorf("Compile(\"a\", %q): no error, want invalid flags", flags)
		}
	}
}

func TestFlagsOrder(t *testing.T) {
	re := mustCompile(t, "a", "sgi")
	if re.Flags != "gis" {
		t.Errorf("Flags = %q, want %q", re.Flags, "gis")
	}
}

func TestExecCaptures(t *testing.T) {
	re := mustCompile(t, "(a)(b)?", "")
	m := re.Exec(jsstr.Of("a"))
	if m == nil {
		t.Fatal("Exec: no match")
	}
	if got := m.Count(); got != 3 {
		t.Fatalf("Count() = %d, want 3", got)
	}
	if text, ok := m.Group(1); !ok || text.Go() != "a" {
		t.Errorf("Group(1) = %q, %v, want \"a\", true", text.Go(), ok)
	}
	if _, ok := m.Group(2); ok {
		t.Error("Group(2) took part, want the undefined hole")
	}
	if m.Index != 0 {
		t.Errorf("Index = %d, want 0", m.Index)
	}
}

func TestReplaceDollarOne(t *testing.T) {
	re := mustCompile(t, "(a)(b)", "g")
	got := Replace(jsstr.Of("abab"), re, jsstr.Of("$2$1")).Go()
	if got != "baba" {
		t.Errorf("Replace = %q, want %q", got, "baba")
	}
}

func TestSearchAndMatchAll(t *testing.T) {
	re := mustCompile(t, "b", "g")
	if got := Search(jsstr.Of("abc"), re); got != 1 {
		t.Errorf("Search = %d, want 1", got)
	}
	all := MatchAll(jsstr.Of("abcb"), mustCompile(t, "b", "g"))
	if len(all) != 2 {
		t.Errorf("MatchAll = %d matches, want 2", len(all))
	}
}

func TestDecodeSetRoundTrip(t *testing.T) {
	// Every stored set must decode to ranges that rise, do not touch and stay inside the code points.
	for i := range ucdSets {
		ranges := decodeSet(i)
		prev := rune(-2)
		for _, r := range ranges {
			if r.lo <= prev+1 || r.hi < r.lo || r.hi > 0x10ffff {
				t.Fatalf("set %d: range %v out of order after %d", i, r, prev)
			}
			prev = r.hi
		}
	}
}

func TestUcdNamesCover(t *testing.T) {
	for _, name := range []string{"L", "Lu", "ASCII", "Any", "Alphabetic", "Emoji", "Script=Latin",
		"sc=Latn", "Script_Extensions=Latin", "scx=Latn", "General_Category=Nd", "gc=Nd", "White_Space"} {
		if _, ok := ucdProperty(name); !ok {
			t.Errorf("ucdProperty(%q): not known", name)
		}
	}
	for _, name := range []string{"Nope", "Latin", "Latn", "script=Latin", "Script=Nope", "RGI_Emoji"} {
		if _, ok := ucdProperty(name); ok {
			t.Errorf("ucdProperty(%q): known, want unknown", name)
		}
	}
}
