// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"math"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

// ToDate reads the string as a date, giving null when it is not one.
func ToDate(input jsval.Value) jsval.Value {
	s := str(input)
	parsed := jsval.ParseDate(s)
	if math.IsNaN(parsed) {
		return jsval.Null
	}
	return jsval.NewDate(parsed)
}

// ToFloat reads the string as a floating point number, giving not-a-number when it is not one.
func ToFloat(input jsval.Value) jsval.Value {
	if !jsval.Truthy(IsFloat(input, jsval.Undefined)) {
		return math.NaN()
	}
	return jsstr.ParseFloat(jsval.Str(input))
}

// ToInt reads the string as an integer in the given radix, ten by default.
func ToInt(input, radix jsval.Value) jsval.Value {
	s := str(input)
	base := or(radix, float64(10))
	return jsstr.ParseInt(s, int(jsstr.ToInt32(jsval.ToNumber(base))))
}

var reTrue = re("^true$", "i")

var reFalse = re("^false$", "i")

// ToBoolean reads the string as a boolean. Strictly, only "1" and "true" are true; otherwise everything but "0",
// "false" and the empty string is.
func ToBoolean(input, strict jsval.Value) jsval.Value {
	s := str(input)
	if jsval.Truthy(strict) {
		return s.Is("1") || reTrue.Test(s)
	}
	return !s.Is("0") && !reFalse.Test(s) && len(s) != 0
}

// Equals reports whether the string is exactly the comparison value.
func Equals(input, comparison jsval.Value) jsval.Value {
	s := str(input)
	return jsval.StrictEquals(s, comparison)
}

var containsDefaults = jsval.ObjectOf(jsval.Of("ignoreCase"), false, jsval.Of("minOccurrences"), float64(1))

// Contains reports whether the element occurs in the string at least as often as the options ask.
func Contains(input, elem, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, containsDefaults)
	if jsval.Truthy(opts.Get("ignoreCase")) {
		parts := s.ToLower().SplitStr(jsval.ToStringUtil(elem).ToLower())
		return jsval.Compare(float64(len(parts)), opts.Get("minOccurrences"), ">")
	}
	parts := s.SplitStr(jsval.ToStringUtil(elem))
	return jsval.Compare(float64(len(parts)), opts.Get("minOccurrences"), ">")
}

// Matches reports whether the pattern matches the string. The pattern may be a regular expression or the text of
// one, with the modifiers as its flags.
func Matches(input, pattern, modifiers jsval.Value) jsval.Value {
	s := str(input)
	// A pattern that is already a regular expression is used as it is, flags and all; only a pattern given as
	// something else is built with the modifiers.
	compiled, ok := pattern.(*jsre.Regexp)
	if !ok {
		compiled = newRegExp(pattern, modifiers)
	}
	all, m := jsre.Matches(s, compiled)
	return all != nil || m != nil
}

var (
	strictBooleans = []string{"true", "false", "1", "0"}
	looseBooleans  = []string{"true", "false", "1", "0", "yes", "no"}
)

// IsBoolean reports whether the string is a boolean. Loosely, "yes" and "no" count as well.
func IsBoolean(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(options) {
		options = jsval.ObjectOf(jsval.Of("loose"), false)
	}
	if jsval.Truthy(prop(options, "loose")) {
		return includesText(looseBooleans, s.ToLower())
	}
	return includesText(strictBooleans, s)
}

func includesText(list []string, s jsstr.String) bool {
	for _, item := range list {
		if s.Is(item) {
			return true
		}
	}
	return false
}

var isEmptyDefaults = jsval.ObjectOf(jsval.Of("ignore_whitespace"), false)

// IsEmpty reports whether the string has no characters, optionally ignoring the white space around them.
func IsEmpty(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, isEmptyDefaults)
	if jsval.Truthy(opts.Get("ignore_whitespace")) {
		return len(s.Trim()) == 0
	}
	return len(s) == 0
}

var (
	rePresentation = re("[^\\uFE0F\\uFE0E][\\uFE0F\\uFE0E]", "g")
	reSurrogate    = re("[\\uD800-\\uDBFF][\\uDC00-\\uDFFF]", "g")
)

