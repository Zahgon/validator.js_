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

// identityCheck is one country's national identity number check.
type identityCheck struct {
	locale string
	fn     func(jsstr.String) jsval.Value
}

// digitsOf reads every code unit of the string as a decimal number, the way the library's
// split(”).map(a => parseInt(a, 10)) does; a unit that is not a digit becomes NaN.
func digitsOf(s jsstr.String) []float64 {
	out := make([]float64, len(s))
	for i := range s {
		out[i] = jsstr.ParseInt(s.CharAt(i), 10)
	}
	return out
}

// numbersOf reads every code unit of the string with Number, which differs from parseInt: a space becomes zero
// and anything else that is not a digit becomes NaN.
func numbersOf(s jsstr.String) []float64 {
	out := make([]float64, len(s))
	for i := range s {
		out[i] = jsstr.ToNumber(s.CharAt(i))
	}
	return out
}

// at returns the number at an index of a list that may be shorter, which reads as undefined and so as NaN.
func at(digits []float64, i int) float64 {
	if i < 0 || i >= len(digits) {
		return math.NaN()
	}
	return digits[i]
}

// localDate builds a date out of a year, a month and a day the way the Date constructor of three arguments does:
// a year below one hundred belongs to the twentieth century, and a part that is not a number makes the date
// invalid. The runtime reads the parts in local time, which is universal time here.
func localDate(year, month, day float64) *jsval.Date {
	if math.IsNaN(year) || math.IsNaN(month) || math.IsNaN(day) ||
		math.IsInf(year, 0) || math.IsInf(month, 0) || math.IsInf(day, 0) {
		return jsval.NewDate(math.NaN())
	}
	y := math.Trunc(year)
	if y >= 0 && y <= 99 {
		y += 1900
	}
	return jsval.NewDate(jsval.MakeTime(int(y), int(math.Trunc(month)), int(math.Trunc(day)), 0, 0, 0, 0))
}

// isDateText asks isDate about a date written in one of the library's own formats.
func isDateText(text jsstr.String, format string) bool {
	return jsval.Truthy(IsDate(text, jsval.Of(format)))
}

var (
	reIdentityES      = re("^[0-9X-Z][0-9]{7}[TRWAGMYFPDXBNJZSQVHLCKE]$", "")
	reIdentityESLead  = re("[X,Y,Z]", "g")
	reIdentityFI      = re("^\\d{6}[\\-A\\+]\\d{3}[0-9ABCDEFHJKLMNPRSTUVWXY]{1}$", "")
	reIdentityIN      = re("^[1-9]\\d{3}\\s?\\d{4}\\s?\\d{4}$", "")
	reIdentityINSpace = re("\\s", "g")
	reIdentityIR      = re("^\\d{10}$", "")
	reIdentityIT      = re("C[A-Z]\\d{5}[A-Z]{2}", "is")
	reIdentityTH      = re("^[1-8]\\d{12}$", "")
	reIdentityLKOld   = re("^[1-9]\\d{8}[vx]$", "i")
	reIdentityLKNew   = re("^[1-9]\\d{11}$", "i")
	reIdentityHEIL    = re("^\\d{9}$", "")
	reIdentityARLY    = re("^(1|2)\\d{11}$", "")
	reIdentityARTN    = re("^\\d{8}$", "")
	reIdentityCN15    = re("^[1-9]\\d{7}((0[1-9])|(1[0-2]))((0[1-9])|([1-2][0-9])|(3[0-1]))\\d{3}$", "")
	reIdentityCN18    = re("^[1-9]\\d{5}[1-9]\\d{3}((0[1-9])|(1[0-2]))((0[1-9])|([1-2][0-9])|(3[0-1]))\\d{3}(\\d|x|X)$", "")
	reIdentityCNAny   = re("^\\d{15}|(\\d{17}(\\d|x|X))$", "")
	reIdentityHK      = re("^[A-Z]{1,2}[0-9]{6}((\\([0-9A]\\))|(\\[[0-9A]\\])|([0-9A]))$", "")
	reIdentityHKDigit = re("^[0-9]$", "")
	reIdentityHKPunct = re("\\[|\\]|\\(|\\)", "g")
	reIdentityTW      = re("^[A-Z][0-9]{9}$", "")
	reIdentityPK      = re("^[1-7][0-9]{4}-[0-9]{7}-[1-9]$", "")
	reIdentityVowel   = re("[AEIOU]", "")
)

