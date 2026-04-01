// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

// Package jsval is the value model the validators work with. The library they are ported from takes options of
// any type and reacts to the type it is handed, so the ported code keeps that: a Value is undefined, null, a
// boolean, a number, a string, an array, an object, a regular expression or a date, and the conversions and the
// comparisons here are the ones the source language defines.
package jsval

import (
	"math"
	"strings"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
)

// Value is any value a validator may be handed or may return.
type Value interface{}

type undefinedType struct{}

type nullType struct{}

// Undefined is the value a missing argument or a missing property has.
var Undefined Value = undefinedType{}

// Null is the null value.
var Null Value = nullType{}

// Object is an object with its keys in the order they were added, which is the order the keys of an object are
// reported in.
type Object struct {
	keys   []string
	values map[string]Value
}

// NewObject makes an empty object.
func NewObject() *Object { return &Object{values: map[string]Value{}} }

// Set adds or replaces a property.
func (o *Object) Set(key string, value Value) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// Get returns a property, or undefined when the object does not have it.
func (o *Object) Get(key string) Value {
	if o == nil {
		return Undefined
	}
	if v, ok := o.values[key]; ok {
		return v
	}
	return Undefined
}

// Has reports whether the object has the property as its own.
func (o *Object) Has(key string) bool {
	if o == nil {
		return false
	}
	_, ok := o.values[key]
	return ok
}

// Keys returns the property names in the order they were added.
func (o *Object) Keys() []string {
	if o == nil {
		return nil
	}
	return o.keys
}

// Thrown is an error the library throws, with the name and the message the source runtime gives it.
type Thrown struct {
	// Name is the error's constructor name, such as TypeError.
	Name string
	// Message is the message property.
	Message string
}

// Error returns the thrown error as "Name: message", the way the runtime prints it.
func (t *Thrown) Error() string { return t.Name + ": " + t.Message }

// Throw raises an error of the given name, which the package boundary turns back into a returned error.
func Throw(name, message string) { panic(&Thrown{Name: name, Message: message}) }

// TypeOf is the typeof operator: an object, an array, a date and a regular expression are all "object".
func TypeOf(v Value) string {
	switch v.(type) {
	case undefinedType:
		return "undefined"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case jsstr.String:
		return "string"
	case *BigInt:
		return "bigint"
	}
	return "object"
}

// ConstructorName is the name of the value's constructor, which the library puts in the message of the error it
// throws when it is not handed a string.
func ConstructorName(v Value) string {
	switch v.(type) {
	case undefinedType:
		return "undefined"
	case nullType:
		return "null"
	case bool:
		return "Boolean"
	case float64:
		return "Number"
	case jsstr.String:
		return "String"
	case []Value:
		return "Array"
	case *jsre.Regexp:
		return "RegExp"
	case *Date:
		return "Date"
	case *BigInt:
		return "BigInt"
	}
	return "Object"
}

// IsString reports whether the value is a string.
func IsString(v Value) bool {
	_, ok := v.(jsstr.String)
	return ok
}

// IsUndefined reports whether the value is undefined.
func IsUndefined(v Value) bool {
	_, ok := v.(undefinedType)
	return ok
}

// IsNull reports whether the value is null.
func IsNull(v Value) bool {
	_, ok := v.(nullType)
	return ok
}

// IsNullOrUndefined reports whether the value is null or undefined.
func IsNullOrUndefined(v Value) bool { return IsNull(v) || IsUndefined(v) }

// Str returns the value as a string, which only makes sense once it is known to be one.
func Str(v Value) jsstr.String {
	if s, ok := v.(jsstr.String); ok {
		return s
	}
	return jsstr.String{}
}

// AssertString raises the type error the library raises when what it was handed is not a string.
func AssertString(v Value) {
	if IsUndefined(v) || IsNull(v) {
		Throw("TypeError", "Expected a string but received a "+ConstructorName(v))
	}
	if !IsString(v) {
		Throw("TypeError", "Expected a string but received a "+ConstructorName(v))
	}
}

// Truthy is how a value reads in a condition.
func Truthy(v Value) bool {
	switch t := v.(type) {
	case undefinedType, nullType:
		return false
	case bool:
		return t
	case float64:
		return t != 0 && !math.IsNaN(t)
	case jsstr.String:
		return len(t) > 0
	}
	return true
}

// ToNumber converts a value to a number the way the language does.
func ToNumber(v Value) float64 {
	switch t := v.(type) {
	case undefinedType:
		return math.NaN()
	case nullType:
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case float64:
		return t
	case jsstr.String:
		return jsstr.ToNumber(t)
	case []Value:
		if len(t) == 0 {
			return 0
		}
		if len(t) == 1 {
			return ToNumber(t[0])
		}
		return math.NaN()
	case *Date:
		return t.Time
	}
	return math.NaN()
}

