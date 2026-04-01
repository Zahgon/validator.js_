// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

// spec is one case of the library's own test suite: a function, the arguments after the string, and the strings
// it must accept, reject or raise an error for, or, for a sanitizer, what it must give back for each string.
type spec struct {
	Validator string            `json:"validator"`
	Sanitizer string            `json:"sanitizer"`
	Args      []json.RawMessage `json:"args"`
	Valid     []json.RawMessage `json:"valid"`
	Invalid   []json.RawMessage `json:"invalid"`
	Error     []json.RawMessage `json:"error"`
	Expect    map[string]json.RawMessage
}

// rawSpec reads the Expect map, whose keys are the strings handed to a sanitizer.
func (s *spec) UnmarshalJSON(data []byte) error {
	type plain spec
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	var holder struct {
		Expect map[string]json.RawMessage `json:"expect"`
	}
	if err := json.Unmarshal(data, &holder); err != nil {
		return err
	}
	*s = spec(p)
	s.Expect = holder.Expect
	return nil
}

// value turns one JSON value of the suite into a value of the library, reading the markers the suite writes for
// what JSON cannot hold: a pattern, a date, the undefined value and the number that is not one.
func value(t *testing.T, raw json.RawMessage) jsval.Value {
	t.Helper()
	parsed, err := jsval.ParseJSON(jsstr.Of(string(raw)))
	if err != nil {
		t.Fatalf("test data is not JSON: %v", err)
	}
	return resolve(t, parsed)
}

func resolve(t *testing.T, v jsval.Value) jsval.Value {
	t.Helper()
	switch item := v.(type) {
	case []jsval.Value:
		out := make([]jsval.Value, len(item))
		for i, element := range item {
			out[i] = resolve(t, element)
		}
		return out
	case *jsval.Object:
		if item.Has("$") {
			switch jsval.ToString(item.Get("$")).Go() {
			case "undefined":
				return jsval.Undefined
			case "NaN":
				return math.NaN()
			case "regexp":
				flags := ""
				if !jsval.IsUndefined(item.Get("flags")) {
					flags = jsval.ToString(item.Get("flags")).Go()
				}
				return jsre.MustCompile(jsval.ToString(item.Get("source")).Go(), flags)
			case "date":
				return jsval.NewDate(jsval.ToNumber(item.Get("value")))
			}
		}
		out := jsval.NewObject()
		for _, key := range item.Keys() {
			out.Set(key, resolve(t, item.Get(key)))
		}
		return out
	}
	return v
}

// TestUpstreamSpecs runs every case of the library's own test suite through the ported functions.
func TestUpstreamSpecs(t *testing.T) {
	data, err := os.ReadFile("testdata/upstream.json")
	if err != nil {
		t.Fatal(err)
	}
	var specs []spec
	if err := json.Unmarshal(data, &specs); err != nil {
		t.Fatal(err)
	}
	if len(specs) < 500 {
		t.Fatalf("expected the whole suite, got %d cases", len(specs))
	}
	cases := 0
	for _, s := range specs {
		name := s.Validator
		if name == "" {
			name = s.Sanitizer
		}
		rest := make([]jsval.Value, len(s.Args))
		for i, raw := range s.Args {
			rest[i] = value(t, raw)
		}
		call := func(input jsval.Value) (jsval.Value, error) {
			return Call(name, append([]jsval.Value{input}, rest...))
		}
		for _, raw := range s.Valid {
			cases++
			got, err := call(value(t, raw))
			if err != nil || got != jsval.Value(true) {
				t.Errorf("%s(%s, ...) = %v, %v; want true", name, raw, got, err)
			}
		}
		for _, raw := range s.Invalid {
			cases++
			got, err := call(value(t, raw))
			if err != nil || got != jsval.Value(false) {
				t.Errorf("%s(%s, ...) = %v, %v; want false", name, raw, got, err)
			}
		}
		for _, raw := range s.Error {
			cases++
			if _, err := call(value(t, raw)); err == nil {
				t.Errorf("%s(%s, ...) returned a value; want an error", name, raw)
			}
		}
		for input, raw := range s.Expect {
			cases++
			want := value(t, raw)
			got, err := call(jsval.Of(input))
			if err != nil {
				t.Errorf("%s(%q, ...) raised %v; want %v", name, input, err, want)
				continue
			}
			if !sameValue(got, want) {
				t.Errorf("%s(%q, ...) = %s; want %s", name, input, show(got), show(want))
			}
		}
	}
	if cases < 10000 {
		t.Errorf("ran %d assertions of the suite, expected more than ten thousand", cases)
	}
	t.Logf("%d cases, %d assertions", len(specs), cases)
}

// sameValue compares what a sanitizer gave back with what the suite expects, counting two numbers that are not
// numbers as the same, which is what the suite's own comparison does.
func sameValue(got, want jsval.Value) bool {
	a, aok := got.(float64)
	b, bok := want.(float64)
	if aok && bok && math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return jsval.StrictEquals(got, want)
}

func show(v jsval.Value) string {
	if text, ok := v.(jsstr.String); ok {
		return text.Quote()
	}
	return jsval.ToString(v).Go()
}