// IsLength reports whether the string is between the given lengths, counting a surrogate pair and a presentation
// sequence as one character. The lengths may be given as options or as the second and third arguments.
func IsLength(args []jsval.Value) jsval.Value {
	s := str(arg(args, 0))
	options := arg(args, 1)
	var min, max jsval.Value = float64(0), jsval.Undefined
	if jsval.TypeOf(options) == "object" {
		min = or(prop(options, "min"), float64(0))
		max = prop(options, "max")
	} else {
		min = or(arg(args, 1), float64(0))
		max = arg(args, 2)
	}
	presentation, _ := jsre.Matches(s, rePresentation)
	surrogates, _ := jsre.Matches(s, reSurrogate)
	length := float64(len(s) - len(presentation) - len(surrogates))
	inside := jsval.Compare(length, min, ">=") && (jsval.IsUndefined(max) || jsval.Compare(length, max, "<="))
	if inside {
		if discrete, ok := jsval.Array(optionalProp(options, "discreteLengths")); ok {
			for _, item := range discrete {
				if jsval.StrictEquals(item, length) {
					return true
				}
			}
			return false
		}
	}
	return inside
}

// optionalProp reads a property with the optional chaining operator: a null or undefined value answers undefined
// instead of raising.
func optionalProp(v jsval.Value, key string) jsval.Value {
	if jsval.IsNullOrUndefined(v) {
		return jsval.Undefined
	}
	return jsval.Prop(v, key)
}

var reURIOctet = re("%..|.", "")

// IsByteLength reports whether the string takes between the given numbers of bytes once it is encoded.
func IsByteLength(args []jsval.Value) jsval.Value {
	s := str(arg(args, 0))
	options := arg(args, 1)
	var min, max jsval.Value
	if jsval.TypeOf(options) == "object" {
		min = or(prop(options, "min"), float64(0))
		max = prop(options, "max")
	} else {
		min = arg(args, 1)
		max = arg(args, 2)
	}
	encoded := jsstr.Of(jsval.EncodeURI(s))
	length := float64(len(jsre.Split(encoded, reURIOctet)) - 1)
	return jsval.Compare(length, min, ">=") && (jsval.IsUndefined(max) || jsval.Compare(length, max, "<="))
}

// IsIn reports whether the string is one of the values: a member of an array, a key of an object or a part of a
// longer string.
func IsIn(input, options jsval.Value) jsval.Value {
	s := str(input)
	if list, ok := jsval.Array(options); ok {
		for _, item := range list {
			if s.Equal(jsval.ToStringUtil(item)) {
				return true
			}
		}
		return false
	}
	if jsval.TypeOf(options) == "object" {
		if jsval.IsNull(options) {
			jsval.Throw("TypeError", "Cannot read properties of null (reading 'hasOwnProperty')")
		}
		if has(options, "hasOwnProperty") {
			jsval.Throw("TypeError", "options.hasOwnProperty is not a function")
		}
		return has(options, s.Go())
	}
	if text, ok := options.(jsstr.String); ok && jsval.Truthy(options) {
		return text.IndexOf(s, 0) >= 0
	}
	return false
}

// IsDivisibleBy reports whether the number the string holds divides by the given number with nothing left over.
func IsDivisibleBy(input, num jsval.Value) jsval.Value {
	s := str(input)
	_ = s
	left := jsval.ToNumber(ToFloat(input))
	right := jsstr.ParseInt(jsval.ToString(num), 10)
	return math.Mod(left, right) == 0
}

var (
	reInt             = re("^(?:[-+]?(?:0|[1-9][0-9]*))$", "")
	reIntLeadingZeros = re("^[-+]?[0-9]+$", "")
)

