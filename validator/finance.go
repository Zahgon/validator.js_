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

// iso7064Check is the ISO 7064 check, modulus eleven over ten, over a string of digits whose last digit is the
// check digit.
func iso7064Check(s jsstr.String) bool {
	checkvalue := float64(10)
	for i := 0; i < len(s)-1; i++ {
		digit := jsstr.ParseInt(s.CharAt(i), 10)
		if math.Mod(digit+checkvalue, 10) == 0 {
			checkvalue = math.Mod(10*2, 11)
		} else {
			checkvalue = math.Mod(math.Mod(digit+checkvalue, 10)*2, 11)
		}
	}
	if checkvalue == 1 {
		checkvalue = 0
	} else {
		checkvalue = 11 - checkvalue
	}
	return checkvalue == jsstr.ParseInt(s.CharAt(10), 10)
}

// luhnCheck is the Luhn check, modulus ten, over a string of digits whose last digit is the check digit.
func luhnCheck(s jsstr.String) bool {
	checksum := float64(0)
	second := false
	for i := len(s) - 1; i >= 0; i-- {
		digit := jsstr.ParseInt(s.CharAt(i), 10)
		if second {
			product := digit * 2
			if product > 9 {
				for _, c := range jsstr.NumberToString(product) {
					checksum += jsstr.ParseInt(jsstr.Of(string(c)), 10)
				}
			} else {
				checksum += product
			}
		} else {
			checksum += digit
		}
		second = !second
	}
	return math.Mod(checksum, 10) == 0
}

