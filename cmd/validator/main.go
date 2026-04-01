// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

// Command validator is the command line over the validator library: it runs one function, or a script of them,
// and prints what they give back. Nothing here validates or sanitizes anything itself.
//
// The encoding of the values and the shape of the script are described in the usage text and in the package
// documentation of the library.
package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
	"github.com/validatorjs/validator/validator"
)

const usage = `usage: validator-cli <command> [argument…]

  script              read JSON Lines operations from standard input
  call <fn> [arg…]    call one function, standard input first unless --no-input
  get <name>          print a property of the library
  list                print the library's property names
  version             print the versions
  help                print this usage
`

// usageError is a mistake in how the command was called, which is answered with the usage status.
type usageError struct{ message string }

// Error returns the usage message.
func (e *usageError) Error() string { return e.message }

func usagef(format string, args ...any) error {
	return &usageError{message: fmt.Sprintf(format, args...)}
}

// chainedError is an error raised by one step of a chain, which carries the step it happened in.
type chainedError struct{ message string }

// Error returns the chained error's message.
func (e *chainedError) Error() string { return e.message }

// funcName is the name a function of the library reports, which is the name of the function it was written as.
func funcName(name string) string {
	if name == "isFreightContainerID" {
		return "isISO6346"
	}
	return name
}