// identityPL checks a PESEL number: eleven digits whose last one is the check digit.
func identityPL(s jsstr.String) jsval.Value {
	weights := []float64{1, 3, 7, 9, 1, 3, 7, 9, 1, 3, 0}
	if len(s) == 11 && jsval.Truthy(IsInt(s, jsval.ObjectOf(jsval.Of("allow_leading_zeroes"), true))) {
		digits := s.Chars()[:len(s)-1]
		sum := float64(0)
		for i, digit := range digits {
			sum += jsstr.ToNumber(digit) * weights[i]
		}
		modulo := math.Mod(sum, 10)
		lastDigit := jsstr.ToNumber(s.CharAt(len(s) - 1))
		if (modulo == 0 && lastDigit == 0) || lastDigit == 10-modulo {
			return true
		}
	}
	return false
}

// identityES checks a Spanish identity number, whose last letter is a check character.
func identityES(s jsstr.String) jsval.Value {
	charsValue := map[string]string{"X": "0", "Y": "1", "Z": "2"}
	controlDigits := []string{
		"T", "R", "W", "A", "G", "M", "Y", "F", "P", "D", "X", "B",
		"N", "J", "Z", "S", "Q", "V", "H", "L", "C", "K", "E",
	}
	sanitized := s.Trim().ToUpper()
	if !reIdentityES.Test(sanitized) {
		return false
	}
	number := jsre.ReplaceFunc(sanitized.Slice(0, -1), reIdentityESLead, func(m *jsre.Match) jsstr.String {
		return jsstr.Of(charsValue[m.Text().Go()])
	})
	index := math.Mod(jsstr.ToNumber(number), 23)
	if math.IsNaN(index) || index < 0 || int(index) >= len(controlDigits) {
		return sanitized.EndsWith(jsstr.Of("undefined"))
	}
	return sanitized.EndsWith(jsstr.Of(controlDigits[int(index)]))
}

// identityFI checks a Finnish personal identity code, whose last character follows from the rest modulo 31.
func identityFI(s jsstr.String) jsval.Value {
	if len(s) != 11 {
		return false
	}
	if !reIdentityFI.Test(s) {
		return false
	}
	checkDigits := jsstr.Of("0123456789ABCDEFHJKLMNPRSTUVWXY")
	idAsNumber := jsstr.ParseInt(s.Slice(0, 6), 10)*1000 + jsstr.ParseInt(s.Slice(7, 10), 10)
	remainder := math.Mod(idAsNumber, 31)
	if math.IsNaN(remainder) {
		return false
	}
	return checkDigits.CharAt(int(remainder)).Equal(s.Slice(10, 11))
}

