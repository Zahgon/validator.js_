// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

var (
	reUpperChar  = re("^[A-Z]$", "")
	reLowerChar  = re("^[a-z]$", "")
	reNumberChar = re("^[0-9]$", "")
	reSymbolChar = re("^[-#!$@\\u00a3%^&*()_+|~=`{}\\[\\]:\";'<>?,.\\/\\\\ ]$", "")
)

var strongPasswordDefaults = jsval.ObjectOf(
	jsval.Of("minLength"), float64(8),
	jsval.Of("minLowercase"), float64(1),
	jsval.Of("minUppercase"), float64(1),
	jsval.Of("minNumbers"), float64(1),
	jsval.Of("minSymbols"), float64(1),
	jsval.Of("returnScore"), false,
	jsval.Of("pointsPerUnique"), float64(1),
	jsval.Of("pointsPerRepeat"), 0.5,
	jsval.Of("pointsForContainingLower"), float64(10),
	jsval.Of("pointsForContainingUpper"), float64(10),
	jsval.Of("pointsForContainingNumber"), float64(10),
	jsval.Of("pointsForContainingSymbol"), float64(10),
)

// passwordAnalysis is what a password is made of: how long it is, how many different characters it uses and how
// many of its characters are of each kind.
type passwordAnalysis struct {
	length         float64
	uniqueChars    float64
	uppercaseCount float64
	lowercaseCount float64
	numberCount    float64
	symbolCount    float64
}

// analyzePassword counts the characters of the password by code point, the way Array.from walks a string.
func analyzePassword(password jsstr.String) passwordAnalysis {
	counts := map[string]float64{}
	order := []jsstr.String{}
	for i := 0; i < len(password); {
		_, width := password.CodePointAt(i)
		char := password.Slice(i, i+width)
		key := char.Go()
		if _, seen := counts[key]; !seen {
			order = append(order, char)
		}
		counts[key]++
		i += width
	}
	analysis := passwordAnalysis{length: float64(len(password)), uniqueChars: float64(len(order))}
	for _, char := range order {
		count := counts[char.Go()]
		switch {
		case reUpperChar.Test(char):
			analysis.uppercaseCount += count
		case reLowerChar.Test(char):
			analysis.lowercaseCount += count
		case reNumberChar.Test(char):
			analysis.numberCount += count
		case reSymbolChar.Test(char):
			analysis.symbolCount += count
		}
	}
	return analysis
}

// scorePassword turns the analysis into points, the way the library scores a password.
func scorePassword(analysis passwordAnalysis, options *jsval.Object) float64 {
	points := float64(0)
	points += analysis.uniqueChars * jsval.ToNumber(options.Get("pointsPerUnique"))
	points += (analysis.length - analysis.uniqueChars) * jsval.ToNumber(options.Get("pointsPerRepeat"))
	if analysis.lowercaseCount > 0 {
		points += jsval.ToNumber(options.Get("pointsForContainingLower"))
	}
	if analysis.uppercaseCount > 0 {
		points += jsval.ToNumber(options.Get("pointsForContainingUpper"))
	}
	if analysis.numberCount > 0 {
		points += jsval.ToNumber(options.Get("pointsForContainingNumber"))
	}
	if analysis.symbolCount > 0 {
		points += jsval.ToNumber(options.Get("pointsForContainingSymbol"))
	}
	return points
}

// IsStrongPassword reports whether the password has enough characters of each kind, or returns its score when
// the options ask for the score instead.
func IsStrongPassword(input, options jsval.Value) jsval.Value {
	s := str(input)
	analysis := analyzePassword(s)
	if !jsval.Truthy(options) {
		options = jsval.NewObject()
	}
	opts := jsval.Merge(options, strongPasswordDefaults)
	if jsval.Truthy(opts.Get("returnScore")) {
		return scorePassword(analysis, opts)
	}
	return jsval.Compare(analysis.length, opts.Get("minLength"), ">=") &&
		jsval.Compare(analysis.lowercaseCount, opts.Get("minLowercase"), ">=") &&
		jsval.Compare(analysis.uppercaseCount, opts.Get("minUppercase"), ">=") &&
		jsval.Compare(analysis.numberCount, opts.Get("minNumbers"), ">=") &&
		jsval.Compare(analysis.symbolCount, opts.Get("minSymbols"), ">=")
}

