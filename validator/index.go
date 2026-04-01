// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import "github.com/validatorjs/validator/jsval"

// one registers a function of one argument.
func one(name string, fn func(jsval.Value) jsval.Value) {
	register(name, func(args []jsval.Value) jsval.Value { return fn(arg(args, 0)) })
}

// two registers a function of two arguments.
func two(name string, fn func(a, b jsval.Value) jsval.Value) {
	register(name, func(args []jsval.Value) jsval.Value { return fn(arg(args, 0), arg(args, 1)) })
}

// three registers a function of three arguments.
func three(name string, fn func(a, b, c jsval.Value) jsval.Value) {
	register(name, func(args []jsval.Value) jsval.Value {
		return fn(arg(args, 0), arg(args, 1), arg(args, 2))
	})
}

// ToStringGlobal is the toString the library ends up exposing: the method of the object prototype, which the
// library's source picks up from the global object instead of from its own helper. Called on its own it answers
// for the undefined value.
func ToStringGlobal(args []jsval.Value) jsval.Value { return jsval.Of("[object Undefined]") }

// The library's names are registered in the order its own source lists them, which is the order it reports them
// in.
func init() {
	registerValue("version", jsval.Of(Version))
	one("toDate", ToDate)
	one("toFloat", ToFloat)
	two("toInt", ToInt)
	two("toBoolean", ToBoolean)
	two("equals", Equals)
	three("contains", Contains)
	three("matches", Matches)
	two("isEmail", IsEmail)
	two("isURL", IsURL)
	two("isMACAddress", IsMACAddress)
	register("isIP", IsIP)
	two("isIPRange", IsIPRange)
	two("isFQDN", IsFQDN)
	two("isBoolean", IsBoolean)
	two("isIBAN", IsIBAN)
	one("isBIC", IsBIC)
	one("isAbaRouting", IsAbaRouting)
	three("isAlpha", IsAlpha)
	registerValue("isAlphaLocales", alphaTable.Locales())
	three("isAlphanumeric", IsAlphanumeric)
	registerValue("isAlphanumericLocales", alphanumericTable.Locales())
	two("isNumeric", IsNumeric)
	two("isPassportNumber", IsPassportNumber)
	registerValue("passportNumberLocales", passportTable.Locales())
	one("isPort", IsPort)
	one("isLowercase", IsLowercase)
	one("isUppercase", IsUppercase)
	one("isAscii", IsAscii)
	one("isFullWidth", IsFullWidth)
	one("isHalfWidth", IsHalfWidth)
	one("isVariableWidth", IsVariableWidth)
	one("isMultibyte", IsMultibyte)
	one("isSemVer", IsSemVer)
	one("isSurrogatePair", IsSurrogatePair)
	two("isInt", IsInt)
	two("isIMEI", IsIMEI)
	two("isFloat", IsFloat)
	registerValue("isFloatLocales", decimalTable.Locales())
	two("isDecimal", IsDecimal)
	one("isHexadecimal", IsHexadecimal)
	one("isOctal", IsOctal)
	two("isDivisibleBy", IsDivisibleBy)
	two("isHexColor", IsHexColor)
	register("isRgbColor", IsRgbColor)
	one("isHSL", IsHSL)
	one("isISRC", IsISRC)
	one("isMD5", IsMD5)
	two("isHash", IsHash)
	one("isJWT", IsJWT)
	two("isJSON", IsJSON)
	two("isEmpty", IsEmpty)
	register("isLength", IsLength)
	one("isLocale", IsLocale)
	register("isByteLength", IsByteLength)
	one("isULID", IsULID)
	two("isUUID", IsUUID)
	one("isMongoId", IsMongoId)
	two("isAfter", IsAfter)
	two("isBefore", IsBefore)
	two("isIn", IsIn)
	one("isLuhnNumber", IsLuhnNumber)
	two("isCreditCard", IsCreditCard)
	two("isIdentityCard", IsIdentityCard)
	one("isEAN", IsEAN)
	one("isISIN", IsISIN)
	two("isISBN", IsISBN)
	two("isISSN", IsISSN)
	three("isMobilePhone", IsMobilePhone)
	registerValue("isMobilePhoneLocales", mobilePhoneTable.Locales())
	two("isPostalCode", IsPostalCode)
	registerValue("isPostalCodeLocales", postalCodeTable.Locales())
	one("isEthereumAddress", IsEthereumAddress)
	two("isCurrency", IsCurrency)
	one("isBtcAddress", IsBtcAddress)
	one("isISO6346", IsISO6346)
	one("isFreightContainerID", IsFreightContainerID)
	one("isISO6391", IsISO6391)
	two("isISO8601", IsISO8601)
	one("isISO15924", IsISO15924)
	one("isRFC3339", IsRFC3339)
	two("isISO31661Alpha2", IsISO31661Alpha2)
	two("isISO31661Alpha3", IsISO31661Alpha3)
	one("isISO31661Numeric", IsISO31661Numeric)
	one("isISO4217", IsISO4217)
	two("isBase32", IsBase32)
	one("isBase58", IsBase58)
	two("isBase64", IsBase64)
	one("isDataURI", IsDataURI)
	one("isMagnetURI", IsMagnetURI)
	two("isMailtoURI", IsMailtoURI)
	one("isMimeType", IsMimeType)
	two("isLatLong", IsLatLong)
	two("ltrim", Ltrim)
	two("rtrim", Rtrim)
	two("trim", Trim)
	one("escape", Escape)
	one("unescape", Unescape)
	two("stripLow", StripLow)
	two("whitelist", Whitelist)
	two("blacklist", Blacklist)
	two("isWhitelisted", IsWhitelisted)
	two("normalizeEmail", NormalizeEmail)
	register("toString", ToStringGlobal)
	one("isSlug", IsSlug)
	two("isStrongPassword", IsStrongPassword)
	two("isTaxID", IsTaxID)
	two("isDate", IsDate)
	two("isTime", IsTime)
	two("isLicensePlate", IsLicensePlate)
	two("isVAT", IsVAT)
	registerValue("ibanLocales", ibanTable.Locales())
}
