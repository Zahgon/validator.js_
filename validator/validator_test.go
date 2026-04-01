// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"math"
	"testing"

	"github.com/validatorjs/validator/jsval"
)

// call runs a function of the library and fails the test when it raises.
func call(t *testing.T, name string, args []jsval.Value) jsval.Value {
	t.Helper()
	value, err := Call(name, args)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

// The verdict of every validator for the inputs and options given, as the library gives it.
func TestValidators(t *testing.T) {
	cases := []struct {
		fn   string
		args []jsval.Value
		want bool
	}{
		{"isEmail", []jsval.Value{jsval.Of("foo@bar.com")}, true},
		{"isEmail", []jsval.Value{jsval.Of("foo@bar")}, false},
		{"isEmail", []jsval.Value{jsval.Of("foo@bar"), jsval.ObjectOf(jsval.Of("require_tld"), false)}, true},
		{"isEmail", []jsval.Value{jsval.Of("Foo <foo@bar.com>")}, false},
		{"isEmail", []jsval.Value{jsval.Of("Foo <foo@bar.com>"), jsval.ObjectOf(jsval.Of("allow_display_name"), true)}, true},
		{"isEmail", []jsval.Value{jsval.Of("foo@bar.com"), jsval.ObjectOf(jsval.Of("require_display_name"), true)}, false},
		{"isEmail", []jsval.Value{jsval.Of("ä@bar.com")}, true},
		{"isEmail", []jsval.Value{jsval.Of("ä@bar.com"), jsval.ObjectOf(jsval.Of("allow_utf8_local_part"), false)}, false},
		{"isEmail", []jsval.Value{jsval.Of("foo+bar@bar.com"), jsval.ObjectOf(jsval.Of("blacklisted_chars"), jsval.Of("+"))}, false},
		{"isEmail", []jsval.Value{jsval.Of("foo@gmail.com"), jsval.ObjectOf(jsval.Of("host_blacklist"), []jsval.Value{jsval.Of("gmail.com")})}, false},
		{"isEmail", []jsval.Value{jsval.Of("foo@sub.bar.com"), jsval.ObjectOf(jsval.Of("host_whitelist"), []jsval.Value{jsval.Of("*.bar.com")})}, false},
		{"isEmail", []jsval.Value{jsval.Of("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@bar.com")}, false},
		{"isEmail", []jsval.Value{jsval.Of("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@bar.com"), jsval.ObjectOf(jsval.Of("ignore_max_length"), true)}, true},
		{"isURL", []jsval.Value{jsval.Of("http://foo.com")}, true},
		{"isURL", []jsval.Value{jsval.Of("foo.com")}, true},
		{"isURL", []jsval.Value{jsval.Of("foo.com"), jsval.ObjectOf(jsval.Of("require_protocol"), true)}, false},
		{"isURL", []jsval.Value{jsval.Of("//foo.com"), jsval.ObjectOf(jsval.Of("allow_protocol_relative_urls"), true)}, true},
		{"isURL", []jsval.Value{jsval.Of("ftp://foo.com"), jsval.ObjectOf(jsval.Of("protocols"), []jsval.Value{jsval.Of("http")})}, false},
		{"isURL", []jsval.Value{jsval.Of("http://foo.com:65536")}, false},
		{"isURL", []jsval.Value{jsval.Of("http://foo.com/a?b#c"), jsval.ObjectOf(jsval.Of("allow_query_components"), false)}, false},
		{"isURL", []jsval.Value{jsval.Of("http://user:pass@foo.com"), jsval.ObjectOf(jsval.Of("disallow_auth"), true)}, false},
		{"isFQDN", []jsval.Value{jsval.Of("foo.com")}, true},
		{"isFQDN", []jsval.Value{jsval.Of("foo")}, false},
		{"isFQDN", []jsval.Value{jsval.Of("foo"), jsval.ObjectOf(jsval.Of("require_tld"), false)}, true},
		{"isFQDN", []jsval.Value{jsval.Of("foo.com."), jsval.ObjectOf(jsval.Of("allow_trailing_dot"), true)}, true},
		{"isFQDN", []jsval.Value{jsval.Of("foo.123")}, false},
		{"isFQDN", []jsval.Value{jsval.Of("foo.123"), jsval.ObjectOf(jsval.Of("allow_numeric_tld"), true)}, true},
		{"isFQDN", []jsval.Value{jsval.Of("*.foo.com"), jsval.ObjectOf(jsval.Of("allow_wildcard"), true)}, true},
		{"isFQDN", []jsval.Value{jsval.Of("xn--80ak6aa92e.com")}, true},
		{"isFQDN", []jsval.Value{jsval.Of("日本.jp")}, true},
		{"isIP", []jsval.Value{jsval.Of("1.2.3.4")}, true},
		{"isIP", []jsval.Value{jsval.Of("1.2.3.4"), jsval.Num(6)}, false},
		{"isIP", []jsval.Value{jsval.Of("::1")}, true},
		{"isIP", []jsval.Value{jsval.Of("::1"), jsval.Num(4)}, false},
		{"isIP", []jsval.Value{jsval.Of("01.2.3.4")}, false},
		{"isIP", []jsval.Value{jsval.Of("1.2.3.4"), jsval.Of("6")}, false},
		{"isIP", []jsval.Value{jsval.Of("::ffff:1.2.3.4"), jsval.Num(6)}, true},
		{"isIPRange", []jsval.Value{jsval.Of("1.2.3.4/24")}, true},
		{"isIPRange", []jsval.Value{jsval.Of("1.2.3.4/33")}, false},
		{"isIPRange", []jsval.Value{jsval.Of("::1/128"), jsval.Num(6)}, true},
		{"isMACAddress", []jsval.Value{jsval.Of("00:1B:44:11:3A:B7")}, true},
		{"isMACAddress", []jsval.Value{jsval.Of("001B.4411.3AB7")}, true},
		{"isMACAddress", []jsval.Value{jsval.Of("00:1B:44:11:3A:B7"), jsval.ObjectOf(jsval.Of("no_separators"), true)}, false},
		{"isMACAddress", []jsval.Value{jsval.Of("001B44113AB7"), jsval.ObjectOf(jsval.Of("no_separators"), true)}, true},
		{"isLatLong", []jsval.Value{jsval.Of("40.74,-73.98")}, true},
		{"isLatLong", []jsval.Value{jsval.Of("91,0")}, false},
		{"isLatLong", []jsval.Value{jsval.Of("40° 44' 55.0\" N, 73° 59' 11.0\" W"), jsval.ObjectOf(jsval.Of("checkDMS"), true)}, true},
		{"isInt", []jsval.Value{jsval.Of("42")}, true},
		{"isInt", []jsval.Value{jsval.Of("042")}, true},
		{"isInt", []jsval.Value{jsval.Of("+42")}, true},
		{"isInt", []jsval.Value{jsval.Of("4.2")}, false},
		{"isInt", []jsval.Value{jsval.Of("042"), jsval.ObjectOf(jsval.Of("allow_leading_zeroes"), false)}, false},
		{"isInt", []jsval.Value{jsval.Of("42"), jsval.ObjectOf(jsval.Of("min"), jsval.Num(50))}, false},
		{"isInt", []jsval.Value{jsval.Of("42"), jsval.ObjectOf(jsval.Of("gt"), jsval.Num(42))}, false},
		{"isInt", []jsval.Value{jsval.Of("42"), jsval.ObjectOf(jsval.Of("lt"), jsval.Num(42))}, false},
		{"isFloat", []jsval.Value{jsval.Of("4.2")}, true},
		{"isFloat", []jsval.Value{jsval.Of("4,2")}, false},
		{"isFloat", []jsval.Value{jsval.Of("4,2"), jsval.ObjectOf(jsval.Of("locale"), jsval.Of("de-DE"))}, true},
		{"isFloat", []jsval.Value{jsval.Of("4.2"), jsval.ObjectOf(jsval.Of("min"), jsval.Num(5))}, false},
		{"isFloat", []jsval.Value{jsval.Of("")}, false},
		{"isFloat", []jsval.Value{jsval.Of(".")}, false},
		{"isNumeric", []jsval.Value{jsval.Of("123")}, true},
		{"isNumeric", []jsval.Value{jsval.Of("+123")}, true},
		{"isNumeric", []jsval.Value{jsval.Of("+123"), jsval.ObjectOf(jsval.Of("no_symbols"), true)}, false},
		{"isNumeric", []jsval.Value{jsval.Of("1.2")}, true},
		{"isNumeric", []jsval.Value{jsval.Of("1,2"), jsval.ObjectOf(jsval.Of("locale"), jsval.Of("de-DE"))}, true},
		{"isDecimal", []jsval.Value{jsval.Of("1.2")}, true},
		{"isDecimal", []jsval.Value{jsval.Of("1.")}, false},
		{"isDecimal", []jsval.Value{jsval.Of("1.2"), jsval.ObjectOf(jsval.Of("force_decimal"), true)}, true},
		{"isDecimal", []jsval.Value{jsval.Of("1"), jsval.ObjectOf(jsval.Of("force_decimal"), true)}, false},
		{"isDecimal", []jsval.Value{jsval.Of("1.234"), jsval.ObjectOf(jsval.Of("decimal_digits"), jsval.Of("1,2"))}, false},
		{"isDivisibleBy", []jsval.Value{jsval.Of("10"), jsval.Num(2)}, true},
		{"isDivisibleBy", []jsval.Value{jsval.Of("10"), jsval.Num(3)}, false},
		{"isDivisibleBy", []jsval.Value{jsval.Of("10"), jsval.Num(0)}, false},
		{"isPort", []jsval.Value{jsval.Of("0")}, true},
		{"isPort", []jsval.Value{jsval.Of("65535")}, true},
		{"isPort", []jsval.Value{jsval.Of("65536")}, false},
		{"isAlpha", []jsval.Value{jsval.Of("abc")}, true},
		{"isAlpha", []jsval.Value{jsval.Of("abc1")}, false},
		{"isAlpha", []jsval.Value{jsval.Of("ÄÖÜ"), jsval.Of("de-DE")}, true},
		{"isAlpha", []jsval.Value{jsval.Of("ÄÖÜ"), jsval.Of("en-US")}, false},
		{"isAlpha", []jsval.Value{jsval.Of("abc-def"), jsval.Of("en-US"), jsval.ObjectOf(jsval.Of("ignore"), jsval.Of("-"))}, true},
		{"isAlpha", []jsval.Value{jsval.Of("Αθήνα"), jsval.Of("el-GR")}, true},
		{"isAlpha", []jsval.Value{jsval.Of("日本"), jsval.Of("ja-JP")}, true},
		{"isAlphanumeric", []jsval.Value{jsval.Of("abc123")}, true},
		{"isAlphanumeric", []jsval.Value{jsval.Of("abc 123")}, false},
		{"isAlphanumeric", []jsval.Value{jsval.Of("١٢٣"), jsval.Of("ar")}, true},
		{"isLowercase", []jsval.Value{jsval.Of("abc")}, true},
		{"isLowercase", []jsval.Value{jsval.Of("aBc")}, false},
		{"isLowercase", []jsval.Value{jsval.Of("ß")}, true},
		{"isLowercase", []jsval.Value{jsval.Of("İ")}, false},
		{"isUppercase", []jsval.Value{jsval.Of("ABC")}, true},
		{"isUppercase", []jsval.Value{jsval.Of("ß")}, false},
		{"isUppercase", []jsval.Value{jsval.Of("İ")}, true},
		{"isAscii", []jsval.Value{jsval.Of("abc")}, true},
		{"isAscii", []jsval.Value{jsval.Of("é")}, false},
		{"isFullWidth", []jsval.Value{jsval.Of("ＡＢ")}, true},
		{"isFullWidth", []jsval.Value{jsval.Of("AB")}, false},
		{"isHalfWidth", []jsval.Value{jsval.Of("AB")}, true},
		{"isVariableWidth", []jsval.Value{jsval.Of("ＡB")}, true},
		{"isMultibyte", []jsval.Value{jsval.Of("é")}, true},
		{"isSurrogatePair", []jsval.Value{jsval.Of("😀")}, true},
		{"isSurrogatePair", []jsval.Value{jsval.Of("a")}, false},
		{"isLength", []jsval.Value{jsval.Of("abc"), jsval.ObjectOf(jsval.Of("min"), jsval.Num(1), jsval.Of("max"), jsval.Num(3))}, true},
		{"isLength", []jsval.Value{jsval.Of("abc"), jsval.ObjectOf(jsval.Of("max"), jsval.Num(2))}, false},
		{"isLength", []jsval.Value{jsval.Of("😀"), jsval.ObjectOf(jsval.Of("min"), jsval.Num(1), jsval.Of("max"), jsval.Num(1))}, true},
		{"isLength", []jsval.Value{jsval.Of("abc"), jsval.ObjectOf(jsval.Of("discreteLengths"), []jsval.Value{jsval.Num(2), jsval.Num(3)})}, true},
		{"isByteLength", []jsval.Value{jsval.Of("é"), jsval.ObjectOf(jsval.Of("min"), jsval.Num(2), jsval.Of("max"), jsval.Num(2))}, true},
		{"isEmpty", []jsval.Value{jsval.Of("")}, true},
		{"isEmpty", []jsval.Value{jsval.Of(" ")}, false},
		{"isEmpty", []jsval.Value{jsval.Of(" "), jsval.ObjectOf(jsval.Of("ignore_whitespace"), true)}, true},
		{"equals", []jsval.Value{jsval.Of("a"), jsval.Of("a")}, true},
		{"equals", []jsval.Value{jsval.Of("a"), jsval.Of("A")}, false},
		{"contains", []jsval.Value{jsval.Of("abc"), jsval.Of("b")}, true},
		{"contains", []jsval.Value{jsval.Of("abc"), jsval.Of("B"), jsval.ObjectOf(jsval.Of("ignoreCase"), true)}, true},
		{"contains", []jsval.Value{jsval.Of("abab"), jsval.Of("a"), jsval.ObjectOf(jsval.Of("minOccurrences"), jsval.Num(2))}, true},
		{"contains", []jsval.Value{jsval.Of("abab"), jsval.Of("a"), jsval.ObjectOf(jsval.Of("minOccurrences"), jsval.Num(3))}, false},
		{"isIn", []jsval.Value{jsval.Of("a"), []jsval.Value{jsval.Of("a"), jsval.Of("b")}}, true},
		{"isIn", []jsval.Value{jsval.Of("c"), []jsval.Value{jsval.Of("a"), jsval.Of("b")}}, false},
		{"isIn", []jsval.Value{jsval.Of("a"), jsval.Of("cab")}, true},
		{"isWhitelisted", []jsval.Value{jsval.Of("abc"), jsval.Of("abc")}, true},
		{"isWhitelisted", []jsval.Value{jsval.Of("abd"), jsval.Of("abc")}, false},
		{"isUUID", []jsval.Value{jsval.Of("a987fbc9-4bed-4078-8f07-9141ba07c9f3")}, true},
		{"isUUID", []jsval.Value{jsval.Of("a987fbc9-4bed-4078-8f07-9141ba07c9f3"), jsval.Num(3)}, false},
		{"isUUID", []jsval.Value{jsval.Of("00000000-0000-0000-0000-000000000000"), jsval.Of("nil")}, true},
		{"isUUID", []jsval.Value{jsval.Of("a987fbc9-4bed-4078-8f07-9141ba07c9f3"), jsval.Of("loose")}, true},
		{"isULID", []jsval.Value{jsval.Of("01ARZ3NDEKTSV4RRFFQ69G5FAV")}, true},
		{"isULID", []jsval.Value{jsval.Of("01arz3ndektsv4rrffq69g5fav")}, true},
		{"isMongoId", []jsval.Value{jsval.Of("507f1f77bcf86cd799439011")}, true},
		{"isMD5", []jsval.Value{jsval.Of("d94f3f016ae679c3008de268209132f2")}, true},
		{"isHash", []jsval.Value{jsval.Of("d94f3f016ae679c3008de268209132f2"), jsval.Of("md5")}, true},
		{"isHash", []jsval.Value{jsval.Of("d94f3f016ae679c3008de268209132f2"), jsval.Of("sha1")}, false},
		{"isJWT", []jsval.Value{jsval.Of("eyJhbGciOiJIUzI1NiJ9.eyJhIjoxfQ.x")}, true},
		{"isJWT", []jsval.Value{jsval.Of("a.b")}, false},
		{"isJSON", []jsval.Value{jsval.Of("{\"a\":1}")}, true},
		{"isJSON", []jsval.Value{jsval.Of("1")}, false},
		{"isJSON", []jsval.Value{jsval.Of("1"), jsval.ObjectOf(jsval.Of("allow_primitives"), true)}, false},
		{"isBase32", []jsval.Value{jsval.Of("ZG======")}, true},
		{"isBase58", []jsval.Value{jsval.Of("bf7")}, true},
		{"isBase64", []jsval.Value{jsval.Of("YQ==")}, true},
		{"isBase64", []jsval.Value{jsval.Of("YQ"), jsval.ObjectOf(jsval.Of("urlSafe"), true)}, true},
		{"isCreditCard", []jsval.Value{jsval.Of("4111111111111111")}, true},
		{"isCreditCard", []jsval.Value{jsval.Of("4111111111111112")}, false},
		{"isCreditCard", []jsval.Value{jsval.Of("4111111111111111"), jsval.ObjectOf(jsval.Of("provider"), jsval.Of("visa"))}, true},
		{"isCreditCard", []jsval.Value{jsval.Of("4111111111111111"), jsval.ObjectOf(jsval.Of("provider"), jsval.Of("amex"))}, false},
		{"isLuhnNumber", []jsval.Value{jsval.Of("4111111111111111")}, true},
		{"isIBAN", []jsval.Value{jsval.Of("GB82WEST12345698765432")}, true},
		{"isIBAN", []jsval.Value{jsval.Of("GB82TEST12345698765432")}, false},
		{"isIBAN", []jsval.Value{jsval.Of("GB82WEST12345698765432"), jsval.ObjectOf(jsval.Of("whitelist"), []jsval.Value{jsval.Of("DE")})}, false},
		{"isBIC", []jsval.Value{jsval.Of("DEUTDEFF")}, true},
		{"isBIC", []jsval.Value{jsval.Of("DEUTDEFF500")}, true},
		{"isBIC", []jsval.Value{jsval.Of("DEUTDEF")}, false},
		{"isAbaRouting", []jsval.Value{jsval.Of("026009593")}, true},
		{"isISIN", []jsval.Value{jsval.Of("US0378331005")}, true},
		{"isISIN", []jsval.Value{jsval.Of("US0378331004")}, false},
		{"isEAN", []jsval.Value{jsval.Of("9780306406157")}, true},
		{"isISBN", []jsval.Value{jsval.Of("0306406152")}, true},
		{"isISBN", []jsval.Value{jsval.Of("0306406152"), jsval.Num(13)}, false},
		{"isISSN", []jsval.Value{jsval.Of("0378-5955")}, true},
		{"isISSN", []jsval.Value{jsval.Of("03785955"), jsval.ObjectOf(jsval.Of("require_hyphen"), true)}, false},
		{"isIMEI", []jsval.Value{jsval.Of("490154203237518")}, true},
		{"isISO6346", []jsval.Value{jsval.Of("CSCU3054383")}, false},
		{"isMobilePhone", []jsval.Value{jsval.Of("+12125551212"), jsval.Of("en-US")}, true},
		{"isMobilePhone", []jsval.Value{jsval.Of("2125551212"), jsval.Of("en-US")}, true},
		{"isMobilePhone", []jsval.Value{jsval.Of("2125551212"), jsval.Of("en-US"), jsval.ObjectOf(jsval.Of("strictMode"), true)}, false},
		{"isMobilePhone", []jsval.Value{jsval.Of("+919876543210"), jsval.Of("any")}, true},
		{"isMobilePhone", []jsval.Value{jsval.Of("123"), jsval.Of("any")}, false},
		{"isPostalCode", []jsval.Value{jsval.Of("12345"), jsval.Of("US")}, true},
		{"isPostalCode", []jsval.Value{jsval.Of("SW1A 1AA"), jsval.Of("GB")}, true},
		{"isPostalCode", []jsval.Value{jsval.Of("12345"), jsval.Of("any")}, true},
		{"isPostalCode", []jsval.Value{jsval.Of("ABC"), jsval.Of("any")}, false},
		{"isPassportNumber", []jsval.Value{jsval.Of("AB1234567"), jsval.Of("US")}, false},
		{"isLicensePlate", []jsval.Value{jsval.Of("FO 1234"), jsval.Of("de-DE")}, true},
		{"isIdentityCard", []jsval.Value{jsval.Of("84842222"), jsval.Of("ES")}, false},
		{"isIdentityCard", []jsval.Value{jsval.Of("99999999R"), jsval.Of("ES")}, true},
		{"isTaxID", []jsval.Value{jsval.Of("01-1234567"), jsval.Of("en-US")}, true},
		{"isTaxID", []jsval.Value{jsval.Of("1234567890"), jsval.Of("en-US")}, false},
		{"isVAT", []jsval.Value{jsval.Of("DE123456789"), jsval.Of("DE")}, true},
		{"isVAT", []jsval.Value{jsval.Of("GB123456789"), jsval.Of("DE")}, false},
		{"isDate", []jsval.Value{jsval.Of("2020-01-02")}, true},
		{"isDate", []jsval.Value{jsval.Of("2020-13-02")}, false},
		{"isDate", []jsval.Value{jsval.Of("02/01/2020"), jsval.Of("DD/MM/YYYY")}, true},
		{"isDate", []jsval.Value{jsval.Of("2020-01-02"), jsval.ObjectOf(jsval.Of("format"), jsval.Of("YYYY-MM-DD"), jsval.Of("strictMode"), true)}, true},
		{"isDate", []jsval.Value{jsval.Of("2020/01/02"), jsval.ObjectOf(jsval.Of("format"), jsval.Of("YYYY-MM-DD"), jsval.Of("strictMode"), true)}, false},
		{"isTime", []jsval.Value{jsval.Of("23:59")}, true},
		{"isTime", []jsval.Value{jsval.Of("24:00")}, false},
		{"isTime", []jsval.Value{jsval.Of("11:59 PM"), jsval.ObjectOf(jsval.Of("hourFormat"), jsval.Of("hour12"))}, true},
		{"isTime", []jsval.Value{jsval.Of("23:59:59"), jsval.ObjectOf(jsval.Of("mode"), jsval.Of("withSeconds"))}, true},
		{"isISO8601", []jsval.Value{jsval.Of("2020-01-02T03:04:05Z")}, true},
		{"isISO8601", []jsval.Value{jsval.Of("2020-02-30")}, true},
		{"isISO8601", []jsval.Value{jsval.Of("2020-02-30"), jsval.ObjectOf(jsval.Of("strict"), true)}, false},
		{"isISO8601", []jsval.Value{jsval.Of("2020-01-02 03:04:05Z"), jsval.ObjectOf(jsval.Of("strictSeparator"), true)}, false},
		{"isRFC3339", []jsval.Value{jsval.Of("2020-01-02T03:04:05Z")}, true},
		{"isRFC3339", []jsval.Value{jsval.Of("2020-01-02")}, false},
		{"isAfter", []jsval.Value{jsval.Of("2020-01-02"), jsval.Of("2020-01-01")}, true},
		{"isBefore", []jsval.Value{jsval.Of("2020-01-02"), jsval.Of("2020-01-01")}, false},
		{"isISO31661Alpha2", []jsval.Value{jsval.Of("US")}, true},
		{"isISO31661Alpha3", []jsval.Value{jsval.Of("USA")}, true},
		{"isISO31661Alpha2", []jsval.Value{jsval.Of("xk"), jsval.ObjectOf(jsval.Of("userAssignedCodes"), []jsval.Value{jsval.Of("XK"), jsval.Of("ABC")})}, true},
		{"isISO31661Alpha2", []jsval.Value{jsval.Of("ABC"), jsval.ObjectOf(jsval.Of("userAssignedCodes"), []jsval.Value{jsval.Of("ABC")})}, false},
		{"isISO31661Alpha3", []jsval.Value{jsval.Of("XXK"), jsval.ObjectOf(jsval.Of("userAssignedCodes"), []jsval.Value{jsval.Of("xxk")})}, true},
		{"isISO31661Numeric", []jsval.Value{jsval.Of("840")}, true},
		{"isISO4217", []jsval.Value{jsval.Of("USD")}, true},
		{"isISO6391", []jsval.Value{jsval.Of("en")}, true},
		{"isISO15924", []jsval.Value{jsval.Of("Latn")}, true},
		{"isHexColor", []jsval.Value{jsval.Of("#fff")}, true},
		{"isHexColor", []jsval.Value{jsval.Of("fff")}, true},
		{"isHexColor", []jsval.Value{jsval.Of("#ffff")}, true},
		{"isRgbColor", []jsval.Value{jsval.Of("rgb(1,2,3)")}, true},
		{"isRgbColor", []jsval.Value{jsval.Of("rgb(1,2,3)"), false}, true},
		{"isRgbColor", []jsval.Value{jsval.Of("rgba(1,2,3,0.5)")}, true},
		{"isHSL", []jsval.Value{jsval.Of("hsl(120,50%,50%)")}, true},
		{"isHexadecimal", []jsval.Value{jsval.Of("deadBEEF")}, true},
		{"isOctal", []jsval.Value{jsval.Of("0o17")}, true},
		{"isOctal", []jsval.Value{jsval.Of("17")}, true},
		{"isSemVer", []jsval.Value{jsval.Of("1.2.3")}, true},
		{"isSemVer", []jsval.Value{jsval.Of("1.2")}, false},
		{"isSlug", []jsval.Value{jsval.Of("a-b-c")}, true},
		{"isSlug", []jsval.Value{jsval.Of("a--b")}, false},
		{"isLocale", []jsval.Value{jsval.Of("en_US")}, true},
		{"isLocale", []jsval.Value{jsval.Of("en-US")}, true},
		{"isMimeType", []jsval.Value{jsval.Of("text/plain")}, true},
		{"isDataURI", []jsval.Value{jsval.Of("data:text/plain;base64,YQ==")}, true},
		{"isMagnetURI", []jsval.Value{jsval.Of("magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}, true},
		{"isMailtoURI", []jsval.Value{jsval.Of("mailto:a@b.co")}, true},
		{"isCurrency", []jsval.Value{jsval.Of("$1,000.00")}, true},
		{"isCurrency", []jsval.Value{jsval.Of("1.000,00 €"), jsval.ObjectOf(jsval.Of("symbol"), jsval.Of("€"), jsval.Of("thousands_separator"), jsval.Of("."), jsval.Of("decimal_separator"), jsval.Of(","), jsval.Of("symbol_after_digits"), true, jsval.Of("allow_space_after_digits"), true)}, true},
		{"isBtcAddress", []jsval.Value{jsval.Of("1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2")}, true},
		{"isEthereumAddress", []jsval.Value{jsval.Of("0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe")}, true},
		{"isBoolean", []jsval.Value{jsval.Of("true")}, true},
		{"isBoolean", []jsval.Value{jsval.Of("TRUE")}, false},
		{"isBoolean", []jsval.Value{jsval.Of("TRUE"), jsval.ObjectOf(jsval.Of("loose"), true)}, true},
		{"isStrongPassword", []jsval.Value{jsval.Of("Abcdef1!")}, true},
		{"isStrongPassword", []jsval.Value{jsval.Of("abc")}, false},
		{"isISRC", []jsval.Value{jsval.Of("USAT29900609")}, true},
		{"isFreightContainerID", []jsval.Value{jsval.Of("CSCU3054383")}, false},
		{"matches", []jsval.Value{jsval.Of("abc"), jsval.Of("^a")}, true},
		{"matches", []jsval.Value{jsval.Of("ABC"), jsval.Of("^a"), jsval.Of("i")}, true},
	}
	for _, c := range cases {
		if got := jsval.Truthy(call(t, c.fn, c.args)); got != c.want {
			t.Errorf("%s(%v) = %v, want %v", c.fn, c.args, got, c.want)
		}
	}
}

// The string every sanitizer gives back.
func TestSanitizers(t *testing.T) {
	cases := []struct {
		fn   string
		args []jsval.Value
		want string
	}{
		{"trim", []jsval.Value{jsval.Of("  a  ")}, "a"},
		{"trim", []jsval.Value{jsval.Of("xxaxx"), jsval.Of("x")}, "a"},
		{"ltrim", []jsval.Value{jsval.Of("  a  ")}, "a  "},
		{"rtrim", []jsval.Value{jsval.Of("  a  ")}, "  a"},
		{"trim", []jsval.Value{jsval.Of("\t\na\r\n")}, "a"},
		{"trim", []jsval.Value{jsval.Of(" a ")}, "a"},
		{"trim", []jsval.Value{jsval.Of("​a​")}, "​a​"},
		{"blacklist", []jsval.Value{jsval.Of("abc"), jsval.Of("b")}, "ac"},
		{"blacklist", []jsval.Value{jsval.Of("a.c"), jsval.Of(".")}, "ac"},
		{"whitelist", []jsval.Value{jsval.Of("abc"), jsval.Of("ac")}, "ac"},
		{"escape", []jsval.Value{jsval.Of("<a href=\"x\">&'`/</a>")}, "&lt;a href=&quot;x&quot;&gt;&amp;&#x27;&#96;&#x2F;&lt;&#x2F;a&gt;"},
		{"unescape", []jsval.Value{jsval.Of("&lt;&gt;&amp;&quot;&#x27;&#96;&#x2F;")}, "<>&\"'`/"},
		{"unescape", []jsval.Value{jsval.Of("&amp;amp;")}, "&amp;"},
		{"escape", []jsval.Value{jsval.Of("é")}, "é"},
		{"stripLow", []jsval.Value{jsval.Of("a\u0000b\u001fc")}, "abc"},
		{"stripLow", []jsval.Value{jsval.Of("a\nb"), true}, "a\nb"},
		{"normalizeEmail", []jsval.Value{jsval.Of("Foo.Bar+baz@GoogleMail.com")}, "foobar@gmail.com"},
		{"normalizeEmail", []jsval.Value{jsval.Of("Foo.Bar+baz@gmail.com"), jsval.ObjectOf(jsval.Of("gmail_remove_dots"), false)}, "foo.bar@gmail.com"},
		{"normalizeEmail", []jsval.Value{jsval.Of("Foo.Bar+baz@gmail.com"), jsval.ObjectOf(jsval.Of("gmail_remove_subaddress"), false)}, "foobar+baz@gmail.com"},
		{"normalizeEmail", []jsval.Value{jsval.Of("FOO@OUTLOOK.COM")}, "foo@outlook.com"},
		{"normalizeEmail", []jsval.Value{jsval.Of("a+b@yandex.ru")}, "a+b@yandex.ru"},
		{"normalizeEmail", []jsval.Value{jsval.Of("A.B@icloud.com")}, "a.b@icloud.com"},
		{"normalizeEmail", []jsval.Value{jsval.Of("a@b.co"), jsval.ObjectOf(jsval.Of("all_lowercase"), false)}, "a@b.co"},
		{"normalizeEmail", []jsval.Value{jsval.Of("not an email")}, "@not an email"},
		{"toString", []jsval.Value{jsval.Of("a")}, "[object Undefined]"},
	}
	for _, c := range cases {
		if got := jsval.Str(call(t, c.fn, c.args)).Go(); got != c.want {
			t.Errorf("%s(%v) = %q, want %q", c.fn, c.args, got, c.want)
		}
	}
}

// The number a converter gives back, written as the runtime writes it.
func TestNumbers(t *testing.T) {
	cases := []struct {
		fn   string
		args []jsval.Value
		want string
	}{
		{"toFloat", []jsval.Value{jsval.Of("1.5")}, "1.5"},
		{"toFloat", []jsval.Value{jsval.Of("abc")}, "NaN"},
		{"toFloat", []jsval.Value{jsval.Of("1e21")}, "1e+21"},
		{"toFloat", []jsval.Value{jsval.Of("1e-7")}, "1e-7"},
		{"toInt", []jsval.Value{jsval.Of("42abc")}, "42"},
		{"toInt", []jsval.Value{jsval.Of("1f"), jsval.Num(16)}, "31"},
		{"toInt", []jsval.Value{jsval.Of("abc")}, "NaN"},
		{"toInt", []jsval.Value{jsval.Of("0x1f")}, "0"},
		{"isStrongPassword", []jsval.Value{jsval.Of("Abcdef1!"), jsval.ObjectOf(jsval.Of("returnScore"), true)}, "48"},
	}
	for _, c := range cases {
		value := call(t, c.fn, c.args)
		number, ok := value.(float64)
		if !ok {
			t.Errorf("%s(%v) = %v, want a number", c.fn, c.args, value)
			continue
		}
		if got := jsval.ToString(jsval.Num(number)).Go(); got != c.want {
			t.Errorf("%s(%v) = %s, want %s", c.fn, c.args, got, c.want)
		}
		if c.want == "NaN" && !math.IsNaN(number) {
			t.Errorf("%s(%v) = %v, want NaN", c.fn, c.args, number)
		}
	}
}

// The date toDate gives back, or null for a string it cannot read.
func TestDates(t *testing.T) {
	cases := []struct {
		fn   string
		args []jsval.Value
		want string
	}{
		{"toDate", []jsval.Value{jsval.Of("2020-01-02")}, "2020-01-02T00:00:00.000Z"},
		{"toDate", []jsval.Value{jsval.Of("not a date")}, ""},
		{"toDate", []jsval.Value{jsval.Of("2020-01-02T03:04:05.678Z")}, "2020-01-02T03:04:05.678Z"},
	}
	for _, c := range cases {
		value := call(t, c.fn, c.args)
		if c.want == "" {
			if !jsval.IsNull(value) {
				t.Errorf("%s(%v) = %v, want null", c.fn, c.args, value)
			}
			continue
		}
		date, ok := value.(*jsval.Date)
		if !ok {
			t.Errorf("%s(%v) = %v, want a date", c.fn, c.args, value)
			continue
		}
		if got := date.ToISOString(); got != c.want {
			t.Errorf("%s(%v) = %s, want %s", c.fn, c.args, got, c.want)
		}
	}
}

// What the library raises, with the name and the message of the original.
func TestThrows(t *testing.T) {
	cases := []struct {
		fn   string
		args []jsval.Value
		want string
	}{
		{"isEmail", []jsval.Value{jsval.Num(1)}, "TypeError: Expected a string but received a Number"},
		{"isEmail", []jsval.Value{jsval.Null}, "TypeError: Expected a string but received a null"},
		{"isEmail", []jsval.Value{jsval.Undefined}, "TypeError: Expected a string but received a undefined"},
		{"isEmail", []jsval.Value{true}, "TypeError: Expected a string but received a Boolean"},
		{"isEmail", []jsval.Value{[]jsval.Value{}}, "TypeError: Expected a string but received a Array"},
		{"trim", []jsval.Value{jsval.Num(1)}, "TypeError: Expected a string but received a Number"},
		{"isAlpha", []jsval.Value{jsval.Of("abc"), jsval.Of("xx-XX")}, "Error: Invalid locale 'xx-XX'"},
		{"isMobilePhone", []jsval.Value{jsval.Of("123"), jsval.Of("xx-XX")}, "Error: Invalid locale 'xx-XX'"},
		{"isPostalCode", []jsval.Value{jsval.Of("123"), jsval.Of("xx")}, "Error: Invalid locale 'xx'"},
		{"isTaxID", []jsval.Value{jsval.Of("123"), jsval.Of("xx-XX")}, "Error: Invalid locale 'xx-XX'"},
		{"isIdentityCard", []jsval.Value{jsval.Of("123"), jsval.Of("xx")}, "Error: Invalid locale 'xx'"},
		{"isLicensePlate", []jsval.Value{jsval.Of("123"), jsval.Of("xx")}, "Error: Invalid locale 'xx'"},
		{"matches", []jsval.Value{jsval.Of("a"), jsval.Of("(")}, "SyntaxError: Invalid regular expression: /(/: Unterminated group"},
		{"matches", []jsval.Value{jsval.Of("a"), jsval.Of("a"), jsval.Of("x")}, "SyntaxError: Invalid flags supplied to RegExp constructor 'x'"},
		{"isAlphanumeric", []jsval.Value{jsval.Of("abc"), jsval.Of("xx-XX")}, "Error: Invalid locale 'xx-XX'"},
		{"isDecimal", []jsval.Value{jsval.Of("1"), jsval.ObjectOf(jsval.Of("locale"), jsval.Of("xx-XX"))}, "Error: Invalid locale 'xx-XX'"},
		{"isVAT", []jsval.Value{jsval.Of("DE1"), jsval.Of("xx")}, "Error: Invalid country code: 'xx'"},
		{"isIBAN", []jsval.Value{jsval.Num(1)}, "TypeError: Expected a string but received a Number"},
		{"normalizeEmail", []jsval.Value{jsval.Num(1)}, "TypeError: email.split is not a function"},
	}
	for _, c := range cases {
		_, err := Call(c.fn, c.args)
		if err == nil {
			t.Errorf("%s(%v): no error, want %q", c.fn, c.args, c.want)
			continue
		}
		thrown, ok := err.(*jsval.Thrown)
		if !ok {
			t.Errorf("%s(%v): %v, want a raised value", c.fn, c.args, err)
			continue
		}
		if got := thrown.Name + ": " + thrown.Message; got != c.want {
			t.Errorf("%s(%v) raised %q, want %q", c.fn, c.args, got, c.want)
		}
	}
}

// The exported locale lists, with their length and their first and last entry in order.
func TestLocaleLists(t *testing.T) {
	cases := []struct {
		name        string
		count       int
		first, last string
	}{
		{"isAlphaLocales", 76, "en-US", "fa-AF"},
		{"isAlphanumericLocales", 76, "en-US", "pl-Pl"},
		{"isFloatLocales", 68, "en-US", "pl-Pl"},
		{"isMobilePhoneLocales", 169, "am-AM", "it-CH"},
		{"isPostalCodeLocales", 70, "AD", "ZM"},
		{"passportNumberLocales", 60, "AM", "ZA"},
		{"ibanLocales", 80, "AD", "XK"},
	}
	for _, c := range cases {
		value, ok := Lookup(c.name)
		if !ok {
			t.Errorf("Lookup(%q): not known", c.name)
			continue
		}
		items, ok := jsval.Array(value)
		if !ok {
			t.Errorf("%s: %v, want a list", c.name, value)
			continue
		}
		if len(items) != c.count {
			t.Errorf("%s: %d entries, want %d", c.name, len(items), c.count)
			continue
		}
		if got := jsval.Str(items[0]).Go(); got != c.first {
			t.Errorf("%s[0] = %q, want %q", c.name, got, c.first)
		}
		if got := jsval.Str(items[len(items)-1]).Go(); got != c.last {
			t.Errorf("%s[last] = %q, want %q", c.name, got, c.last)
		}
		if IsFunc(c.name) {
			t.Errorf("IsFunc(%q) = true, want false", c.name)
		}
	}
}

// The library's property names, in the order the original exports them.
func TestNamesInOrder(t *testing.T) {
	want := []string{
		"version", "toDate", "toFloat", "toInt", "toBoolean", "equals", "contains", "matches", "isEmail", "isURL",
		"isMACAddress", "isIP", "isIPRange", "isFQDN", "isBoolean", "isIBAN", "isBIC", "isAbaRouting", "isAlpha",
		"isAlphaLocales", "isAlphanumeric", "isAlphanumericLocales", "isNumeric", "isPassportNumber",
		"passportNumberLocales", "isPort", "isLowercase", "isUppercase", "isAscii", "isFullWidth", "isHalfWidth",
		"isVariableWidth", "isMultibyte", "isSemVer", "isSurrogatePair", "isInt", "isIMEI", "isFloat", "isFloatLocales",
		"isDecimal", "isHexadecimal", "isOctal", "isDivisibleBy", "isHexColor", "isRgbColor", "isHSL", "isISRC", "isMD5",
		"isHash", "isJWT", "isJSON", "isEmpty", "isLength", "isLocale", "isByteLength", "isULID", "isUUID", "isMongoId",
		"isAfter", "isBefore", "isIn", "isLuhnNumber", "isCreditCard", "isIdentityCard", "isEAN", "isISIN", "isISBN",
		"isISSN", "isMobilePhone", "isMobilePhoneLocales", "isPostalCode", "isPostalCodeLocales", "isEthereumAddress",
		"isCurrency", "isBtcAddress", "isISO6346", "isFreightContainerID", "isISO6391", "isISO8601", "isISO15924",
		"isRFC3339", "isISO31661Alpha2", "isISO31661Alpha3", "isISO31661Numeric", "isISO4217", "isBase32", "isBase58",
		"isBase64", "isDataURI", "isMagnetURI", "isMailtoURI", "isMimeType", "isLatLong", "ltrim", "rtrim", "trim",
		"escape", "unescape", "stripLow", "whitelist", "blacklist", "isWhitelisted", "normalizeEmail", "toString",
		"isSlug", "isStrongPassword", "isTaxID", "isDate", "isTime", "isLicensePlate", "isVAT", "ibanLocales",
	}
	got := Names()
	if len(got) != len(want) {
		t.Fatalf("Names() has %d names, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(\"nope\"): known, want unknown")
	}
}
