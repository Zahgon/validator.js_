# validator

A library of string validators and sanitizers for Go, ported from
[validator.js](https://github.com/validatorjs/validator.js) 13.15.35.

The port keeps the behaviour of the JavaScript library exactly, including its defects: the same validators and
sanitizers, the same options with the same defaults, the same locale tables, the same values handed back and the
same errors raised with the same messages. Because the original's answers rest on JavaScript's own semantics,
those are part of the port too and live in packages of their own: the regular expression dialect, number and
string conversion, Unicode case mapping, and date reading and writing.

Everything is computed with the Go standard library. There are no dependencies.

```
go get github.com/validatorjs/validator
```

## Packages

| Package | What it holds |
| --- | --- |
| `validator` | the validators, the sanitizers and the converters, with their options, locale tables and exported locale lists |
| `validator/jsre` | the regular expression dialect the validators are written in: look-around, back references, UTF-16 semantics, the `u` flag, the Unicode property escapes and the dialect's own error messages |
| `validator/jsstr` | strings as sequences of UTF-16 code units, the case mappings of the whole of Unicode, and number to string and string to number conversion |
| `validator/jsval` | the values the library passes around (undefined, booleans, numbers, strings, dates, patterns, arrays and option objects), date reading and `toISOString`, and JSON |
| `validator/cmd/validator` | the `validator` command, a thin layer over the library |

## Using the library

Every function of the library takes and returns a `jsval.Value`, because the original's answers depend on which
kind of value it was handed: an option that is absent, `undefined` or of the wrong type is not the same as one
that is set, and several functions raise an error instead of returning a boolean.

```go
package main

import (
	"fmt"

	"github.com/validatorjs/validator/jsval"
	"github.com/validatorjs/validator/validator"
)

func main() {
	ok, err := validator.Call("isEmail", []jsval.Value{jsval.Of("foo@bar.com")})
	if err != nil {
		// The library raised: err is a *jsval.Thrown with the name and message of the original.
		fmt.Println("raised:", err)
		return
	}
	fmt.Println(jsval.Truthy(ok)) // true
}
```

The options of a function are an object, built with `jsval.Object`:

```go
options := jsval.ObjectOf(
	jsval.Of("allow_display_name"), true,
	jsval.Of("host_blacklist"), []jsval.Value{jsval.Of("gmail.com")},
)
ok, err := validator.Call("isEmail", []jsval.Value{jsval.Of("Foo <a@gmail.com>"), options})
```

An option left out of the object takes the default of the original, and an option whose value is of the wrong
type behaves as the original's `merge` helper makes it behave. `jsval.Num`, `jsval.Undefined`, `jsval.Null`,
a `*jsval.Date`, a `*jsre.Regexp` and a `[]jsval.Value` are the other kinds of value the options take.

A sanitizer gives a value back rather than a verdict:

```go
value, err := validator.Call("normalizeEmail", []jsval.Value{jsval.Of("Foo.Bar+baz@GoogleMail.com")})
// value is the string "foobar@gmail.com"
```

`validator.Lookup` reads a property that is not a function, such as a locale list or the version:

```go
locales, _ := validator.Lookup("isMobilePhoneLocales")
version, _ := validator.Lookup("version")
```

`validator.Names` gives the library's property names in the order the original exports them, and
`validator.IsFunc` says whether a name is a function.

### The regular expression dialect

The standard library's regular expressions are a different language: they have no look-around and no back
references, they match over UTF-8 bytes rather than UTF-16 code units, their case folding is not the dialect's,
and they have no Unicode property escapes. `jsre` is the dialect itself, so that a pattern of the library, or one
a caller hands to `matches` or to the `ignore` option of `isAlpha`, is read and matched as the original reads and
matches it.

```go
re, err := jsre.Compile("^\\p{Script_Extensions=Greek}+$", "u")
if err != nil {
	// err is a *jsre.SyntaxError whose message is the dialect's own.
}
fmt.Println(re.Test(jsstr.Of("αβγ"))) // true
```

The property escapes cover every name the dialect accepts, in the spelling it accepts it: the binary properties
and their aliases, the general categories as a lone name or after `General_Category=` or `gc=`, and the scripts
and script extensions after `Script=`, `sc=`, `Script_Extensions=` or `scx=`. Their code points are in
`jsre/ucdtable.go`, which is generated: the standard library's `unicode` package has no table for script
extensions, none for most of the binary properties, and the tables it does have are of a different version of the
standard. The `v` flag and the properties of strings it adds are not supported.

## Using the command

`cmd/validator` exposes the library as a command, so that one invocation has a standard output, a standard error
and an exit status:

```
validator script              read JSON Lines operations from standard input, one result line each
validator call <fn> [arg…]    call one function; standard input is its first argument unless --no-input
validator get <name>          print a non-function property of the library
validator list                print the library's property names, in the library's own order
validator version             print the versions
validator help                print this usage
```

```console
$ printf 'foo@bar.com' | validator call isEmail
true
$ printf ' Foo.Bar+baz@GoogleMail.com ' | validator call normalizeEmail
"foobar@gmail.com"
$ printf '%s\n' '{"fn":"isInt","args":["42",{"min":50}]}' '{"chain":[["trim"],["isEmail"]],"input":" a@b.co "}' | validator script
false
true
```

Values are printed in one line each: `undefined` and `null` as themselves, a boolean as `true` or `false`, a
number as `#` and the number, a string as its JSON form, a date as `@` and its ISO form, a pattern as
`/source/flags`, an array as its bracketed elements and an object as its braced members. A raised error prints as
`!` and `name: message`. The exit status is 1 when an operation raised, 2 for a usage error and 0 otherwise.

Arguments are JSON, with an object of one key `$` standing for what JSON cannot write: `{"$":"undefined"}`,
`{"$":"NaN"}`, `{"$":"Infinity"}`, `{"$":"-0"}`, `{"$":"regexp","source":"…","flags":"…"}`,
`{"$":"date","value":…}` and `{"$":"bigint","value":"…"}`. The marker is recognised at any depth.

## Validators

`isAbaRouting`, `isAfter`, `isAlpha`, `isAlphanumeric`, `isAscii`, `isBase32`, `isBase58`, `isBase64`,
`isBefore`, `isBIC`, `isBoolean`, `isBtcAddress`, `isByteLength`, `isCreditCard`, `isCurrency`, `isDataURI`,
`isDate`, `isDecimal`, `isDivisibleBy`, `isEAN`, `isEmail`, `isEmpty`, `isEthereumAddress`, `isFloat`,
`isFQDN`, `isFreightContainerID`, `isFullWidth`, `isHalfWidth`, `isHash`, `isHexadecimal`, `isHexColor`,
`isHSL`, `isIBAN`, `isIdentityCard`, `isIMEI`, `isIn`, `isInt`, `isIP`, `isIPRange`, `isISBN`, `isISIN`,
`isISO4217`, `isISO6346`, `isISO6391`, `isISO8601`, `isISO15924`, `isISO31661Alpha2`, `isISO31661Alpha3`,
`isISO31661Numeric`, `isISRC`, `isISSN`, `isJSON`, `isJWT`, `isLatLong`, `isLength`, `isLicensePlate`,
`isLocale`, `isLowercase`, `isLuhnNumber`, `isMACAddress`, `isMagnetURI`, `isMailtoURI`, `isMD5`,
`isMimeType`, `isMobilePhone`, `isMongoId`, `isMultibyte`, `isNumeric`, `isOctal`, `isPassportNumber`,
`isPort`, `isPostalCode`, `isRFC3339`, `isRgbColor`, `isSemVer`, `isSlug`, `isStrongPassword`,
`isSurrogatePair`, `isTaxID`, `isTime`, `isULID`, `isUppercase`, `isURL`, `isUUID`, `isVariableWidth`,
`isVAT`, `isWhitelisted`, `contains`, `equals`, `matches`.

## Sanitizers and converters

`blacklist`, `escape`, `ltrim`, `normalizeEmail`, `rtrim`, `stripLow`, `toBoolean`, `toDate`, `toFloat`,
`toInt`, `trim`, `unescape`, `whitelist`.

## Locale lists

`isAlphaLocales`, `isAlphanumericLocales`, `isFloatLocales`, `isMobilePhoneLocales`, `isPostalCodeLocales`,
`passportNumberLocales`, `ibanLocales`.

## Tests

```
go build ./...
go vet ./...
go test ./...
```

The tests run offline. `validator/upstream_test.go` replays the assertions of the JavaScript suite from
`validator/testdata/upstream.json`, which holds the inputs of that suite, and the rest of the tests assert the
behaviour of each package directly.

## Licence

MIT, as the JavaScript library it is ported from; see [LICENSE](LICENSE). The Go files ported from it carry its
copyright notice.
