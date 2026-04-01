// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

// Package validator checks and cleans up strings: it is the Go form of the validator.js library, keeping its
// behaviour, its options and the quirks of the language it was written in. Every validator takes the string to
// check and, where it has them, its options, and the sanitizers return the cleaned string.
//
// The values the functions take and give back are jsval.Value, because the library reacts to the type of what it
// is handed; Lookup and Call reach a function by its name, which is what the command does.
package validator

import (
	"fmt"
	"sort"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

// Version is the version of the library this one was ported from.
const Version = "13.15.35"

// Func is a function of the library reached by name.
type Func func(args []jsval.Value) jsval.Value

type member struct {
	fn    Func
	value jsval.Value
}

var (
	members = map[string]member{}
	names   []string
)

// register records a function under its name, keeping the order the names are registered in.
func register(name string, fn Func) {
	if _, ok := members[name]; !ok {
		names = append(names, name)
	}
	members[name] = member{fn: fn}
}

// registerValue records a value that is not a function, such as a list of locales.
func registerValue(name string, value jsval.Value) {
	if _, ok := members[name]; !ok {
		names = append(names, name)
	}
	members[name] = member{value: value}
}

// Names lists everything the library exposes, in the order the library itself reports.
func Names() []string { return append([]string(nil), names...) }

// Lookup returns what the library exposes under a name, and false when it exposes nothing under it.
func Lookup(name string) (jsval.Value, bool) {
	m, ok := members[name]
	if !ok {
		return nil, false
	}
	if m.fn != nil {
		return m.fn, true
	}
	return m.value, true
}

// IsFunc reports whether the name belongs to a function of the library.
func IsFunc(name string) bool {
	m, ok := members[name]
	return ok && m.fn != nil
}

// Call runs a function of the library by name. An error the library throws comes back as a *jsval.Thrown.
func Call(name string, args []jsval.Value) (result jsval.Value, err error) {
	m, ok := members[name]
	if !ok {
		return nil, &jsval.Thrown{Name: "Error", Message: fmt.Sprintf("unknown name '%s'", name)}
	}
	if m.fn == nil {
		return nil, &jsval.Thrown{Name: "Error", Message: fmt.Sprintf("'%s' is not a function", name)}
	}
	defer func() {
		if r := recover(); r != nil {
			thrown, ok := r.(*jsval.Thrown)
			if !ok {
				panic(r)
			}
			result, err = nil, thrown
		}
	}()
	return m.fn(args), nil
}

// arg returns the argument at i, or undefined when the call did not reach that far.
func arg(args []jsval.Value, i int) jsval.Value {
	if i < len(args) {
		return args[i]
	}
	return jsval.Undefined
}

// str asserts that the value is a string and returns it.
func str(v jsval.Value) jsstr.String {
	jsval.AssertString(v)
	return jsval.Str(v)
}

// prop reads a property, raising the type error the runtime raises when the value is null or undefined.
func prop(v jsval.Value, key string) jsval.Value {
	if jsval.IsNull(v) {
		jsval.Throw("TypeError", fmt.Sprintf("Cannot read properties of null (reading '%s')", key))
	}
	if jsval.IsUndefined(v) {
		jsval.Throw("TypeError", fmt.Sprintf("Cannot read properties of undefined (reading '%s')", key))
	}
	return jsval.Prop(v, key)
}

// or is the || operator: the left value unless it is falsy.
func or(a, b jsval.Value) jsval.Value {
	if jsval.Truthy(a) {
		return a
	}
	return b
}

// has reports whether the value has the property as its own, the way hasOwnProperty does.
func has(v jsval.Value, key string) bool {
	o, ok := v.(*jsval.Object)
	return ok && o.Has(key)
}

// options returns the value as an object, or an empty one, without the filling in that Merge does.
func object(v jsval.Value) *jsval.Object {
	if o, ok := v.(*jsval.Object); ok {
		return o
	}
	return jsval.NewObject()
}

// sortedNames is used by the tests to check that nothing is registered twice.
func sortedNames() []string {
	out := Names()
	sort.Strings(out)
	return out
}

// newRegExp builds a pattern from values, the way the RegExp constructor does, raising a syntax error with the
// runtime's message when the pattern is not one.
func newRegExp(pattern, modifiers jsval.Value) *jsre.Regexp {
	if re, ok := pattern.(*jsre.Regexp); ok && jsval.IsUndefined(modifiers) {
		return re
	}
	source := ""
	if !jsval.IsUndefined(pattern) {
		source = jsval.ToString(pattern).Go()
	}
	flags := ""
	if !jsval.IsUndefined(modifiers) {
		flags = jsval.ToString(modifiers).Go()
	}
	re, err := jsre.Compile(source, flags)
	if err != nil {
		jsval.Throw("SyntaxError", err.(*jsre.SyntaxError).Message)
	}
	return re
}

// re builds a pattern that is part of the library itself and is known to be valid.
func re(source, flags string) *jsre.Regexp { return jsre.MustCompile(source, flags) }
