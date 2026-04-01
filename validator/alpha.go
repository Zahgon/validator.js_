// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

// localeRegexp is one row of a per-locale pattern table.
type localeRegexp struct {
	locale string
	source string
	flags  string
}

// localeText is one row of a per-locale text table.
type localeText struct {
	locale string
	text   string
}

// patternTable holds per-locale patterns and remembers the order the locales were added in, which is the order
// the library reports its locale lists in.
type patternTable struct {
	keys     []string
	patterns map[string]*jsre.Regexp
}

func newPatternTable(entries []localeRegexp) *patternTable {
	t := &patternTable{patterns: map[string]*jsre.Regexp{}}
	for _, e := range entries {
		if _, ok := t.patterns[e.locale]; !ok {
			t.keys = append(t.keys, e.locale)
		}
		t.patterns[e.locale] = re(e.source, e.flags)
	}
	return t
}

// Has reports whether the table knows the locale.
func (t *patternTable) Has(locale string) bool {
	_, ok := t.patterns[locale]
	return ok
}

// Keys lists the locales in the order they were added.
func (t *patternTable) Keys() []string { return t.keys }

// Locales returns the locale names as a value, the way the library exposes them.
func (t *patternTable) Locales() jsval.Value {
	out := make([]jsval.Value, len(t.keys))
	for i, k := range t.keys {
		out[i] = jsval.Of(k)
	}
	return out
}

// textTable holds per-locale text and remembers the order the locales were added in.
type textTable struct {
	keys   []string
	values map[string]string
}

func newTextTable(entries []localeText) *textTable {
	t := &textTable{values: map[string]string{}}
	for _, e := range entries {
		if _, ok := t.values[e.locale]; !ok {
			t.keys = append(t.keys, e.locale)
		}
		t.values[e.locale] = e.text
	}
	return t
}

// Has reports whether the table knows the locale.
func (t *textTable) Has(locale string) bool {
	_, ok := t.values[locale]
	return ok
}

// Keys lists the locales in the order they were added.
func (t *textTable) Keys() []string { return t.keys }

// Locales returns the locale names as a value.
func (t *textTable) Locales() jsval.Value {
	out := make([]jsval.Value, len(t.keys))
	for i, k := range t.keys {
		out[i] = jsval.Of(k)
	}
	return out
}

var (
	alphaTable        = newPatternTable(alphaEntries)
	alphanumericTable = newPatternTable(alphanumericEntries)
	decimalTable      = newTextTable(decimalEntries)
)

var reIgnoreEscape = re("[-[\\]{}()*+?.,\\\\^$|#\\\\s]", "g")

// applyIgnore removes the characters the ignore option names before the locale pattern is tried. The option is
// either a pattern or the characters themselves, which are put into a character class after escaping.
func applyIgnore(s jsstr.String, options jsval.Value) jsstr.String {
	if jsval.IsNull(options) {
		jsval.Throw("TypeError", "Cannot destructure property 'ignore' of 'options' as it is null.")
	}
	ignore := jsval.Prop(options, "ignore")
	if !jsval.Truthy(ignore) {
		return s
	}
	if pattern, ok := ignore.(*jsre.Regexp); ok {
		return jsre.Replace(s, pattern, jsstr.String{})
	}
	if text, ok := ignore.(jsstr.String); ok {
		escaped := jsre.Replace(text, reIgnoreEscape, jsstr.Of("\\$&"))
		return jsre.Replace(s, newRegExp(jsval.Of("["+escaped.Go()+"]"), jsval.Of("g")), jsstr.String{})
	}
	jsval.Throw("Error", "ignore should be instance of a String or RegExp")
	return s
}