var (
	identityINMultiply = [10][10]int{
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
	identityINPermute = [8][10]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
		{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
		{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
		{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
		{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
		{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
		{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
	}
)

// identityIN checks an Aadhaar number with the Verhoeff check its digits carry.
func identityIN(s jsstr.String) jsval.Value {
	sanitized := s.Trim()
	if !reIdentityIN.Test(sanitized) {
		return false
	}
	c := 0
	inverted := numbersOf(jsre.Replace(sanitized, reIdentityINSpace, jsstr.String{}))
	for i, j := 0, len(inverted)-1; i < j; i, j = i+1, j-1 {
		inverted[i], inverted[j] = inverted[j], inverted[i]
	}
	for i, val := range inverted {
		c = identityINMultiply[c][identityINPermute[i%8][int(val)]]
	}
	return c == 0
}

// identityIR checks an Iranian national number modulo eleven.
func identityIR(s jsstr.String) jsval.Value {
	if !reIdentityIR.Test(s) {
		return false
	}
	s = jsstr.Concat(jsstr.Of("0000"), s).From(len(s) - 6)
	if jsstr.ParseInt(s.Slice(3, 9), 10) == 0 {
		return false
	}
	lastNumber := jsstr.ParseInt(s.Slice(9, 10), 10)
	sum := float64(0)
	for i := 0; i < 9; i++ {
		sum += jsstr.ParseInt(s.Slice(i, i+1), 10) * float64(10-i)
	}
	sum = math.Mod(sum, 11)
	return (sum < 2 && lastNumber == sum) || (sum >= 2 && lastNumber == 11-sum)
}

// identityIT checks the shape of an Italian electronic identity card number.
func identityIT(s jsstr.String) jsval.Value {
	if len(s) != 9 {
		return false
	}
	if s.Is("CA00000AA") {
		return false
	}
	return jsre.Search(s, reIdentityIT) > -1
}

// identityNO checks a Norwegian birth number, which carries two check digits.
func identityNO(s jsstr.String) jsval.Value {
	sanitized := s.Trim()
	if math.IsNaN(jsstr.ToNumber(sanitized)) {
		return false
	}
	if len(sanitized) != 11 {
		return false
	}
	if sanitized.Is("00000000000") {
		return false
	}
	f := numbersOf(sanitized)
	k1 := math.Mod(11-math.Mod(3*f[0]+7*f[1]+6*f[2]+1*f[3]+8*f[4]+9*f[5]+4*f[6]+5*f[7]+2*f[8], 11), 11)
	k2 := math.Mod(11-math.Mod(5*f[0]+4*f[1]+3*f[2]+2*f[3]+7*f[4]+6*f[5]+5*f[6]+4*f[7]+3*f[8]+2*k1, 11), 11)
	if k1 != f[9] || k2 != f[10] {
		return false
	}
	return true
}

// identityTH checks a Thai identity number modulo eleven.
func identityTH(s jsstr.String) jsval.Value {
	if !reIdentityTH.Test(s) {
		return false
	}
	sum := float64(0)
	for i := 0; i < 12; i++ {
		sum += jsstr.ParseInt(s.CharAt(i), 10) * float64(13-i)
	}
	return s.CharAt(12).Is(jsstr.NumberToString(math.Mod(11-math.Mod(sum, 11), 10)))
}

// identityLK checks the two shapes of a Sri Lankan identity number.
func identityLK(s jsstr.String) jsval.Value {
	if len(s) == 10 && reIdentityLKOld.Test(s) {
		return true
	} else if len(s) == 12 && reIdentityLKNew.Test(s) {
		return true
	}
	return false
}

// identityHEIL checks an Israeli identity number with its Luhn-like check digit.
func identityHEIL(s jsstr.String) jsval.Value {
	sanitized := s.Trim()
	if !reIdentityHEIL.Test(sanitized) {
		return false
	}
	sum := float64(0)
	for i := 0; i < len(sanitized); i++ {
		incNum := jsstr.ToNumber(sanitized.CharAt(i)) * float64(i%2+1)
		if incNum > 9 {
			sum += incNum - 9
		} else {
			sum += incNum
		}
	}
	return math.Mod(sum, 10) == 0
}

// identityARLY checks the shape of a Libyan national identity number.
func identityARLY(s jsstr.String) jsval.Value { return reIdentityARLY.Test(s.Trim()) }

// identityARTN checks the shape of a Tunisian identity number.
func identityARTN(s jsstr.String) jsval.Value { return reIdentityARTN.Test(s.Trim()) }

// identityCNAreas holds the province and city codes a Chinese identity number may start with.
var identityCNAreas = []string{
	"11", "12", "13", "14", "15", "21", "22", "23", "31", "32", "33", "34", "35", "36", "37",
	"41", "42", "43", "44", "45", "46", "50", "51", "52", "53", "54", "61", "62", "63", "64",
	"65", "71", "81", "82", "91",
}

// identityCNParity is the check character of a Chinese identity number of eighteen characters.
func identityCNParity(s jsstr.String) jsstr.String {
	powers := []float64{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	parityBit := []string{"1", "0", "X", "9", "8", "7", "6", "5", "4", "3", "2"}
	id17 := s.Substring(0, 17)
	power := float64(0)
	for i := 0; i < 17; i++ {
		power += jsstr.ParseInt(id17.CharAt(i), 10) * powers[i]
	}
	mod := math.Mod(power, 11)
	if math.IsNaN(mod) || mod < 0 || int(mod) >= len(parityBit) {
		return jsstr.Of("undefined")
	}
	return jsstr.Of(parityBit[int(mod)])
}

// identityCNBirthday checks that a birth date of eight digits is a date that has already come.
func identityCNBirthday(code jsstr.String) bool {
	yyyy := jsstr.ParseInt(code.Substring(0, 4), 10)
	mm := jsstr.ParseInt(code.Substring(4, 6), 10)
	dd := jsstr.ParseInt(code.From(6), 10)
	xdata := localDate(yyyy, mm-1, dd)
	now := jsval.NewDate(float64(nowMilli()))
	if xdata.Valid() && xdata.Time > now.Time {
		return false
	} else if xdata.Valid() && float64(xdata.FullYear()) == yyyy &&
		float64(xdata.Month()) == mm-1 && float64(xdata.Day()) == dd {
		return true
	}
	return false
}

// identityZHCN checks a Chinese identity number of fifteen or eighteen characters.
func identityZHCN(s jsstr.String) jsval.Value {
	if !reIdentityCNAny.Test(s) {
		return false
	}
	if len(s) == 15 {
		if !reIdentityCN15.Test(s) {
			return false
		}
		if !includesText(identityCNAreas, s.Substring(0, 2)) {
			return false
		}
		return identityCNBirthday(jsstr.Concat(jsstr.Of("19"), s.Substring(6, 12)))
	}
	if !reIdentityCN18.Test(s) {
		return false
	}
	if !includesText(identityCNAreas, s.Substring(0, 2)) {
		return false
	}
	if !identityCNBirthday(s.Substring(6, 14)) {
		return false
	}
	return identityCNParity(s).Equal(s.CharAt(17).ToUpper())
}

// identityZHHK checks a Hong Kong identity number, whose check character may be bracketed.
func identityZHHK(s jsstr.String) jsval.Value {
	s = s.Trim().ToUpper()
	if !reIdentityHK.Test(s) {
		return false
	}
	s = jsre.Replace(s, reIdentityHKPunct, jsstr.String{})
	if len(s) == 8 {
		s = jsstr.Concat(jsstr.Of("3"), s)
	}
	checkSumVal := float64(0)
	for i := 0; i <= 7; i++ {
		var convertedChar float64
		if !reIdentityHKDigit.Test(s.CharAt(i)) {
			unit, _ := s.CodeUnit(i)
			convertedChar = math.Mod(float64(unit)-55, 11)
		} else {
			convertedChar = jsstr.ToNumber(s.CharAt(i))
		}
		checkSumVal += convertedChar * float64(9-i)
	}
	checkSumVal = math.Mod(checkSumVal, 11)
	var checkSumConverted string
	if checkSumVal == 0 {
		checkSumConverted = "0"
	} else if checkSumVal == 1 {
		checkSumConverted = "A"
	} else {
		checkSumConverted = jsstr.NumberToString(11 - checkSumVal)
	}
	return s.CharAt(len(s) - 1).Is(checkSumConverted)
}

// identityZHTW checks a Taiwanese identity number, whose first letter stands for two digits.
func identityZHTW(s jsstr.String) jsval.Value {
	codes := map[string]float64{
		"A": 10, "B": 11, "C": 12, "D": 13, "E": 14, "F": 15, "G": 16, "H": 17, "I": 34,
		"J": 18, "K": 19, "L": 20, "M": 21, "N": 22, "O": 35, "P": 23, "Q": 24, "R": 25,
		"S": 26, "T": 27, "U": 28, "V": 29, "W": 32, "X": 30, "Y": 31, "Z": 33,
	}
	sanitized := s.Trim().ToUpper()
	if !reIdentityTW.Test(sanitized) {
		return false
	}
	sum := float64(0)
	for index := 0; index < len(sanitized); index++ {
		number := sanitized.CharAt(index)
		if index == 0 {
			code, ok := codes[number.Go()]
			if !ok {
				code = math.NaN()
			}
			sum = math.Mod(code, 10)*9 + math.Floor(code/10)
			continue
		}
		if index == 9 {
			return math.Mod(10-math.Mod(sum, 10)-jsstr.ToNumber(number), 10) == 0
		}
		sum += jsstr.ToNumber(number) * float64(9-index)
	}
	return sum
}

// identityPK checks the shape of a Pakistani identity number.
func identityPK(s jsstr.String) jsval.Value { return reIdentityPK.Test(s.Trim()) }

// identityChecks lists the countries the library checks identity numbers for, in its own order, which is the
// order the locale "any" tries them in.
var identityChecks = []identityCheck{
	{"PL", identityPL},
	{"ES", identityES},
	{"FI", identityFI},
	{"IN", identityIN},
	{"IR", identityIR},
	{"IT", identityIT},
	{"NO", identityNO},
	{"TH", identityTH},
	{"LK", identityLK},
	{"he-IL", identityHEIL},
	{"ar-LY", identityARLY},
	{"ar-TN", identityARTN},
	{"zh-CN", identityZHCN},
	{"zh-HK", identityZHHK},
	{"zh-TW", identityZHTW},
	{"PK", identityPK},
}

// IsIdentityCard reports whether the string is a national identity number of the locale, or of any locale when
// the locale is "any".
//
// The locale is required, exactly as in the original implementation: an omitted or unknown locale raises
// "Invalid locale '<locale>'" (so no locale raises "Invalid locale 'undefined'"), and "any" accepts the
// string when any locale's check does. The original README lists the locale as optional, but its code
// never defaults it, and this port keeps the code's behaviour.
func IsIdentityCard(input, locale jsval.Value) jsval.Value {
	s := str(input)
	name := jsval.ToString(locale).Go()
	for _, check := range identityChecks {
		if check.locale == name {
			return check.fn(s)
		}
	}
	if name == "any" {
		for _, check := range identityChecks {
			if jsval.Truthy(check.fn(s)) {
				return true
			}
		}
		return false
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}