// ToString converts a value to a string the way the language does.
func ToString(v Value) jsstr.String {
	switch t := v.(type) {
	case undefinedType:
		return jsstr.Of("undefined")
	case nullType:
		return jsstr.Of("null")
	case bool:
		if t {
			return jsstr.Of("true")
		}
		return jsstr.Of("false")
	case float64:
		return jsstr.Of(jsstr.NumberToString(t))
	case jsstr.String:
		return t
	case []Value:
		parts := make([]string, len(t))
		for i, item := range t {
			if IsNullOrUndefined(item) {
				continue
			}
			parts[i] = ToString(item).Go()
		}
		return jsstr.Of(strings.Join(parts, ","))
	case *jsre.Regexp:
		return jsstr.Of("/" + t.Source + "/" + t.Flags)
	case *Date:
		return jsstr.Of(t.String())
	}
	return jsstr.Of("[object Object]")
}

// ToStringUtil is the library's own toString helper, which turns a value that is not a number and has no length
// into the empty string before converting it.
func ToStringUtil(v Value) jsstr.String {
	switch t := v.(type) {
	case undefinedType, nullType:
		return jsstr.String{}
	case *Object:
		return jsstr.Of("[object Object]")
	case []Value, *Date, *jsre.Regexp:
		return ToString(v)
	case float64:
		if math.IsNaN(t) {
			return jsstr.String{}
		}
	case bool:
		return ToString(v)
	case jsstr.String:
		return t
	}
	return ToString(v)
}

// StrictEquals is the === operator.
func StrictEquals(a, b Value) bool {
	switch x := a.(type) {
	case undefinedType:
		return IsUndefined(b)
	case nullType:
		return IsNull(b)
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case jsstr.String:
		y, ok := b.(jsstr.String)
		return ok && x.Equal(y)
	}
	return a == b
}

// Compare is a relational operator: op is "<", "<=", ">" or ">=". Two strings are compared by their code units
// and anything else as numbers, with a comparison against a value that is not a number always false.
func Compare(a, b Value, op string) bool {
	left, leftIsString := a.(jsstr.String)
	right, rightIsString := b.(jsstr.String)
	if leftIsString && rightIsString {
		order := compareStrings(left, right)
		switch op {
		case "<":
			return order < 0
		case "<=":
			return order <= 0
		case ">":
			return order > 0
		default:
			return order >= 0
		}
	}
	x, y := ToNumber(a), ToNumber(b)
	if math.IsNaN(x) || math.IsNaN(y) {
		return false
	}
	switch op {
	case "<":
		return x < y
	case "<=":
		return x <= y
	case ">":
		return x > y
	default:
		return x >= y
	}
}

func compareStrings(a, b jsstr.String) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// Num makes a number value.
func Num(f float64) Value { return f }

// Of makes a string value out of UTF-8 text.
func Of(s string) Value { return jsstr.Of(s) }

// Array returns the value as an array, and false when it is not one.
func Array(v Value) ([]Value, bool) {
	a, ok := v.([]Value)
	return a, ok
}

// Obj returns the value as an object, and false when it is not one.
func Obj(v Value) (*Object, bool) {
	o, ok := v.(*Object)
	return o, ok
}

// Prop reads a property of a value: objects answer from their own properties, a string answers its length, and
// anything else answers undefined.
func Prop(v Value, key string) Value {
	switch t := v.(type) {
	case *Object:
		return t.Get(key)
	case jsstr.String:
		if key == "length" {
			return float64(len(t))
		}
	case []Value:
		if key == "length" {
			return float64(len(t))
		}
	}
	return Undefined
}

// Merge fills in the properties the options do not have from the defaults, as the library's merge helper does.
// Options that are not an object are replaced by an empty one.
func Merge(options Value, defaults *Object) *Object {
	out, ok := options.(*Object)
	if !ok || TypeOf(options) != "object" || IsNull(options) {
		out = NewObject()
	}
	for _, key := range defaults.Keys() {
		if IsUndefined(out.Get(key)) {
			out.Set(key, defaults.Get(key))
		}
	}
	return out
}

// ObjectOf builds an object from pairs of a key and a value, keeping the order they are given in.
func ObjectOf(pairs ...Value) *Object {
	out := NewObject()
	for i := 0; i+1 < len(pairs); i += 2 {
		out.Set(Str(pairs[i]).Go(), pairs[i+1])
	}
	return out
}