// IsAlpha reports whether the string holds only letters of the locale, which is American English by default.
func IsAlpha(input, locale, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(locale) {
		locale = jsval.Of("en-US")
	}
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	s = applyIgnore(s, options)
	name := jsval.ToString(locale).Go()
	if alphaTable.Has(name) {
		return alphaTable.patterns[name].Test(s)
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}

// IsAlphanumeric reports whether the string holds only letters and digits of the locale.
func IsAlphanumeric(input, locale, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(locale) {
		locale = jsval.Of("en-US")
	}
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	s = applyIgnore(s, options)
	name := jsval.ToString(locale).Go()
	if alphanumericTable.Has(name) {
		return alphanumericTable.patterns[name].Test(s)
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}

// IsFloat reports whether the string is a floating point number, within the bounds the options give and with the
// decimal mark of the locale.
func IsFloat(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := options
	if !jsval.Truthy(opts) {
		opts = jsval.NewObject()
	}
	mark := "."
	if locale := prop(opts, "locale"); jsval.Truthy(locale) {
		mark = decimalTable.values[jsval.ToString(locale).Go()]
	}
	pattern := newRegExp(jsval.Of("^(?:[-+])?(?:[0-9]+)?(?:\\"+mark+"[0-9]*)?(?:[eE][\\+\\-]?(?:[0-9]+))?$"),
		jsval.Undefined)
	if s.Is("") || s.Is(".") || s.Is(",") || s.Is("-") || s.Is("+") {
		return false
	}
	value := jsstr.ParseFloat(jsre.Replace(s, reComma, jsstr.Of(".")))
	return pattern.Test(s) &&
		(!has(opts, "min") || jsval.IsNullOrUndefined(prop(opts, "min")) || jsval.Compare(value, prop(opts, "min"), ">=")) &&
		(!has(opts, "max") || jsval.IsNullOrUndefined(prop(opts, "max")) || jsval.Compare(value, prop(opts, "max"), "<=")) &&
		(!has(opts, "lt") || jsval.IsNullOrUndefined(prop(opts, "lt")) || jsval.Compare(value, prop(opts, "lt"), "<")) &&
		(!has(opts, "gt") || jsval.IsNullOrUndefined(prop(opts, "gt")) || jsval.Compare(value, prop(opts, "gt"), ">"))
}

var reComma = re(",", "")

var isDecimalDefaults = jsval.ObjectOf(
	jsval.Of("force_decimal"), false,
	jsval.Of("decimal_digits"), jsval.Of("1,"),
	jsval.Of("locale"), jsval.Of("en-US"),
)

var reSpaces = re(" ", "g")

// IsDecimal reports whether the string is a decimal number with the decimal mark of the locale and the number of
// decimal digits the options ask for.
func IsDecimal(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, isDecimalDefaults)
	name := jsval.ToString(opts.Get("locale")).Go()
	if decimalTable.Has(name) {
		stripped := jsre.Replace(s, reSpaces, jsstr.String{})
		if stripped.Is("") || stripped.Is("-") || stripped.Is("+") {
			return false
		}
		force := ""
		if !jsval.Truthy(opts.Get("force_decimal")) {
			force = "?"
		}
		pattern := newRegExp(jsval.Of("^[-+]?([0-9]+)?(\\"+decimalTable.values[name]+"[0-9]{"+
			jsval.ToString(opts.Get("decimal_digits")).Go()+"})"+force+"$"), jsval.Undefined)
		return pattern.Test(s)
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}

var reNumericNoSymbols = re("^[0-9]+$", "")

// IsNumeric reports whether the string is a number, with the decimal mark of the locale.
func IsNumeric(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.Truthy(options) && jsval.Truthy(prop(options, "no_symbols")) {
		return reNumericNoSymbols.Test(s)
	}
	mark := "."
	if jsval.Truthy(options) {
		if locale := prop(options, "locale"); jsval.Truthy(locale) {
			// A locale the table does not hold reads as the undefined value, which the source joins into the
			// pattern as the text of that value, so the class becomes the letters of "undefined".
			var ok bool
			if mark, ok = decimalTable.values[jsval.ToString(locale).Go()]; !ok {
				mark = "undefined"
			}
		}
	}
	return newRegExp(jsval.Of("^[+-]?([0-9]*["+mark+"])?[0-9]+$"), jsval.Undefined).Test(s)
}

var reLtrimEscape = re("[.*+?^${}()|[\\]\\\\]", "g")

var reLeadingSpace = re("^\\s+", "g")

// Ltrim removes the given characters, or the white space, from the front of the string.
func Ltrim(input, chars jsval.Value) jsval.Value {
	s := str(input)
	pattern := reLeadingSpace
	if jsval.Truthy(chars) {
		escaped := jsre.Replace(jsval.Str(chars), reLtrimEscape, jsstr.Of("\\$&"))
		pattern = newRegExp(jsval.Of("^["+escaped.Go()+"]+"), jsval.Of("g"))
	}
	return jsre.Replace(s, pattern, jsstr.String{})
}

var reWhitespace = re("\\s", "")

// Rtrim removes the given characters, or the white space, from the end of the string.
func Rtrim(input, chars jsval.Value) jsval.Value {
	s := str(input)
	if jsval.Truthy(chars) {
		escaped := jsre.Replace(jsval.Str(chars), reLtrimEscape, jsstr.Of("\\$&"))
		return jsre.Replace(s, newRegExp(jsval.Of("["+escaped.Go()+"]+$"), jsval.Of("g")), jsstr.String{})
	}
	index := len(s) - 1
	for reWhitespace.Test(s.CharAt(index)) {
		index--
	}
	return s.Slice(0, index+1)
}

// Trim removes the given characters, or the white space, from both ends of the string.
func Trim(input, chars jsval.Value) jsval.Value {
	return Rtrim(Ltrim(input, chars), chars)
}

// Blacklist removes every character named by the characters, which are read as a character class.
func Blacklist(input, chars jsval.Value) jsval.Value {
	s := str(input)
	pattern := newRegExp(jsval.Of("["+jsval.ToString(chars).Go()+"]+"), jsval.Of("g"))
	return jsre.Replace(s, pattern, jsstr.String{})
}

// Whitelist removes every character the characters do not name.
func Whitelist(input, chars jsval.Value) jsval.Value {
	s := str(input)
	pattern := newRegExp(jsval.Of("[^"+jsval.ToString(chars).Go()+"]+"), jsval.Of("g"))
	return jsre.Replace(s, pattern, jsstr.String{})
}

var escapeSteps = []struct {
	pattern *jsre.Regexp
	with    string
}{
	{re("&", "g"), "&amp;"},
	{re("\"", "g"), "&quot;"},
	{re("'", "g"), "&#x27;"},
	{re("<", "g"), "&lt;"},
	{re(">", "g"), "&gt;"},
	{re("\\/", "g"), "&#x2F;"},
	{re("\\\\", "g"), "&#x5C;"},
	{re("`", "g"), "&#96;"},
}

// Escape replaces the characters that have a meaning in HTML with their references.
func Escape(input jsval.Value) jsval.Value {
	s := str(input)
	for _, step := range escapeSteps {
		s = jsre.Replace(s, step.pattern, jsstr.Of(step.with))
	}
	return s
}

var unescapeSteps = []struct {
	pattern *jsre.Regexp
	with    string
}{
	{re("&quot;", "g"), "\""},
	{re("&#x27;", "g"), "'"},
	{re("&lt;", "g"), "<"},
	{re("&gt;", "g"), ">"},
	{re("&#x2F;", "g"), "/"},
	{re("&#x5C;", "g"), "\\"},
	{re("&#96;", "g"), "`"},
	{re("&amp;", "g"), "&"},
}

// Unescape puts back the characters Escape replaced, the ampersand last so that a reference written inside
// another one does not come apart.
func Unescape(input jsval.Value) jsval.Value {
	s := str(input)
	for _, step := range unescapeSteps {
		s = jsre.Replace(s, step.pattern, jsstr.Of(step.with))
	}
	return s
}

// StripLow removes the control characters, keeping the line breaks when it is asked to.
func StripLow(input, keepNewLines jsval.Value) jsval.Value {
	chars := "\\x00-\\x1F\\x7F"
	if jsval.Truthy(keepNewLines) {
		chars = "\\x00-\\x09\\x0B\\x0C\\x0E-\\x1F\\x7F"
	}
	return Blacklist(input, jsval.Of(chars))
}

const (
	localeExtlang    = "([A-Za-z]{3}(-[A-Za-z]{3}){0,2})"
	localeLanguage   = "(([a-zA-Z]{2,3}(-" + localeExtlang + ")?)|([a-zA-Z]{5,8}))"
	localeScript     = "([A-Za-z]{4})"
	localeRegion     = "([A-Za-z]{2}|\\d{3})"
	localeVariant    = "([A-Za-z0-9]{5,8}|(\\d[A-Z-a-z0-9]{3}))"
	localeSingleton  = "(\\d|[A-W]|[Y-Z]|[a-w]|[y-z])"
	localeExtension  = "(" + localeSingleton + "(-[A-Za-z0-9]{2,8})+)"
	localePrivateUse = "(x(-[A-Za-z0-9]{1,8})+)"
	localeIrregular  = "((en-GB-oed)|(i-ami)|(i-bnn)|(i-default)|(i-enochian)|" +
		"(i-hak)|(i-klingon)|(i-lux)|(i-mingo)|(i-navajo)|(i-pwn)|(i-tao)|" +
		"(i-tay)|(i-tsu)|(sgn-BE-FR)|(sgn-BE-NL)|(sgn-CH-DE))"
	localeRegular = "((art-lojban)|(cel-gaulish)|(no-bok)|(no-nyn)|(zh-guoyu)|" +
		"(zh-hakka)|(zh-min)|(zh-min-nan)|(zh-xiang))"
	localeGrandfathered = "(" + localeIrregular + "|" + localeRegular + ")"
	localeDelimiter     = "(-|_)"
	localeLangtag       = localeLanguage + "(" + localeDelimiter + localeScript + ")?(" + localeDelimiter +
		localeRegion + ")?(" + localeDelimiter + localeVariant + ")*(" + localeDelimiter +
		localeExtension + ")*(" + localeDelimiter + localePrivateUse + ")?"
)

var reLanguageTag = re("(^"+localePrivateUse+"$)|(^"+localeGrandfathered+"$)|(^"+localeLangtag+"$)", "")

// IsLocale reports whether the string is a language tag.
func IsLocale(input jsval.Value) jsval.Value { return reLanguageTag.Test(str(input)) }