// IsInt reports whether the string is an integer, within the bounds the options give. The bounds are compared the
// way the source language compares them, so a bound given as text is compared as text.
func IsInt(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := options
	if !jsval.Truthy(opts) {
		opts = jsval.NewObject()
	}
	pattern := reIntLeadingZeros
	if jsval.StrictEquals(prop(opts, "allow_leading_zeroes"), false) {
		pattern = reInt
	}
	minOK := !has(opts, "min") || jsval.IsNullOrUndefined(prop(opts, "min")) || jsval.Compare(s, prop(opts, "min"), ">=")
	maxOK := !has(opts, "max") || jsval.IsNullOrUndefined(prop(opts, "max")) || jsval.Compare(s, prop(opts, "max"), "<=")
	ltOK := !has(opts, "lt") || jsval.IsNullOrUndefined(prop(opts, "lt")) || jsval.Compare(s, prop(opts, "lt"), "<")
	gtOK := !has(opts, "gt") || jsval.IsNullOrUndefined(prop(opts, "gt")) || jsval.Compare(s, prop(opts, "gt"), ">")
	return pattern.Test(s) && minOK && maxOK && ltOK && gtOK
}

// IsPort reports whether the string is a port number.
func IsPort(input jsval.Value) jsval.Value {
	return IsInt(input, jsval.ObjectOf(
		jsval.Of("allow_leading_zeroes"), false,
		jsval.Of("min"), float64(0),
		jsval.Of("max"), float64(65535),
	))
}

var reHexadecimal = re("^(0x|0h)?[0-9A-F]+$", "i")

// IsHexadecimal reports whether the string is a hexadecimal number.
func IsHexadecimal(input jsval.Value) jsval.Value { return reHexadecimal.Test(str(input)) }

var reOctal = re("^(0o)?[0-7]+$", "i")

// IsOctal reports whether the string is an octal number.
func IsOctal(input jsval.Value) jsval.Value { return reOctal.Test(str(input)) }

// IsLowercase reports whether the string is unchanged by lowering its case.
func IsLowercase(input jsval.Value) jsval.Value {
	s := str(input)
	return s.Equal(s.ToLower())
}

// IsUppercase reports whether the string is unchanged by raising its case.
func IsUppercase(input jsval.Value) jsval.Value {
	s := str(input)
	return s.Equal(s.ToUpper())
}

var reAscii = re("^[\\x00-\\x7F]+$", "")

// IsAscii reports whether every character of the string is an ASCII one.
func IsAscii(input jsval.Value) jsval.Value { return reAscii.Test(str(input)) }

var reFullWidth = re("[^\\u0020-\\u007E\\uFF61-\\uFF9F\\uFFA0-\\uFFDC\\uFFE8-\\uFFEE0-9a-zA-Z]", "")

// IsFullWidth reports whether the string holds a full width character.
func IsFullWidth(input jsval.Value) jsval.Value { return reFullWidth.Test(str(input)) }

var reHalfWidth = re("[\\u0020-\\u007E\\uFF61-\\uFF9F\\uFFA0-\\uFFDC\\uFFE8-\\uFFEE0-9a-zA-Z]", "")

// IsHalfWidth reports whether the string holds a half width character.
func IsHalfWidth(input jsval.Value) jsval.Value { return reHalfWidth.Test(str(input)) }

// IsVariableWidth reports whether the string holds both a full width and a half width character.
func IsVariableWidth(input jsval.Value) jsval.Value {
	s := str(input)
	return reFullWidth.Test(s) && reHalfWidth.Test(s)
}

var reMultibyte = re("[^\\x00-\\x7F]", "")

// IsMultibyte reports whether the string holds a character outside ASCII.
func IsMultibyte(input jsval.Value) jsval.Value { return reMultibyte.Test(str(input)) }

var reSurrogatePair = re("[\\uD800-\\uDBFF][\\uDC00-\\uDFFF]", "")

// IsSurrogatePair reports whether the string holds a surrogate pair.
func IsSurrogatePair(input jsval.Value) jsval.Value { return reSurrogatePair.Test(str(input)) }

var reSemVer = re("^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)"+
	"(?:-((?:0|[1-9]\\d*|\\d*[a-z-][0-9a-z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-z-][0-9a-z-]*))*))"+
	"?(?:\\+([0-9a-z-]+(?:\\.[0-9a-z-]+)*))?$", "i")