var currencyDefaults = jsval.ObjectOf(
	jsval.Of("symbol"), jsval.Of("$"),
	jsval.Of("require_symbol"), false,
	jsval.Of("allow_space_after_symbol"), false,
	jsval.Of("symbol_after_digits"), false,
	jsval.Of("allow_negatives"), true,
	jsval.Of("parens_for_negatives"), false,
	jsval.Of("negative_sign_before_digits"), false,
	jsval.Of("negative_sign_after_digits"), false,
	jsval.Of("allow_negative_sign_placeholder"), false,
	jsval.Of("thousands_separator"), jsval.Of(","),
	jsval.Of("decimal_separator"), jsval.Of("."),
	jsval.Of("allow_decimal"), true,
	jsval.Of("require_decimal"), false,
	jsval.Of("digits_after_decimal"), []jsval.Value{float64(2)},
	jsval.Of("allow_space_after_digits"), false,
)

var reNonWord = re("\\W", "")

// currencyPattern builds the pattern for an amount of money out of the options, the way the library builds it.
func currencyPattern(options *jsval.Object) *jsre.Regexp {
	given := options.Get("digits_after_decimal")
	if jsval.IsNullOrUndefined(given) {
		jsval.Throw("TypeError", "Cannot read properties of "+jsval.ToString(given).Go()+" (reading '0')")
	}
	digits, isList := jsval.Array(given)
	first := "undefined"
	if len(digits) > 0 {
		first = jsval.ToString(digits[0]).Go()
	} else if text, ok := given.(jsstr.String); ok && len(text) > 0 {
		first = text.CharAt(0).Go()
	}
	if !isList {
		jsval.Throw("TypeError", "options.digits_after_decimal.forEach is not a function")
	}
	decimalDigits := "\\d{" + first + "}"
	for i, digit := range digits {
		if i != 0 {
			decimalDigits += "|\\d{" + jsval.ToString(digit).Go() + "}"
		}
	}
	symbolText := jsre.ReplaceFunc(jsval.ToString(options.Get("symbol")), reNonWord,
		func(m *jsre.Match) jsstr.String { return jsstr.Concat(jsstr.Of("\\"), m.Text()) })
	optional := "?"
	if jsval.Truthy(options.Get("require_symbol")) {
		optional = ""
	}
	symbol := "(" + symbolText.Go() + ")" + optional
	negative := "-?"
	wholeWithoutSeparator := "[1-9]\\d*"
	wholeWithSeparator := "[1-9]\\d{0,2}(\\" + jsval.ToString(options.Get("thousands_separator")).Go() + "\\d{3})*"
	whole := "(0|" + wholeWithoutSeparator + "|" + wholeWithSeparator + ")?"
	requireDecimal := "?"
	if jsval.Truthy(options.Get("require_decimal")) {
		requireDecimal = ""
	}
	decimalAmount := "(\\" + jsval.ToString(options.Get("decimal_separator")).Go() + "(" + decimalDigits + "))" +
		requireDecimal
	pattern := whole
	if jsval.Truthy(options.Get("allow_decimal")) || jsval.Truthy(options.Get("require_decimal")) {
		pattern += decimalAmount
	}
	if jsval.Truthy(options.Get("allow_negatives")) && !jsval.Truthy(options.Get("parens_for_negatives")) {
		if jsval.Truthy(options.Get("negative_sign_after_digits")) {
			pattern += negative
		} else if jsval.Truthy(options.Get("negative_sign_before_digits")) {
			pattern = negative + pattern
		}
	}
	switch {
	case jsval.Truthy(options.Get("allow_negative_sign_placeholder")):
		pattern = "( (?!\\-))?" + pattern
	case jsval.Truthy(options.Get("allow_space_after_symbol")):
		pattern = " ?" + pattern
	case jsval.Truthy(options.Get("allow_space_after_digits")):
		pattern += "( (?!$))?"
	}
	if jsval.Truthy(options.Get("symbol_after_digits")) {
		pattern += symbol
	} else {
		pattern = symbol + pattern
	}
	if jsval.Truthy(options.Get("allow_negatives")) {
		if jsval.Truthy(options.Get("parens_for_negatives")) {
			pattern = "(\\(" + pattern + "\\)|" + pattern + ")"
		} else if !(jsval.Truthy(options.Get("negative_sign_before_digits")) ||
			jsval.Truthy(options.Get("negative_sign_after_digits"))) {
			pattern = negative + pattern
		}
	}
	return newRegExp(jsval.Of("^(?!-? )(?=.*\\d)"+pattern+"$"), jsval.Undefined)
}