// reverseMultiplyAndSum multiplies the digits by a falling multiplier and adds the products, which is how several
// modulus eleven checks are written.
func reverseMultiplyAndSum(digits []float64, base float64) float64 {
	total := float64(0)
	for i, digit := range digits {
		total += digit * (base - float64(i))
	}
	return total
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
	{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
	{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
	{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

// verhoeffCheck is the Verhoeff check over a string of digits.
func verhoeffCheck(s jsstr.String) bool {
	reversed := s.Reverse()
	checksum := 0
	for i := 0; i < len(reversed); i++ {
		digit := int(jsstr.ParseInt(reversed.CharAt(i), 10))
		if digit < 0 || digit > 9 {
			return false
		}
		checksum = verhoeffD[checksum][verhoeffP[i%8][digit]]
	}
	return checksum == 0
}

var reSeparators = re("[- ]+", "g")

// IsLuhnNumber reports whether the digits pass the Luhn check, ignoring spaces and hyphens.
func IsLuhnNumber(input jsval.Value) jsval.Value {
	s := str(input)
	sanitized := jsre.Replace(s, reSeparators, jsstr.String{})
	sum := float64(0)
	shouldDouble := false
	for i := len(sanitized) - 1; i >= 0; i-- {
		value := jsstr.ParseInt(sanitized.Substring(i, i+1), 10)
		if shouldDouble {
			value *= 2
			if value >= 10 {
				sum += math.Mod(value, 10) + 1
			} else {
				sum += value
			}
		} else {
			sum += value
		}
		shouldDouble = !shouldDouble
	}
	return math.Mod(sum, 10) == 0 && len(sanitized) > 0
}

var cardOrder = []string{"amex", "dinersclub", "discover", "jcb", "mastercard", "unionpay", "visa"}

var cardPatterns = map[string]*jsre.Regexp{
	"amex":       re("^3[47][0-9]{13}$", ""),
	"dinersclub": re("^3(?:0[0-5]|[68][0-9])[0-9]{11}$", ""),
	"discover":   re("^6(?:011|5[0-9][0-9])[0-9]{12,15}$", ""),
	"jcb":        re("^(?:2131|1800|35\\d{3})\\d{11}$", ""),
	"mastercard": re("^5[1-5][0-9]{2}|(222[1-9]|22[3-9][0-9]|2[3-6][0-9]{2}|27[01][0-9]|2720)[0-9]{12}$", ""),
	"unionpay":   re("^(6[27][0-9]{14}|^(81[0-9]{14,17}))$", ""),
	"visa":       re("^(?:4[0-9]{12})(?:[0-9]{3,6})?$", ""),
}

// IsCreditCard reports whether the string is a payment card number, of the provider the options name.
func IsCreditCard(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	if jsval.IsNull(options) {
		jsval.Throw("TypeError", "Cannot destructure property 'provider' of 'options' as it is null.")
	}
	provider := prop(options, "provider")
	sanitized := jsre.Replace(s, reSeparators, jsstr.String{})
	if jsval.Truthy(provider) {
		name := jsval.ToString(provider).ToLower().Go()
		pattern, known := cardPatterns[name]
		if known {
			if !pattern.Test(sanitized) {
				return false
			}
		} else {
			jsval.Throw("Error", jsval.ToString(provider).Go()+" is not a valid credit card provider.")
		}
	} else {
		matched := false
		for _, name := range cardOrder {
			if cardPatterns[name].Test(sanitized) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return IsLuhnNumber(input)
}

var reRouting = re("^(?!(1[3-9])|(20)|(3[3-9])|(4[0-9])|(5[0-9])|(60)|(7[3-9])|(8[1-9])|(9[0-2])|(9[3-9]))"+
	"[0-9]{9}$", "")

// IsAbaRouting reports whether the string is an American bank routing number.
func IsAbaRouting(input jsval.Value) jsval.Value {
	s := str(input)
	if !reRouting.Test(s) {
		return false
	}
	checkSum := float64(0)
	for i := 0; i < len(s); i++ {
		digit := jsstr.ToNumber(s.CharAt(i))
		switch i % 3 {
		case 0:
			checkSum += digit * 3
		case 1:
			checkSum += digit * 7
		default:
			checkSum += digit
		}
	}
	return math.Mod(checkSum, 10) == 0
}

var reBIC = re("^[A-Za-z]{6}[A-Za-z0-9]{2}([A-Za-z0-9]{3})?$", "")

// IsBIC reports whether the string is a bank identifier code.
func IsBIC(input jsval.Value) jsval.Value {
	s := str(input)
	countryCode := s.Slice(4, 6).ToUpper()
	if !includesText(iso31661Alpha2Codes, countryCode) && !countryCode.Is("XK") {
		return false
	}
	return reBIC.Test(s)
}

var ibanTable = newPatternTable(ibanEntries)

var (
	reIBANStrip    = re("[\\s\\-]+", "gi")
	reIBANNonAlnum = re("[^A-Z0-9]+", "gi")
	reUpperLetters = re("[A-Z]", "g")
	reDigitGroups  = re("\\d{1,7}", "g")
)

// hasValidIbanFormat reports whether the country of the account number allows it and its pattern accepts it.
func hasValidIbanFormat(s jsstr.String, options jsval.Value) bool {
	stripped := jsre.Replace(s, reIBANStrip, jsstr.String{}).ToUpper()
	country := stripped.Slice(0, 2).ToUpper().Go()
	known := ibanTable.Has(country)
	if whitelist := prop(options, "whitelist"); jsval.Truthy(whitelist) {
		list, ok := jsval.Array(whitelist)
		if !ok {
			jsval.Throw("TypeError", "countryCodeArray.filter is not a function")
		}
		for _, code := range list {
			if !ibanTable.Has(jsval.ToString(code).Go()) {
				return false
			}
		}
		found := false
		for _, code := range list {
			if jsval.StrictEquals(code, jsstr.Of(country)) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if blacklist := prop(options, "blacklist"); jsval.Truthy(blacklist) {
		list, ok := jsval.Array(blacklist)
		if !ok {
			return false
		}
		for _, code := range list {
			if jsval.StrictEquals(code, jsstr.Of(country)) {
				return false
			}
		}
	}
	return known && ibanTable.patterns[country].Test(stripped)
}

// hasValidIbanChecksum moves the first four characters to the end, turns the letters into numbers and checks that
// the result leaves one over when divided by ninety seven.
func hasValidIbanChecksum(s jsstr.String) bool {
	stripped := jsre.Replace(s, reIBANNonAlnum, jsstr.String{}).ToUpper()
	rearranged := jsstr.Concat(stripped.From(4), stripped.Slice(0, 4))
	digits := jsre.ReplaceFunc(rearranged, reUpperLetters, func(m *jsre.Match) jsstr.String {
		return jsstr.Of(jsstr.NumberToString(float64(m.Text()[0]) - 55))
	})
	groups, _ := jsre.Matches(digits, reDigitGroups)
	if groups == nil {
		jsval.Throw("TypeError", "Cannot read properties of null (reading 'reduce')")
	}
	accumulator := jsstr.String{}
	remainder := float64(0)
	for i, group := range groups {
		if i == 0 {
			remainder = math.Mod(jsstr.ToNumber(jsstr.Concat(accumulator, group)), 97)
		} else {
			remainder = math.Mod(jsstr.ToNumber(jsstr.Concat(jsstr.Of(jsstr.NumberToString(remainder)), group)), 97)
		}
	}
	return remainder == 1
}

// IsIBAN reports whether the string is an international bank account number, of a country the options allow.
func IsIBAN(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	return hasValidIbanFormat(s, options) && hasValidIbanChecksum(s)
}

var reValidEAN = re("^(\\d{8}|\\d{13}|\\d{14})$", "")

// eanWeight is the weight of the digit at the position, which depends on how long the number is.
func eanWeight(length, index int) float64 {
	if length == 8 || length == 14 {
		if index%2 == 0 {
			return 3
		}
		return 1
	}
	if index%2 == 0 {
		return 1
	}
	return 3
}

// IsEAN reports whether the string is an international article number with the right check digit.
func IsEAN(input jsval.Value) jsval.Value {
	s := str(input)
	actual := jsstr.ToNumber(s.From(len(s) - 1))
	checksum := float64(0)
	body := s.Slice(0, -1)
	for i := 0; i < len(body); i++ {
		checksum += jsstr.ToNumber(body.CharAt(i)) * eanWeight(len(s), i)
	}
	remainder := 10 - math.Mod(checksum, 10)
	if remainder >= 10 {
		remainder = 0
	}
	return reValidEAN.Test(s) && actual == remainder
}

var reISIN = re("^[A-Z]{2}[0-9A-Z]{9}[0-9]$", "")

// IsISIN reports whether the string is an international securities identification number.
func IsISIN(input jsval.Value) jsval.Value {
	s := str(input)
	if !reISIN.Test(s) {
		return false
	}
	double := true
	sum := 0
	add := func(digit int) {
		if double {
			if digit >= 5 {
				sum += 1 + (digit-5)*2
			} else {
				sum += digit * 2
			}
		} else {
			sum += digit
		}
		double = !double
	}
	for i := len(s) - 2; i >= 0; i-- {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			value := int(c) - 55
			add(value % 10)
			add(value / 10)
			continue
		}
		add(int(c) - '0')
	}
	check := (sum+9)/10*10 - sum
	return int(jsstr.ToNumber(s.CharAt(len(s)-1))) == check
}

var (
	rePossibleISBN10 = re("^(?:[0-9]{9}X|[0-9]{10})$", "")
	rePossibleISBN13 = re("^(?:[0-9]{13})$", "")
	reISBNStrip      = re("[\\s-]+", "g")
)

// IsISBN reports whether the string is an international standard book number of the version the options name,
// of either version when none is named.
func IsISBN(input, options jsval.Value) jsval.Value {
	s := str(input)
	chosen := or(optionalProp(options, "version"), options)
	version := jsval.ToString(chosen).Go()
	if !jsval.Truthy(chosen) {
		return jsval.Truthy(IsISBN(input, jsval.ObjectOf(jsval.Of("version"), float64(10)))) ||
			jsval.Truthy(IsISBN(input, jsval.ObjectOf(jsval.Of("version"), float64(13))))
	}
	sanitized := jsre.Replace(s, reISBNStrip, jsstr.String{})
	checksum := float64(0)
	switch version {
	case "10":
		if !rePossibleISBN10.Test(sanitized) {
			return false
		}
		for i := 0; i < 9; i++ {
			checksum += float64(i+1) * jsstr.ToNumber(sanitized.CharAt(i))
		}
		if sanitized.CharAt(9).Is("X") {
			checksum += 100
		} else {
			checksum += 10 * jsstr.ToNumber(sanitized.CharAt(9))
		}
		if math.Mod(checksum, 11) == 0 {
			return true
		}
	case "13":
		if !rePossibleISBN13.Test(sanitized) {
			return false
		}
		for i := 0; i < 12; i++ {
			factor := float64(1)
			if i%2 == 1 {
				factor = 3
			}
			checksum += factor * jsstr.ToNumber(sanitized.CharAt(i))
		}
		if jsstr.ToNumber(sanitized.CharAt(12))-math.Mod(10-math.Mod(checksum, 10), 10) == 0 {
			return true
		}
	}
	return false
}

const issnSource = "^\\d{4}-?\\d{3}[\\dX]$"

var reHyphen = re("-", "")

// IsISSN reports whether the string is an international standard serial number.
func IsISSN(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	source := issnSource
	if jsval.Truthy(prop(options, "require_hyphen")) {
		source = replaceOnce(source, "?", "")
	}
	flags := "i"
	if jsval.Truthy(prop(options, "case_sensitive")) {
		flags = ""
	}
	if !newRegExp(jsval.Of(source), jsval.Of(flags)).Test(s) {
		return false
	}
	digits := s.ReplaceText(jsstr.Of("-"), jsstr.String{}).ToUpper()
	checksum := float64(0)
	for i := 0; i < len(digits); i++ {
		digit := digits.CharAt(i)
		value := jsstr.ToNumber(digit)
		if digit.Is("X") {
			value = 10
		}
		checksum += value * float64(8-i)
	}
	return math.Mod(checksum, 11) == 0
}

var (
	reIMEIPlain  = re("^[0-9]{15}$", "")
	reIMEIHyphen = re("^\\d{2}-\\d{6}-\\d{6}-\\d{1}$", "")
	reHyphens    = re("-", "g")
)

// IsIMEI reports whether the string is a mobile equipment identity, written with hyphens when the options allow
// them.
func IsIMEI(input, options jsval.Value) jsval.Value {
	s := str(input)
	if !jsval.Truthy(options) {
		options = jsval.NewObject()
	}
	pattern := reIMEIPlain
	if jsval.Truthy(prop(options, "allow_hyphens")) {
		pattern = reIMEIHyphen
	}
	if !pattern.Test(s) {
		return false
	}
	s = jsre.Replace(s, reHyphens, jsstr.String{})
	sum := float64(0)
	mul := float64(2)
	for i := 0; i < 14; i++ {
		product := jsstr.ParseInt(s.Substring(14-i-1, 14-i), 10) * mul
		if product >= 10 {
			sum += math.Mod(product, 10) + 1
		} else {
			sum += product
		}
		if mul == 1 {
			mul++
		} else {
			mul--
		}
	}
	check := math.Mod(10-math.Mod(sum, 10), 10)
	return check == jsstr.ParseInt(s.Substring(14, 15), 10)
}

var (
	reISO6346 = re("^[A-Z]{3}(U[0-9]{7})|([J,Z][0-9]{6,7})$", "")
	reDigit   = re("^[0-9]$", "")
)

// IsISO6346 reports whether the string is a freight container identifier.
func IsISO6346(input jsval.Value) jsval.Value {
	s := str(input).ToUpper()
	if !reISO6346.Test(s) {
		return false
	}
	if len(s) == 11 {
		sum := float64(0)
		for i := 0; i < len(s)-1; i++ {
			if !reDigit.Test(s.CharAt(i)) {
				letterCode := float64(s[i]) - 55
				var converted float64
				switch {
				case letterCode < 11:
					converted = letterCode
				case letterCode <= 20:
					converted = 12 + math.Mod(letterCode, 11)
				case letterCode <= 30:
					converted = 23 + math.Mod(letterCode, 21)
				default:
					converted = 34 + math.Mod(letterCode, 31)
				}
				sum += converted * math.Pow(2, float64(i))
			} else {
				sum += jsstr.ToNumber(s.CharAt(i)) * math.Pow(2, float64(i))
			}
		}
		check := math.Mod(sum, 11)
		if check == 10 {
			check = 0
		}
		return jsstr.ToNumber(s.CharAt(len(s)-1)) == check
	}
	return true
}

// IsFreightContainerID is another name for IsISO6346, which the library also exposes.
func IsFreightContainerID(input jsval.Value) jsval.Value { return IsISO6346(input) }

// IsISO31661Alpha2 reports whether the string is a two letter country code, or one of the
// options' userAssignedCodes that is itself two letters.
func IsISO31661Alpha2(input, options jsval.Value) jsval.Value {
	s := str(input).ToUpper()
	return userAssigned(options, reAlpha2Code, s) || includesText(iso31661Alpha2Codes, s)
}

// IsISO31661Alpha3 reports whether the string is a three letter country code, or one of the
// options' userAssignedCodes that is itself three letters.
func IsISO31661Alpha3(input, options jsval.Value) jsval.Value {
	s := str(input).ToUpper()
	return userAssigned(options, reAlpha3Code, s) || includesText(iso31661Alpha3Codes, s)
}

var (
	reAlpha2Code = re("^[a-zA-Z]{2}$", "")
	reAlpha3Code = re("^[a-zA-Z]{3}$", "")
)

// userAssigned reports whether the upper-cased code is among options.userAssignedCodes, keeping only the entries that
// match the code pattern, as the original does. Omitted options mean none; null options raise the original's type error.
func userAssigned(options jsval.Value, pattern *jsre.Regexp, code jsstr.String) bool {
	if jsval.IsUndefined(options) {
		return false
	}
	codes := prop(options, "userAssignedCodes")
	if !jsval.Truthy(codes) {
		return false
	}
	items, ok := jsval.Array(codes)
	if !ok {
		jsval.Throw("TypeError", "(userAssignedCodes || []).reduce is not a function")
	}
	for _, item := range items {
		if pattern.Test(jsval.ToString(item)) && jsval.ToString(item).ToUpper().Equal(code) {
			return true
		}
	}
	return false
}

// IsISO31661Numeric reports whether the string is a numeric country code.
func IsISO31661Numeric(input jsval.Value) jsval.Value {
	return includesText(iso31661NumericCodes, str(input))
}

// IsISO4217 reports whether the string is a currency code.
func IsISO4217(input jsval.Value) jsval.Value {
	return includesText(iso4217Codes, str(input).ToUpper())
}

// IsISO6391 reports whether the string is a two letter language code.
func IsISO6391(input jsval.Value) jsval.Value { return includesText(iso6391Codes, str(input)) }

// IsISO15924 reports whether the string is a script code.
func IsISO15924(input jsval.Value) jsval.Value { return includesText(iso15924Codes, str(input)) }

var mobilePhoneTable = newPatternTable(mobilePhoneEntries)

// IsMobilePhone reports whether the string is a mobile telephone number of the locale, of any locale when none
// is given or when it is "any", or of one of the locales when several are given.
func IsMobilePhone(input, locale, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.Truthy(options) && jsval.Truthy(prop(options, "strictMode")) && !s.HasPrefix("+") {
		return false
	}
	if list, ok := jsval.Array(locale); ok {
		for _, key := range list {
			name := jsval.ToString(key).Go()
			if mobilePhoneTable.Has(name) && mobilePhoneTable.patterns[name].Test(s) {
				return true
			}
		}
		return false
	}
	name := jsval.ToString(locale).Go()
	if !jsval.IsUndefined(locale) && !jsval.IsNull(locale) && mobilePhoneTable.Has(name) {
		return mobilePhoneTable.patterns[name].Test(s)
	}
	if !jsval.Truthy(locale) || name == "any" {
		for _, key := range mobilePhoneTable.Keys() {
			if mobilePhoneTable.patterns[key].Test(s) {
				return true
			}
		}
		return false
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}

var postalCodeTable = newPatternTable(postalCodeEntries)

// IsPostalCode reports whether the string is a postal code of the locale, or of any locale when the locale is
// "any".
func IsPostalCode(input, locale jsval.Value) jsval.Value {
	s := str(input)
	name := jsval.ToString(locale).Go()
	if !jsval.IsUndefined(locale) && postalCodeTable.Has(name) {
		return postalCodeTable.patterns[name].Test(s)
	}
	if name == "any" {
		for _, key := range postalCodeTable.Keys() {
			if postalCodeTable.patterns[key].Test(s) {
				return true
			}
		}
		return false
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}

var passportTable = newPatternTable(passportEntries)

var reAnySpace = re("\\s", "g")

// IsPassportNumber reports whether the string is a passport number of the country. The library looks the country
// up in upper case but reads the pattern under the code as it was given, so a code that is not already in upper
// case raises an error.
func IsPassportNumber(input, countryCode jsval.Value) jsval.Value {
	s := str(input)
	normalized := jsre.Replace(s, reAnySpace, jsstr.String{}).ToUpper()
	if jsval.IsNullOrUndefined(countryCode) {
		jsval.Throw("TypeError", "Cannot read properties of "+jsval.ToString(countryCode).Go()+
			" (reading 'toUpperCase')")
	}
	upper := jsval.ToString(countryCode).ToUpper().Go()
	if !passportTable.Has(upper) {
		return false
	}
	pattern, ok := passportTable.patterns[jsval.ToString(countryCode).Go()]
	if !ok {
		jsval.Throw("TypeError", "Cannot read properties of undefined (reading 'test')")
	}
	return pattern.Test(normalized)
}

// licensePlatePattern is one number plate format: the pattern and whether the library trims the string first.
type licensePlatePattern struct {
	locale  string
	source  string
	flags   string
	trimmed bool
}

var licensePlateTable = func() ([]string, map[string]licensePlatePattern, map[string]*jsre.Regexp) {
	keys := []string{}
	byKey := map[string]licensePlatePattern{}
	patterns := map[string]*jsre.Regexp{}
	for _, e := range licensePlateEntries {
		keys = append(keys, e.locale)
		byKey[e.locale] = e
		patterns[e.locale] = re(e.source, e.flags)
	}
	return keys, byKey, patterns
}

var (
	licensePlateKeys, licensePlateByKey, licensePlatePatterns = licensePlateTable()
)

func licensePlateMatches(locale string, s jsstr.String) bool {
	entry := licensePlateByKey[locale]
	if entry.trimmed {
		s = s.Trim()
	}
	return licensePlatePatterns[locale].Test(s)
}

// IsLicensePlate reports whether the string is a vehicle number plate of the locale, or of any locale when the
// locale is "any".
func IsLicensePlate(input, locale jsval.Value) jsval.Value {
	s := str(input)
	name := jsval.ToString(locale).Go()
	if _, ok := licensePlateByKey[name]; ok && !jsval.IsUndefined(locale) {
		return licensePlateMatches(name, s)
	}
	if name == "any" {
		for _, key := range licensePlateKeys {
			if licensePlateMatches(key, s) {
				return true
			}
		}
		return false
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}