// IsSemVer reports whether the string is a semantic version.
func IsSemVer(input jsval.Value) jsval.Value { return reSemVer.Test(str(input)) }

// IsWhitelisted reports whether every character of the string is in the given characters.
func IsWhitelisted(input, chars jsval.Value) jsval.Value {
	s := str(input)
	for i := len(s) - 1; i >= 0; i-- {
		if indexOfValue(chars, s.CharAt(i)) == -1 {
			return false
		}
	}
	return true
}

// indexOfValue is the indexOf method of a string or an array, which is what the characters may be given as.
func indexOfValue(haystack jsval.Value, needle jsstr.String) int {
	switch t := haystack.(type) {
	case jsstr.String:
		return t.IndexOf(needle, 0)
	case []jsval.Value:
		for i, item := range t {
			if jsval.StrictEquals(item, needle) {
				return i
			}
		}
		return -1
	}
	if jsval.IsNullOrUndefined(haystack) {
		jsval.Throw("TypeError", "Cannot read properties of "+jsval.ToString(haystack).Go()+
			" (reading 'indexOf')")
	}
	jsval.Throw("TypeError", "chars.indexOf is not a function")
	return -1
}

var reSlug = re("^[a-z0-9](?!.*[-_]{2,})(?:[a-z0-9_-]*[a-z0-9])?$", "")

// IsSlug reports whether the string is a slug.
func IsSlug(input jsval.Value) jsval.Value { return reSlug.Test(str(input)) }

var isJSONDefaults = jsval.ObjectOf(
	jsval.Of("allow_primitives"), false,
	jsval.Of("allow_any_value"), false,
)

// IsJSON reports whether the string is JSON text. By default only an object or an array counts.
func IsJSON(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, isJSONDefaults)
	value, err := jsval.ParseJSON(s)
	if err != nil {
		return false
	}
	if jsval.Truthy(opts.Get("allow_any_value")) {
		return true
	}
	if jsval.Truthy(opts.Get("allow_primitives")) {
		if jsval.IsNull(value) || jsval.StrictEquals(value, false) || jsval.StrictEquals(value, true) {
			return true
		}
	}
	return jsval.Truthy(value) && jsval.TypeOf(value) == "object"
}

// IsMongoId reports whether the string is a MongoDB identifier.
func IsMongoId(input jsval.Value) jsval.Value {
	s := str(input)
	return jsval.Truthy(IsHexadecimal(input)) && len(s) == 24
}

var reULID = re("^[0-7][0-9A-HJKMNP-TV-Z]{25}$", "i")

// IsULID reports whether the string is a lexicographically sortable identifier.
func IsULID(input jsval.Value) jsval.Value { return reULID.Test(str(input)) }

