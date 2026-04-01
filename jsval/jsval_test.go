// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsval

import (
	"math"
	"testing"

	"github.com/validatorjs/validator/jsstr"
)

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want string // the ISO form, or "" for a date that cannot be read
	}{
		{"2020-01-02", "2020-01-02T00:00:00.000Z"},
		{"2020-01-02T03:04:05Z", "2020-01-02T03:04:05.000Z"},
		{"2020-01-02T03:04:05.678Z", "2020-01-02T03:04:05.678Z"},
		{"2020-01-02T03:04:05,678Z", ""}, // a comma is not the standard's separator here
		{"2020-01-02T03:04:05+05:30", "2020-01-01T21:34:05.000Z"},
		{"2020-01-02T03:04:05-00:00", "2020-01-02T03:04:05.000Z"},
		// Without a zone, a date and time is local, and a date alone is universal.
		{"2020-01-02T03:04:05", "2020-01-02T03:04:05.000Z"},
		{"2020-01", "2020-01-01T00:00:00.000Z"},
		{"2020", "2020-01-01T00:00:00.000Z"},
		{"+020020-01-02", "+020020-01-02T00:00:00.000Z"},
		{"-000001-01-01", "-000001-01-01T00:00:00.000Z"},
		{"2020-01-02T24:00:00Z", "2020-01-03T00:00:00.000Z"},
		{"2020-02-29", "2020-02-29T00:00:00.000Z"},
		{"2021-02-29", "2021-03-01T00:00:00.000Z"}, // a day past the month rolls over
		{"2020-13-01", ""},
		{"2020-00-01", ""},
		{"2020-01-32", ""},
		{"2020-01-02T25:00:00Z", ""},
		{"2020-01-02T23:60:00Z", ""},
		{"+275760-09-13T00:00:00Z", "+275760-09-13T00:00:00.000Z"},
		{"+275760-09-14T00:00:00Z", ""},
		{"275760-09-13T00:00:00Z", ""}, // an expanded year needs its sign
		// The legacy grammar.
		{"Jan 2 2020", "2020-01-02T00:00:00.000Z"},
		{"January 2, 2020", "2020-01-02T00:00:00.000Z"},
		{"Thu, 02 Jan 2020 00:00:00 GMT", "2020-01-02T00:00:00.000Z"},
		{"Mon Jan 02 2020 03:04:05 GMT+0000", "2020-01-02T03:04:05.000Z"},
		{"Mon Jan 02 2020 03:04:05 GMT+000020200102", ""},
		{"Jan 02 2020 03:04:05 +00000", ""},
		{"1/2/2020", "2020-01-02T00:00:00.000Z"},
		{"2020/01/02", "2020-01-02T00:00:00.000Z"},
		{"Jan 02 2020 (a comment)", "2020-01-02T00:00:00.000Z"},
		{"not a date", ""},
		{"", ""},
		{" ", ""},
	}
	for _, c := range cases {
		ms := ParseDate(jsstr.Of(c.in))
		if c.want == "" {
			if !math.IsNaN(ms) {
				t.Errorf("ParseDate(%q) = %v, want NaN", c.in, ms)
			}
			continue
		}
		if math.IsNaN(ms) {
			t.Errorf("ParseDate(%q) = NaN, want %s", c.in, c.want)
			continue
		}
		if got := NewDate(ms).ToISOString(); got != c.want {
			t.Errorf("ParseDate(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestDateFields(t *testing.T) {
	d := NewDate(ParseDate(jsstr.Of("2020-03-07T10:20:30.400Z")))
	if got := d.ToISOString(); got != "2020-03-07T10:20:30.400Z" {
		t.Fatalf("ToISOString = %s", got)
	}
	if y, m, day := d.UTCFullYear(), d.UTCMonth(), d.UTCDate(); y != 2020 || m != 2 || day != 7 {
		t.Errorf("year, month, day = %d, %d, %d, want 2020, 2, 7", y, m, day)
	}
	if got := d.Time; got != 1583576430400 {
		t.Errorf("Time = %v, want 1583576430400", got)
	}
}

func TestToNumberAndToString(t *testing.T) {
	cases := []struct {
		in   Value
		num  float64
		text string
	}{
		{Undefined, math.NaN(), "undefined"},
		{Null, 0, "null"},
		{true, 1, "true"},
		{false, 0, "false"},
		{Num(0), 0, "0"},
		{Num(math.Copysign(0, -1)), math.Copysign(0, -1), "0"},
		{Num(1e21), 1e21, "1e+21"},
		{Of(""), 0, ""},
		{Of("12"), 12, "12"},
		{Of("x"), math.NaN(), "x"},
		{[]Value{}, 0, ""},
		{[]Value{Of("a"), Of("b")}, math.NaN(), "a,b"},
		{NewObject(), math.NaN(), "[object Object]"},
	}
	for _, c := range cases {
		got := ToNumber(c.in)
		if math.IsNaN(c.num) != math.IsNaN(got) || (!math.IsNaN(c.num) && got != c.num) {
			t.Errorf("ToNumber(%v) = %v, want %v", c.in, got, c.num)
		}
		if text := ToString(c.in).Go(); text != c.text {
			t.Errorf("ToString(%v) = %q, want %q", c.in, text, c.text)
		}
	}
}

func TestTypeOfAndTruthy(t *testing.T) {
	cases := []struct {
		in     Value
		kind   string
		truthy bool
	}{
		{Undefined, "undefined", false},
		{Null, "object", false},
		{true, "boolean", true},
		{false, "boolean", false},
		{Num(0), "number", false},
		{Num(math.NaN()), "number", false},
		{Num(1), "number", true},
		{Of(""), "string", false},
		{Of("0"), "string", true},
		{[]Value{}, "object", true},
		{NewObject(), "object", true},
	}
	for _, c := range cases {
		if got := TypeOf(c.in); got != c.kind {
			t.Errorf("TypeOf(%v) = %q, want %q", c.in, got, c.kind)
		}
		if got := Truthy(c.in); got != c.truthy {
			t.Errorf("Truthy(%v) = %v, want %v", c.in, got, c.truthy)
		}
	}
}

func TestMergeKeepsDefaultsAndOrder(t *testing.T) {
	defaults := ObjectOf(Of("a"), Num(1), Of("b"), Num(2))
	// An option that is not an object is replaced by an empty one, so every default applies.
	for _, options := range []Value{Undefined, Null, Num(0), Of("x"), true} {
		out := Merge(options, defaults)
		if ToNumber(out.Get("a")) != 1 || ToNumber(out.Get("b")) != 2 {
			t.Errorf("Merge(%v): defaults did not apply", options)
		}
	}
	// An option that is set stays, including one that is false; one that is undefined takes the default.
	out := Merge(ObjectOf(Of("a"), false, Of("b"), Undefined), defaults)
	if Truthy(out.Get("a")) {
		t.Error("Merge: a false option was replaced by its default")
	}
	if ToNumber(out.Get("b")) != 2 {
		t.Error("Merge: an undefined option did not take its default")
	}
	keys := ObjectOf(Of("z"), Num(1), Of("a"), Num(2), Of("m"), Num(3)).Keys()
	if len(keys) != 3 || keys[0] != "z" || keys[1] != "a" || keys[2] != "m" {
		t.Errorf("Keys = %v, want the order they were set in", keys)
	}
}

func TestAssertStringThrows(t *testing.T) {
	cases := []struct {
		in   Value
		want string
	}{
		{Num(1), "Expected a string but received a Number"},
		{Undefined, "Expected a string but received a undefined"},
		{Null, "Expected a string but received a null"},
		{true, "Expected a string but received a Boolean"},
		{[]Value{}, "Expected a string but received a Array"},
		{NewObject(), "Expected a string but received a Object"},
	}
	for _, c := range cases {
		func() {
			defer func() {
				r := recover()
				thrown, ok := r.(*Thrown)
				if !ok {
					t.Errorf("AssertString(%v): did not throw", c.in)
					return
				}
				if thrown.Name != "TypeError" || thrown.Message != c.want {
					t.Errorf("AssertString(%v) threw %s: %s, want TypeError: %s",
						c.in, thrown.Name, thrown.Message, c.want)
				}
			}()
			AssertString(c.in)
		}()
	}
	// A string does not throw.
	AssertString(Of("a"))
}

func TestParseJSON(t *testing.T) {
	for _, text := range []string{"null", "true", "1", "1e3", `"a"`, "[]", "[1,2]", "{}", `{"a":1}`,
		`{"a":[1,{"b":null}]}`, " 1 ", `"é"`, `"😀"`} {
		if _, err := ParseJSON(jsstr.Of(text)); err != nil {
			t.Errorf("ParseJSON(%q): %v", text, err)
		}
	}
	for _, text := range []string{"", " ", "{", "[1,]", "'a'", "01", "+1", ".5", "1.", "NaN", "undefined",
		`{a:1}`, `{"a":1,}`, "[1 2]", `"\x41"`, "tru"} {
		if _, err := ParseJSON(jsstr.Of(text)); err == nil {
			t.Errorf("ParseJSON(%q): no error, want one", text)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b Value
		op   string
		want bool
	}{
		{Num(1), Num(2), "<", true},
		{Of("a"), Of("b"), "<", true},
		{Of("10"), Of("9"), "<", true},
		{Of("10"), Num(9), "<", false},
		{Num(math.NaN()), Num(1), "<", false},
		{Num(math.NaN()), Num(1), ">", false},
		{Undefined, Num(1), "<", false},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b, c.op); got != c.want {
			t.Errorf("Compare(%v, %v, %q) = %v, want %v", c.a, c.b, c.op, got, c.want)
		}
	}
}
