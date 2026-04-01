// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package main

import (
	"math"
	"testing"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

func TestEncode(t *testing.T) {
	cases := []struct {
		in   jsval.Value
		want string
	}{
		{jsval.Undefined, "undefined"},
		{jsval.Null, "null"},
		{true, "true"},
		{false, "false"},
		{jsval.Num(0), "#0"},
		{jsval.Num(negativeZero()), "#-0"},
		{jsval.Num(nan()), "#NaN"},
		{jsval.Num(math.Inf(1)), "#Infinity"},
		{jsval.Num(math.Inf(-1)), "#-Infinity"},
		{jsval.Num(1e21), "#1e+21"},
		{jsval.Of(""), `""`},
		{jsval.Of("a\nb"), `"a\nb"`},
		{jsval.Of("é"), `"é"`},
		{[]jsval.Value{}, "[]"},
		{[]jsval.Value{jsval.Of("a"), jsval.Num(1)}, `["a",#1]`},
		{jsval.NewObject(), "{}"},
		{jsval.ObjectOf(jsval.Of("b"), jsval.Num(2), jsval.Of("a"), jsval.Num(1)), `{"b":#2,"a":#1}`},
		{jsval.NewDate(0), "@1970-01-01T00:00:00.000Z"},
		{jsre.MustCompile("a", "gi"), "/a/gi"},
	}
	for _, c := range cases {
		if got := encode(c.in); got != c.want {
			t.Errorf("encode(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDecodeMarkers(t *testing.T) {
	value, err := decode(jsval.ObjectOf(jsval.Of("$"), jsval.Of("undefined")))
	if err != nil || !jsval.IsUndefined(value) {
		t.Errorf("decode($ undefined) = %v, %v", value, err)
	}
	value, err = decode(jsval.ObjectOf(jsval.Of("$"), jsval.Of("NaN")))
	if err != nil {
		t.Fatalf("decode($ NaN): %v", err)
	}
	if number, ok := value.(float64); !ok || !math.IsNaN(number) {
		t.Errorf("decode($ NaN) = %v, want NaN", value)
	}
	value, err = decode(jsval.ObjectOf(jsval.Of("$"), jsval.Of("regexp"),
		jsval.Of("source"), jsval.Of("^a$"), jsval.Of("flags"), jsval.Of("i")))
	if err != nil {
		t.Fatalf("decode($ regexp): %v", err)
	}
	re, ok := value.(*jsre.Regexp)
	if !ok {
		t.Fatalf("decode($ regexp) = %v, want a pattern", value)
	}
	if !re.Test(jsstr.Of("A")) {
		t.Error("the decoded pattern does not ignore case")
	}
	if _, err := decode(jsval.ObjectOf(jsval.Of("$"), jsval.Of("nope"))); err == nil {
		t.Error("decode($ nope): no error, want one")
	}
	// A marker is recognised at any depth, inside arrays and inside option objects.
	value, err = decode([]jsval.Value{jsval.ObjectOf(jsval.Of("min"),
		jsval.ObjectOf(jsval.Of("$"), jsval.Of("Infinity")))})
	if err != nil {
		t.Fatalf("decode nested: %v", err)
	}
	items, _ := jsval.Array(value)
	if len(items) != 1 || jsval.ToNumber(jsval.Prop(items[0], "min")) != math.Inf(1) {
		t.Errorf("decode nested = %v, want the infinite option", value)
	}
}

func TestNumberText(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{{0, "0"}, {negativeZero(), "-0"}, {nan(), "NaN"}, {math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"}, {1e21, "1e+21"}, {1e-7, "1e-7"}, {42, "42"}}
	for _, c := range cases {
		if got := numberText(c.in); got != c.want {
			t.Errorf("numberText(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRunUsageErrors(t *testing.T) {
	cases := []struct {
		argv []string
		code int
	}{
		{[]string{"help"}, 0},
		{[]string{"version"}, 0},
		{[]string{"list"}, 0},
		{[]string{}, 2},
		{[]string{"nope"}, 2},
		{[]string{"get"}, 2},
		{[]string{"call"}, 2},
		{[]string{"get", "nope"}, 1},
		{[]string{"list", "extra"}, 2},
	}
	for _, c := range cases {
		code, err := run(c.argv)
		if err != nil {
			if _, ok := err.(*usageError); ok {
				code = 2
			} else {
				code = 1
			}
		}
		if code != c.code {
			t.Errorf("run(%v) = %d, %v, want status %d", c.argv, code, err, c.code)
		}
	}
}

func TestScript(t *testing.T) {
	cases := []struct {
		in   string
		code int
		bad  bool // a usage error, which stops reading
	}{
		{"", 0, false},
		{"   \n\n", 0, false},
		{"{\"get\":\"version\"}\n", 0, false},
		{"{\"fn\":\"isEmail\",\"args\":[\"a@b.co\"]}\n", 0, false},
		{"{\"fn\":\"isEmail\",\"args\":[1]}\n", 1, false},
		{"{\"get\":\"nope\"}\n", 1, false},
		{"{\"chain\":[[\"trim\"],[\"isEmail\"]],\"input\":\" a@b.co \"}\n", 0, false},
		{"1\n", 0, true},
		{"[]\n", 0, true},
		{"{\"nope\":1}\n", 0, true},
		{"{\"fn\":\"isEmail\",\"args\":{}}\n", 0, true},
		{"{\"chain\":[[]],\"input\":\"a\"}\n", 0, true},
		{"{bad\n", 0, true},
	}
	for _, c := range cases {
		code, err := script(jsstr.Of(c.in))
		if c.bad {
			if _, ok := err.(*usageError); !ok {
				t.Errorf("script(%q) = %d, %v, want a usage error", c.in, code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("script(%q): %v", c.in, err)
			continue
		}
		if code != c.code {
			t.Errorf("script(%q) = %d, want %d", c.in, code, c.code)
		}
	}
}