var uuidPatterns = map[string]*jsre.Regexp{
	"1":     re("^[0-9A-F]{8}-[0-9A-F]{4}-1[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"2":     re("^[0-9A-F]{8}-[0-9A-F]{4}-2[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"3":     re("^[0-9A-F]{8}-[0-9A-F]{4}-3[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"4":     re("^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"5":     re("^[0-9A-F]{8}-[0-9A-F]{4}-5[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"6":     re("^[0-9A-F]{8}-[0-9A-F]{4}-6[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"7":     re("^[0-9A-F]{8}-[0-9A-F]{4}-7[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"8":     re("^[0-9A-F]{8}-[0-9A-F]{4}-8[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$", "i"),
	"nil":   re("^00000000-0000-0000-0000-000000000000$", "i"),
	"max":   re("^ffffffff-ffff-ffff-ffff-ffffffffffff$", "i"),
	"loose": re("^[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}$", "i"),
	"all": re("^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}"+
		"|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$", "i"),
}

// IsUUID reports whether the string is a universally unique identifier of the given version.
func IsUUID(input, version jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsNullOrUndefined(version) {
		version = jsval.Of("all")
	}
	pattern, ok := uuidPatterns[jsval.ToString(version).Go()]
	if !ok {
		return false
	}
	return pattern.Test(s)
}

var reMD5 = re("^[a-f0-9]{32}$", "")

// IsMD5 reports whether the string is an MD5 digest.
func IsMD5(input jsval.Value) jsval.Value { return reMD5.Test(str(input)) }

var hashLengths = map[string]int{
	"md5": 32, "md4": 32, "sha1": 40, "sha256": 64, "sha384": 96, "sha512": 128,
	"ripemd128": 32, "ripemd160": 40, "tiger128": 32, "tiger160": 40, "tiger192": 48,
	"crc32": 8, "crc32b": 8,
}

// IsHash reports whether the string is a digest of the given algorithm. An algorithm the library does not know
// leaves the length out of the pattern, which then matches the text of the pattern itself.
func IsHash(input, algorithm jsval.Value) jsval.Value {
	s := str(input)
	length := "undefined"
	if n, ok := hashLengths[jsval.ToString(algorithm).Go()]; ok {
		length = jsstr.NumberToString(float64(n))
	}
	return newRegExp(jsval.Of("^[a-fA-F0-9]{"+length+"}$"), jsval.Undefined).Test(s)
}

// IsJWT reports whether the string is a JSON web token.
func IsJWT(input jsval.Value) jsval.Value {
	s := str(input)
	parts := s.Split(".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if !jsval.Truthy(IsBase64(part, jsval.ObjectOf(jsval.Of("urlSafe"), true))) {
			return false
		}
	}
	return true
}

var (
	reBase32          = re("^[A-Z2-7]+=*$", "")
	reCrockfordBase32 = re("^[A-HJKMNP-TV-Z0-9]+$", "")
	base32Defaults    = jsval.ObjectOf(jsval.Of("crockford"), false)
)

// IsBase32 reports whether the string is base 32 text, in the Crockford alphabet when the options ask for it.
func IsBase32(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, base32Defaults)
	if jsval.Truthy(opts.Get("crockford")) {
		return reCrockfordBase32.Test(s)
	}
	return len(s)%8 == 0 && reBase32.Test(s)
}

var reBase58 = re("^[A-HJ-NP-Za-km-z1-9]*$", "")

// IsBase58 reports whether the string is base 58 text.
func IsBase58(input jsval.Value) jsval.Value { return reBase58.Test(str(input)) }

var (
	reBase64WithPadding       = re("^[A-Za-z0-9+/]+={0,2}$", "")
	reBase64WithoutPadding    = re("^[A-Za-z0-9+/]+$", "")
	reBase64URLWithPadding    = re("^[A-Za-z0-9_-]+={0,2}$", "")
	reBase64URLWithoutPadding = re("^[A-Za-z0-9_-]+$", "")
)

// IsBase64 reports whether the string is base 64 text, in the URL safe alphabet when the options ask for it.
func IsBase64(input, options jsval.Value) jsval.Value {
	s := str(input)
	defaults := jsval.ObjectOf(
		jsval.Of("urlSafe"), false,
		jsval.Of("padding"), !jsval.Truthy(optionalProp(options, "urlSafe")),
	)
	opts := jsval.Merge(options, defaults)
	if len(s) == 0 {
		return true
	}
	padding := jsval.Truthy(opts.Get("padding"))
	if padding && len(s)%4 != 0 {
		return false
	}
	var pattern *jsre.Regexp
	if jsval.Truthy(opts.Get("urlSafe")) {
		pattern = reBase64URLWithoutPadding
		if padding {
			pattern = reBase64URLWithPadding
		}
	} else {
		pattern = reBase64WithoutPadding
		if padding {
			pattern = reBase64WithPadding
		}
	}
	return (!padding || len(s)%4 == 0) && pattern.Test(s)
}

var reMagnetURI = re("(?:^magnet:\\?|[^?&]&)xt(?:\\.1)?=urn:(?:(?:aich|bitprint|btih|ed2k|ed2khash|kzhash|md5|"+
	"sha1|tree:tiger):[a-z0-9]{32}(?:[a-z0-9]{8})?|btmh:1220[a-z0-9]{64})(?:$|&)", "i")

// IsMagnetURI reports whether the string is a magnet link.
func IsMagnetURI(input jsval.Value) jsval.Value {
	s := str(input)
	if s.Index("magnet:?") != 0 {
		return false
	}
	return reMagnetURI.Test(s)
}

var reEthereum = re("^(0x)[0-9a-f]{40}$", "i")

// IsEthereumAddress reports whether the string is an Ethereum address.
func IsEthereumAddress(input jsval.Value) jsval.Value { return reEthereum.Test(str(input)) }

var (
	reBech32    = re("^(bc1|tb1|bc1p|tb1p)[ac-hj-np-z02-9]{39,58}$", "")
	reBtcBase58 = re("^(1|2|3|m)[A-HJ-NP-Za-km-z1-9]{25,39}$", "")
)

// IsBtcAddress reports whether the string is a Bitcoin address.
func IsBtcAddress(input jsval.Value) jsval.Value {
	s := str(input)
	return reBech32.Test(s) || reBtcBase58.Test(s)
}

var (
	reHexColor           = re("^#?([0-9A-F]{3}|[0-9A-F]{4}|[0-9A-F]{6}|[0-9A-F]{8})$", "i")
	reHexColorWithPrefix = re("^#([0-9A-F]{3}|[0-9A-F]{4}|[0-9A-F]{6}|[0-9A-F]{8})$", "i")
	hexColorDefaults     = jsval.ObjectOf(jsval.Of("require_hashtag"), false)
)

// IsHexColor reports whether the string is a colour written in hexadecimal.
func IsHexColor(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, hexColorDefaults)
	if jsval.Truthy(opts.Get("require_hashtag")) {
		return reHexColorWithPrefix.Test(s)
	}
	return reHexColor.Test(s)
}

const hslNumber = "((\\+|\\-)?([0-9]+(\\.[0-9]+)?(e(\\+|\\-)?[0-9]+)?|\\.[0-9]+(e(\\+|\\-)?[0-9]+)?))"

var (
	reHSLComma = re("^hsla?\\("+hslNumber+"(deg|grad|rad|turn)?(,(\\+|\\-)?([0-9]+(\\.[0-9]+)?(e(\\+|\\-)?[0-9]+)?|"+
		"\\.[0-9]+(e(\\+|\\-)?[0-9]+)?)%){2}(,((\\+|\\-)?([0-9]+(\\.[0-9]+)?(e(\\+|\\-)?[0-9]+)?|"+
		"\\.[0-9]+(e(\\+|\\-)?[0-9]+)?)%?))?\\)$", "i")
	reHSLSpace = re("^hsla?\\("+hslNumber+"(deg|grad|rad|turn)?(\\s(\\+|\\-)?([0-9]+(\\.[0-9]+)?(e(\\+|\\-)?[0-9]+)?|"+
		"\\.[0-9]+(e(\\+|\\-)?[0-9]+)?)%){2}\\s?(\\/\\s((\\+|\\-)?([0-9]+(\\.[0-9]+)?(e(\\+|\\-)?[0-9]+)?|"+
		"\\.[0-9]+(e(\\+|\\-)?[0-9]+)?)%?)\\s?)?\\)$", "i")
	reHSLSpaces = re("\\s+", "g")
	reHSLAround = re("\\s?(hsla?\\(|\\)|,)\\s?", "ig")
)

// IsHSL reports whether the string is a colour written in hue, saturation and lightness.
func IsHSL(input jsval.Value) jsval.Value {
	s := str(input)
	stripped := jsre.Replace(jsre.Replace(s, reHSLSpaces, jsstr.Of(" ")), reHSLAround, jsstr.Of("$1"))
	if stripped.Index(",") != -1 {
		return reHSLComma.Test(stripped)
	}
	return reHSLSpace.Test(stripped)
}

var (
	reRGB = re("^rgb\\((([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5]),){2}"+
		"([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\\)$", "")
	reRGBA = re("^rgba\\((([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5]),){3}"+
		"(0?\\.\\d\\d?|1(\\.0)?|0(\\.0)?)\\)$", "")
	reRGBPercent  = re("^rgb\\((([0-9]%|[1-9][0-9]%|100%),){2}([0-9]%|[1-9][0-9]%|100%)\\)$", "")
	reRGBAPercent = re("^rgba\\((([0-9]%|[1-9][0-9]%|100%),){3}(0?\\.\\d\\d?|1(\\.0)?|0(\\.0)?)\\)$", "")
	reStartsRGB   = re("^rgba?", "")
	reSpace       = re("\\s", "g")
)

// IsRgbColor reports whether the string is a colour written in red, green and blue.
func IsRgbColor(args []jsval.Value) jsval.Value {
	s := str(arg(args, 0))
	options := arg(args, 1)
	allowSpaces := false
	includePercentValues := true
	if jsval.TypeOf(options) != "object" {
		if len(args) >= 2 {
			includePercentValues = jsval.Truthy(args[1])
		}
	} else {
		if !jsval.IsUndefined(prop(options, "allowSpaces")) {
			allowSpaces = jsval.Truthy(prop(options, "allowSpaces"))
		}
		if !jsval.IsUndefined(prop(options, "includePercentValues")) {
			includePercentValues = jsval.Truthy(prop(options, "includePercentValues"))
		}
	}
	if allowSpaces {
		if !reStartsRGB.Test(s) {
			return false
		}
		s = jsre.Replace(s, reSpace, jsstr.String{})
	}
	if !includePercentValues {
		return reRGB.Test(s) || reRGBA.Test(s)
	}
	return reRGB.Test(s) || reRGBA.Test(s) || reRGBPercent.Test(s) || reRGBAPercent.Test(s)
}

var (
	reLat     = re("^\\(?[+-]?(90(\\.0+)?|[1-8]?\\d(\\.\\d+)?)$", "")
	reLong    = re("^\\s?[+-]?(180(\\.0+)?|1[0-7]\\d(\\.\\d+)?|\\d{1,2}(\\.\\d+)?)\\)?$", "")
	reLatDMS  = re("^(([1-8]?\\d)\\D+([1-5]?\\d|60)\\D+([1-5]?\\d|60)(\\.\\d+)?|90\\D+0\\D+0)\\D+[NSns]?$", "i")
	reLongDMS = re("^\\s*([1-7]?\\d{1,2}\\D+([1-5]?\\d|60)\\D+([1-5]?\\d|60)(\\.\\d+)?|180\\D+0\\D+0)\\D+[EWew]?$", "i")

	latLongDefaults = jsval.ObjectOf(jsval.Of("checkDMS"), false)
)

// IsLatLong reports whether the string is a pair of coordinates, in degrees, minutes and seconds when the options
// ask for it.
func IsLatLong(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, latLongDefaults)
	if !s.Contains(",") {
		return false
	}
	pair := s.Split(",")
	if pair[0].HasPrefix("(") && !pair[1].HasSuffix(")") || pair[1].HasSuffix(")") && !pair[0].HasPrefix("(") {
		return false
	}
	if jsval.Truthy(opts.Get("checkDMS")) {
		return reLatDMS.Test(pair[0]) && reLongDMS.Test(pair[1])
	}
	return reLat.Test(pair[0]) && reLong.Test(pair[1])
}

var (
	reMimeSimple = re("^(application|audio|font|image|message|model|multipart|text|video)"+
		"\\/[a-zA-Z0-9\\.\\-\\+_]{1,100}$", "i")
	reMimeText = re("^text\\/[a-zA-Z0-9\\.\\-\\+]{1,100};\\s?charset=(\"[a-zA-Z0-9\\.\\-\\+\\s]{0,70}\"|"+
		"[a-zA-Z0-9\\.\\-\\+]{0,70})(\\s?\\([a-zA-Z0-9\\.\\-\\+\\s]{1,20}\\))?$", "i")
	reMimeMultipart = re("^multipart\\/[a-zA-Z0-9\\.\\-\\+]{1,100}(;\\s?(boundary|charset)="+
		"(\"[a-zA-Z0-9\\.\\-\\+\\s]{0,70}\"|[a-zA-Z0-9\\.\\-\\+]{0,70})"+
		"(\\s?\\([a-zA-Z0-9\\.\\-\\+\\s]{1,20}\\))?){0,2}$", "i")
)

// IsMimeType reports whether the string is a media type.
func IsMimeType(input jsval.Value) jsval.Value {
	s := str(input)
	return reMimeSimple.Test(s) || reMimeText.Test(s) || reMimeMultipart.Test(s)
}

var (
	reValidMediaType = re("^[a-z]+\\/[a-z0-9\\-\\+\\._]+$", "i")
	reValidAttribute = re("^[a-z\\-]+=[a-z0-9\\-]+$", "i")
	reValidData      = re("^[a-z0-9!\\$&'\\(\\)\\*\\+,;=\\-\\._~:@\\/\\?%\\s]*$", "i")
)

// IsDataURI reports whether the string is a data link.
func IsDataURI(input jsval.Value) jsval.Value {
	s := str(input)
	data := s.Split(",")
	if len(data) < 2 {
		return false
	}
	attributes := data[0].Trim().Split(";")
	data = data[1:]
	schemeAndMediaType := attributes[0]
	attributes = attributes[1:]
	if !schemeAndMediaType.Slice(0, 5).Is("data:") {
		return false
	}
	mediaType := schemeAndMediaType.From(5)
	if len(mediaType) != 0 && !reValidMediaType.Test(mediaType) {
		return false
	}
	for i, attribute := range attributes {
		if !(i == len(attributes)-1 && attribute.ToLower().Is("base64")) && !reValidAttribute.Test(attribute) {
			return false
		}
	}
	for _, part := range data {
		if !reValidData.Test(part) {
			return false
		}
	}
	return true
}

var reISRC = re("^[A-Z]{2}[0-9A-Z]{3}\\d{2}\\d{5}$", "")

// IsISRC reports whether the string is an international standard recording code.
func IsISRC(input jsval.Value) jsval.Value { return reISRC.Test(str(input)) }

var (
	reMAC48             = re("^(?:[0-9a-fA-F]{2}([-:\\s]))([0-9a-fA-F]{2}\\1){4}([0-9a-fA-F]{2})$", "")
	reMAC48NoSeparators = re("^([0-9a-fA-F]){12}$", "")
	reMAC48WithDots     = re("^([0-9a-fA-F]{4}\\.){2}([0-9a-fA-F]{4})$", "")
	reMAC64             = re("^(?:[0-9a-fA-F]{2}([-:\\s]))([0-9a-fA-F]{2}\\1){6}([0-9a-fA-F]{2})$", "")
	reMAC64NoSeparators = re("^([0-9a-fA-F]){16}$", "")
	reMAC64WithDots     = re("^([0-9a-fA-F]{4}\\.){3}([0-9a-fA-F]{4})$", "")
)

// IsMACAddress reports whether the string is a hardware address, of the width the options ask for.
func IsMACAddress(input, options jsval.Value) jsval.Value {
	s := str(input)
	eui := optionalProp(options, "eui")
	if jsval.Truthy(eui) {
		eui = jsval.ToString(eui)
		object(options).Set("eui", eui)
	}
	if jsval.Truthy(optionalProp(options, "no_colons")) || jsval.Truthy(optionalProp(options, "no_separators")) {
		if jsval.StrictEquals(eui, jsval.Of("48")) {
			return reMAC48NoSeparators.Test(s)
		}
		if jsval.StrictEquals(eui, jsval.Of("64")) {
			return reMAC64NoSeparators.Test(s)
		}
		return reMAC48NoSeparators.Test(s) || reMAC64NoSeparators.Test(s)
	}
	if jsval.StrictEquals(eui, jsval.Of("48")) {
		return reMAC48.Test(s) || reMAC48WithDots.Test(s)
	}
	if jsval.StrictEquals(eui, jsval.Of("64")) {
		return reMAC64.Test(s) || reMAC64WithDots.Test(s)
	}
	return jsval.Truthy(IsMACAddress(input, jsval.ObjectOf(jsval.Of("eui"), jsval.Of("48")))) ||
		jsval.Truthy(IsMACAddress(input, jsval.ObjectOf(jsval.Of("eui"), jsval.Of("64"))))
}