// IsCurrency reports whether the string is an amount of money written the way the options describe.
func IsCurrency(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, currencyDefaults)
	return currencyPattern(opts).Test(s)
}

var normalizeEmailDefaults = jsval.ObjectOf(
	jsval.Of("all_lowercase"), true,
	jsval.Of("gmail_lowercase"), true,
	jsval.Of("gmail_remove_dots"), true,
	jsval.Of("gmail_remove_subaddress"), true,
	jsval.Of("gmail_convert_googlemaildotcom"), true,
	jsval.Of("outlookdotcom_lowercase"), true,
	jsval.Of("outlookdotcom_remove_subaddress"), true,
	jsval.Of("yahoo_lowercase"), true,
	jsval.Of("yahoo_remove_subaddress"), true,
	jsval.Of("yandex_lowercase"), true,
	jsval.Of("yandex_convert_yandexru"), true,
	jsval.Of("icloud_lowercase"), true,
	jsval.Of("icloud_remove_subaddress"), true,
)

var icloudDomains = []string{"icloud.com", "me.com"}

var outlookDomains = []string{
	"hotmail.at", "hotmail.be", "hotmail.ca", "hotmail.cl", "hotmail.co.il", "hotmail.co.nz", "hotmail.co.th",
	"hotmail.co.uk", "hotmail.com", "hotmail.com.ar", "hotmail.com.au", "hotmail.com.br", "hotmail.com.gr",
	"hotmail.com.mx", "hotmail.com.pe", "hotmail.com.tr", "hotmail.com.vn", "hotmail.cz", "hotmail.de",
	"hotmail.dk", "hotmail.es", "hotmail.fr", "hotmail.hu", "hotmail.id", "hotmail.ie", "hotmail.in",
	"hotmail.it", "hotmail.jp", "hotmail.kr", "hotmail.lv", "hotmail.my", "hotmail.ph", "hotmail.pt",
	"hotmail.sa", "hotmail.sg", "hotmail.sk", "live.be", "live.co.uk", "live.com", "live.com.ar", "live.com.mx",
	"live.de", "live.es", "live.eu", "live.fr", "live.it", "live.nl", "msn.com", "outlook.at", "outlook.be",
	"outlook.cl", "outlook.co.il", "outlook.co.nz", "outlook.co.th", "outlook.com", "outlook.com.ar",
	"outlook.com.au", "outlook.com.br", "outlook.com.gr", "outlook.com.pe", "outlook.com.tr", "outlook.com.vn",
	"outlook.cz", "outlook.de", "outlook.dk", "outlook.es", "outlook.fr", "outlook.hu", "outlook.id",
	"outlook.ie", "outlook.in", "outlook.it", "outlook.jp", "outlook.kr", "outlook.lv", "outlook.my",
	"outlook.ph", "outlook.pt", "outlook.sa", "outlook.sg", "outlook.sk", "passport.com",
}