// encode writes a value the way the command prints it.
func encode(v jsval.Value) string {
	switch t := v.(type) {
	case nil:
		return "undefined"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return "#" + numberText(t)
	case jsstr.String:
		return t.Quote()
	case *jsval.BigInt:
		return "#" + t.Text() + "n"
	case []jsval.Value:
		parts := make([]string, len(t))
		for i, item := range t {
			parts[i] = encode(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *jsval.Object:
		parts := make([]string, 0, len(t.Keys()))
		for _, key := range t.Keys() {
			parts = append(parts, jsstr.Of(key).Quote()+":"+encode(t.Get(key)))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case *jsval.Date:
		if !t.Valid() {
			return "@Invalid Date"
		}
		return "@" + t.ToISOString()
	case *jsre.Regexp:
		return "/" + t.Source + "/" + t.Flags
	}
	if jsval.IsUndefined(v) {
		return "undefined"
	}
	if jsval.IsNull(v) {
		return "null"
	}
	return "undefined"
}

// numberText writes a number, keeping the values the ordinary conversion hides.
func numberText(f float64) string {
	switch {
	case f != f:
		return "NaN"
	case f > 1.7976931348623157e308:
		return "Infinity"
	case f < -1.7976931348623157e308:
		return "-Infinity"
	case f == 0 && strings.HasPrefix(fmt.Sprintf("%g", 1/f), "-"):
		return "-0"
	}
	return jsstr.NumberToString(f)
}

// encodeThrown writes an error the way the command prints it.
func encodeThrown(err error) string {
	if thrown, ok := err.(*jsval.Thrown); ok {
		return "!" + thrown.Name + ": " + thrown.Message
	}
	return "!Error: " + err.Error()
}

// decode turns a value read from JSON into a value of the library, reading the markers for what JSON cannot
// write.
func decode(v jsval.Value) (jsval.Value, error) {
	switch t := v.(type) {
	case []jsval.Value:
		out := make([]jsval.Value, len(t))
		for i, item := range t {
			decoded, err := decode(item)
			if err != nil {
				return nil, err
			}
			out[i] = decoded
		}
		return out, nil
	case *jsval.Object:
		if t.Has("$") {
			return decodeMarker(t)
		}
		out := jsval.NewObject()
		for _, key := range t.Keys() {
			decoded, err := decode(t.Get(key))
			if err != nil {
				return nil, err
			}
			out.Set(key, decoded)
		}
		return out, nil
	}
	return v, nil
}

func decodeMarker(t *jsval.Object) (jsval.Value, error) {
	switch jsval.ToString(t.Get("$")).Go() {
	case "undefined":
		return jsval.Undefined, nil
	case "NaN":
		return nan(), nil
	case "Infinity":
		return infinity(1), nil
	case "-Infinity":
		return infinity(-1), nil
	case "-0":
		return negativeZero(), nil
	case "regexp":
		flags := ""
		if !jsval.IsUndefined(t.Get("flags")) {
			flags = jsval.ToString(t.Get("flags")).Go()
		}
		compiled, err := jsre.Compile(jsval.ToString(t.Get("source")).Go(), flags)
		if err != nil {
			return nil, &jsval.Thrown{Name: "SyntaxError", Message: err.Error()}
		}
		return compiled, nil
	case "bigint":
		value, err := decode(t.Get("value"))
		if err != nil {
			return nil, err
		}
		return jsval.NewBigInt(value)
	case "date":
		value, err := decode(t.Get("value"))
		if err != nil {
			return nil, err
		}
		if text, ok := value.(jsstr.String); ok {
			return jsval.NewDate(jsval.ParseDate(text)), nil
		}
		return jsval.NewDate(jsval.ToNumber(value)), nil
	}
	return nil, usagef("unknown marker '%s'", jsval.ToString(t.Get("$")).Go())
}

func nan() float64 { var zero float64; return zero / zero }
func infinity(sign int) float64 {
	var zero float64
	if sign < 0 {
		return -1 / zero
	}
	return 1 / zero
}

// negativeZero is the negative zero of the runtime, which a Go constant cannot hold.
func negativeZero() float64 { return math.Copysign(0, -1) }

// callFunction runs a function of the library by name.
func callFunction(name string, args []jsval.Value) (jsval.Value, error) {
	if !validator.IsFunc(name) {
		if _, ok := validator.Lookup(name); !ok {
			return nil, &jsval.Thrown{Name: "Error", Message: fmt.Sprintf("unknown name '%s'", name)}
		}
		return nil, &jsval.Thrown{Name: "Error", Message: fmt.Sprintf("'%s' is not a function", name)}
	}
	return validator.Call(name, args)
}

// lookup reads a property of the library.
func lookup(name string) (jsval.Value, error) {
	value, ok := validator.Lookup(name)
	if !ok {
		return nil, &jsval.Thrown{Name: "Error", Message: fmt.Sprintf("unknown name '%s'", name)}
	}
	if validator.IsFunc(name) {
		return functionValue(funcName(name)), nil
	}
	return value, nil
}

// functionValue stands for a function where a value is printed.
type functionValue string

// String returns the function's source text.
func (f functionValue) String() string { return string(f) }

// runOperation carries out one operation of a script.
func runOperation(op jsval.Value) (jsval.Value, error) {
	object, ok := op.(*jsval.Object)
	if !ok {
		return nil, usagef("operation is not an object")
	}
	if object.Has("fn") {
		args := []jsval.Value{}
		if raw := object.Get("args"); !jsval.IsUndefined(raw) {
			list, ok := jsval.Array(raw)
			if !ok {
				return nil, usagef("'args' is not an array")
			}
			for _, item := range list {
				decoded, err := decode(item)
				if err != nil {
					return nil, err
				}
				args = append(args, decoded)
			}
		}
		return callFunction(jsval.ToString(object.Get("fn")).Go(), args)
	}
	if object.Has("get") {
		return lookup(jsval.ToString(object.Get("get")).Go())
	}
	if object.Has("chain") {
		steps, ok := jsval.Array(object.Get("chain"))
		if !ok {
			return nil, usagef("'chain' is not an array")
		}
		value, err := decode(object.Get("input"))
		if err != nil {
			return nil, err
		}
		for i, raw := range steps {
			step, ok := jsval.Array(raw)
			if !ok || len(step) == 0 {
				return nil, usagef("step %d is not a non-empty array", i)
			}
			args := []jsval.Value{value}
			for _, item := range step[1:] {
				decoded, err := decode(item)
				if err != nil {
					return nil, err
				}
				args = append(args, decoded)
			}
			next, err := callFunction(jsval.ToString(step[0]).Go(), args)
			if err != nil {
				if _, bad := err.(*usageError); bad {
					return nil, err
				}
				return nil, &chainedError{message: fmt.Sprintf("step %d %s", i, encodeThrown(err)[1:])}
			}
			value = next
		}
		return value, nil
	}
	return nil, usagef("operation has none of 'fn', 'get' and 'chain'")
}

// script runs every operation of the standard input and prints one line for each.
func script(text jsstr.String) (int, error) {
	out := []string{}
	failed := false
	lines := text.Split("\n")
	for i, line := range lines {
		if len(line.Trim()) == 0 {
			continue
		}
		op, err := jsval.ParseJSON(line)
		if err != nil {
			return 0, usagef("line %d: %s", i+1, err.Error())
		}
		value, err := runOperation(op)
		if err != nil {
			if bad, ok := err.(*usageError); ok {
				return 0, usagef("line %d: %s", i+1, bad.message)
			}
			failed = true
			if chained, ok := err.(*chainedError); ok {
				out = append(out, "!"+chained.message)
			} else {
				out = append(out, encodeThrown(err))
			}
			continue
		}
		out = append(out, encodeValue(value))
	}
	if len(out) > 0 {
		fmt.Print(strings.Join(out, "\n") + "\n")
	}
	if failed {
		return 1, nil
	}
	return 0, nil
}

// encodeValue writes a value, including the stand-in for a function.
func encodeValue(v jsval.Value) string {
	if fn, ok := v.(functionValue); ok {
		return "<function " + string(fn) + ">"
	}
	return encode(v)
}

func readStdin() jsstr.String {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return jsstr.String{}
	}
	return jsstr.Of(string(data))
}

func run(argv []string) (int, error) {
	if len(argv) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2, nil
	}
	rest := argv[1:]
	switch argv[0] {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0, nil
	case "version", "--version", "-V":
		fmt.Printf("validator-cli 1.0.0 (validator.js %s)\n", validator.Version)
		return 0, nil
	case "list":
		if len(rest) > 0 {
			return 0, usagef("list takes no argument, got '%s'", rest[0])
		}
		fmt.Print(strings.Join(validator.Names(), "\n") + "\n")
		return 0, nil
	case "get":
		if len(rest) != 1 {
			return 0, usagef("get takes one name")
		}
		value, err := lookup(rest[0])
		if err != nil {
			return 0, err
		}
		fmt.Println(encodeValue(value))
		return 0, nil
	case "script":
		if len(rest) > 0 {
			return 0, usagef("script takes no argument, got '%s'", rest[0])
		}
		return script(readStdin())
	case "call":
		args := rest
		useStdin := true
		if len(args) > 0 && args[0] == "--no-input" {
			useStdin = false
			args = args[1:]
		}
		if len(args) == 0 {
			return 0, usagef("call takes a function name")
		}
		name := args[0]
		decoded := []jsval.Value{}
		for i, text := range args[1:] {
			parsed, err := jsval.ParseJSON(jsstr.Of(text))
			if err != nil {
				return 0, usagef("argument %d: %s", i+1, err.Error())
			}
			value, err := decode(parsed)
			if err != nil {
				return 0, err
			}
			decoded = append(decoded, value)
		}
		all := decoded
		if useStdin {
			all = append([]jsval.Value{readStdin()}, decoded...)
		}
		value, err := callFunction(name, all)
		if err != nil {
			fmt.Fprintf(os.Stderr, "validator-cli: %s\n", encodeThrown(err)[1:])
			return 1, nil
		}
		fmt.Println(encodeValue(value))
		return 0, nil
	}
	return 0, usagef("unknown command '%s'", argv[0])
}

func main() {
	status, err := run(os.Args[1:])
	if err != nil {
		if bad, ok := err.(*usageError); ok {
			fmt.Fprintf(os.Stderr, "validator-cli: %s\n", bad.message)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "validator-cli: %s\n", encodeThrown(err)[1:])
		os.Exit(1)
	}
	os.Exit(status)
}