var yahooDomains = []string{
	"rocketmail.com", "yahoo.ca", "yahoo.co.uk", "yahoo.com", "yahoo.de", "yahoo.fr", "yahoo.in", "yahoo.it",
	"ymail.com",
}

var yandexDomains = []string{"yandex.ru", "yandex.ua", "yandex.kz", "yandex.com", "yandex.by", "ya.ru"}

var reDotRun = re("\\.+", "g")

// NormalizeEmail writes an email address in the form its provider treats as the same address, following the
// rules of Gmail, iCloud, Outlook, Yahoo and Yandex. It gives false for an address it cannot normalize.
func NormalizeEmail(input, options jsval.Value) jsval.Value {
	if !jsval.IsString(input) {
		if jsval.IsNullOrUndefined(input) {
			jsval.Throw("TypeError", "Cannot read properties of "+jsval.ToString(input).Go()+" (reading 'split')")
		}
		jsval.Throw("TypeError", "email.split is not a function")
	}
	email := jsval.Str(input)
	opts := jsval.Merge(options, normalizeEmailDefaults)
	rawParts := email.Split("@")
	domain := rawParts[len(rawParts)-1]
	user := jsstr.Join(rawParts[:len(rawParts)-1], "@")
	domain = domain.ToLower()
	switch {
	case domain.Is("gmail.com") || domain.Is("googlemail.com"):
		if jsval.Truthy(opts.Get("gmail_remove_subaddress")) {
			user = user.Split("+")[0]
		}
		if jsval.Truthy(opts.Get("gmail_remove_dots")) {
			user = jsre.ReplaceFunc(user, reDotRun, func(m *jsre.Match) jsstr.String {
				if len(m.Text()) > 1 {
					return m.Text()
				}
				return jsstr.String{}
			})
		}
		if len(user) == 0 {
			return false
		}
		if jsval.Truthy(opts.Get("all_lowercase")) || jsval.Truthy(opts.Get("gmail_lowercase")) {
			user = user.ToLower()
		}
		if jsval.Truthy(opts.Get("gmail_convert_googlemaildotcom")) {
			domain = jsstr.Of("gmail.com")
		}
	case includesText(icloudDomains, domain):
		if jsval.Truthy(opts.Get("icloud_remove_subaddress")) {
			user = user.Split("+")[0]
		}
		if len(user) == 0 {
			return false
		}
		if jsval.Truthy(opts.Get("all_lowercase")) || jsval.Truthy(opts.Get("icloud_lowercase")) {
			user = user.ToLower()
		}
	case includesText(outlookDomains, domain):
		if jsval.Truthy(opts.Get("outlookdotcom_remove_subaddress")) {
			user = user.Split("+")[0]
		}
		if len(user) == 0 {
			return false
		}
		if jsval.Truthy(opts.Get("all_lowercase")) || jsval.Truthy(opts.Get("outlookdotcom_lowercase")) {
			user = user.ToLower()
		}
	case includesText(yahooDomains, domain):
		if jsval.Truthy(opts.Get("yahoo_remove_subaddress")) {
			components := user.Split("-")
			if len(components) > 1 {
				user = jsstr.Join(components[:len(components)-1], "-")
			} else {
				user = components[0]
			}
		}
		if len(user) == 0 {
			return false
		}
		if jsval.Truthy(opts.Get("all_lowercase")) || jsval.Truthy(opts.Get("yahoo_lowercase")) {
			user = user.ToLower()
		}
	case includesText(yandexDomains, domain):
		if jsval.Truthy(opts.Get("all_lowercase")) || jsval.Truthy(opts.Get("yandex_lowercase")) {
			user = user.ToLower()
		}
		if jsval.Truthy(opts.Get("yandex_convert_yandexru")) {
			domain = jsstr.Of("yandex.ru")
		}
	case jsval.Truthy(opts.Get("all_lowercase")):
		user = user.ToLower()
	}
	return jsstr.Concat(user, jsstr.Of("@"), domain)
}
