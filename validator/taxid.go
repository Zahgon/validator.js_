// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

// numText writes a number the way a template literal writes it.
func numText(f float64) string { return jsstr.NumberToString(f) }

// padTwo writes a month or a day with the leading zero the library adds when it is below ten.
func padTwo(f float64) string {
	if f < 10 {
		return "0" + numText(f)
	}
	return numText(f)
}

// taxDate asks isDate about a date the library built out of the parts of a tax number.
func taxDate(text, format string) bool {
	return jsval.Truthy(IsDate(jsval.Of(text), jsval.Of(format)))
}

// charCodeAt reads the code unit at an index, which is NaN past the end of the string.
func charCodeAt(s jsstr.String, i int) float64 {
	unit, ok := s.CodeUnit(i)
	if !ok {
		return math.NaN()
	}
	return float64(unit)
}

/*
 * bg-BG: the personal number carries the birth date in its first six digits and a check digit at the end.
 */
func taxBgBg(tin jsstr.String) bool {
	centuryYear := tin.Slice(0, 2).Go()
	month := jsstr.ParseInt(tin.Slice(2, 4), 10)
	switch {
	case month > 40:
		month -= 40
		centuryYear = "20" + centuryYear
	case month > 20:
		month -= 20
		centuryYear = "18" + centuryYear
	default:
		centuryYear = "19" + centuryYear
	}
	date := centuryYear + "/" + padTwo(month) + "/" + tin.Slice(4, 6).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	multip := []float64{2, 4, 8, 5, 10, 9, 7, 3, 6}
	checksum := float64(0)
	for i, m := range multip {
		checksum += at(digits, i) * m
	}
	if math.Mod(checksum, 11) == 10 {
		checksum = 0
	} else {
		checksum = math.Mod(checksum, 11)
	}
	return checksum == at(digits, 9)
}

/*
 * en-CA: the social insurance number is nine digits with a Luhn check, written out as the library writes it.
 */
func taxCanadianSIN(input jsstr.String) bool {
	chars := input.Chars()
	var doubled strings.Builder
	for i, c := range chars {
		if i%2 != 0 {
			doubled.WriteString(numText(jsstr.ToNumber(c) * 2))
		}
	}
	total := math.NaN()
	first := true
	add := func(v float64) {
		if first {
			total, first = v, false
			return
		}
		total += v
	}
	for i, c := range chars {
		if i%2 == 0 {
			add(jsstr.ToNumber(c))
		}
	}
	for _, c := range jsstr.Of(doubled.String()).Chars() {
		add(jsstr.ToNumber(c))
	}
	return math.Mod(total, 10) == 0
}

var (
	reTaxNonWord    = re("\\W", "")
	reTaxSpaceAll   = re("\\s", "g")
	reTaxCpfPunct   = re("[.\\-/]", "g")
	reTaxAllSame    = re("^(.)\\1+$", "")
	reTaxAllSymbols = re("[-\\\\\\/!@#$%\\^&\\*\\(\\)\\+\\=\\[\\]]+", "g")
	reTaxSlashes    = re("[\\/\\\\]", "g")
	reTaxVowel      = re("[AEIOU]", "")
	reTaxX          = re("X", "")
)

/*
 * cs-CZ: the birth number holds a birth date and is divisible by eleven.
 */
func taxCsCz(tin jsstr.String) bool {
	tin = jsre.Replace(tin, reTaxNonWord, jsstr.String{})
	fullYear := numText(jsstr.ParseInt(tin.Slice(0, 2), 10))
	if len(tin) == 10 {
		if jsstr.ToNumber(jsstr.Of(fullYear)) < 54 {
			fullYear = "20" + fullYear
		} else {
			fullYear = "19" + fullYear
		}
	} else {
		if tin.From(6).Is("000") {
			return false
		}
		if jsstr.ToNumber(jsstr.Of(fullYear)) < 54 {
			fullYear = "19" + fullYear
		} else {
			return false
		}
	}
	if len(fullYear) == 3 {
		fullYear = fullYear[0:2] + "0" + fullYear[2:]
	}
	month := jsstr.ParseInt(tin.Slice(2, 4), 10)
	if month > 50 {
		month -= 50
	}
	if month > 20 {
		if jsstr.ParseInt(jsstr.Of(fullYear), 10) < 2004 {
			return false
		}
		month -= 20
	}
	date := fullYear + "/" + padTwo(month) + "/" + tin.Slice(4, 6).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	if len(tin) == 10 {
		if math.Mod(jsstr.ParseInt(tin, 10), 11) != 0 {
			checkdigit := math.Mod(jsstr.ParseInt(tin.Slice(0, 9), 10), 11)
			if jsstr.ParseInt(jsstr.Of(fullYear), 10) < 1986 && checkdigit == 10 {
				if jsstr.ParseInt(tin.From(9), 10) != 0 {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

/*
 * de-AT: the tax account number carries a Luhn check digit.
 */
func taxDeAt(tin jsstr.String) bool { return luhnCheck(tin) }

/*
 * de-DE: the tax identification number has exactly one repeated digit and an ISO 7064 check digit.
 */
func taxDeDe(tin jsstr.String) bool {
	digits := digitsOf(tin)
	var occurrences []string
	for i := 0; i < len(digits)-1; i++ {
		occurrences = append(occurrences, "")
		for j := 0; j < len(digits)-1; j++ {
			if digits[i] == digits[j] {
				occurrences[i] += numText(float64(j))
			}
		}
	}
	var repeated []string
	for _, o := range occurrences {
		if len(o) > 1 {
			repeated = append(repeated, o)
		}
	}
	if len(repeated) != 2 && len(repeated) != 3 {
		return false
	}
	if len(repeated[0]) == 3 {
		tripLocations := digitsOf(jsstr.Of(repeated[0]))
		recurrent := 0
		for i := 0; i < len(tripLocations)-1; i++ {
			if tripLocations[i]+1 == tripLocations[i+1] {
				recurrent += 1
			}
		}
		if recurrent == 2 {
			return false
		}
	}
	return iso7064Check(tin)
}

/*
 * dk-DK: the personal number holds a birth date whose century follows from the seventh digit.
 */
func taxDkDk(tin jsstr.String) bool {
	tin = jsre.Replace(tin, reTaxNonWord, jsstr.String{})
	year := jsstr.ParseInt(tin.Slice(4, 6), 10)
	yearText := ""
	centuryDigit := tin.Slice(6, 7)
	switch {
	case centuryDigit.Is("0"), centuryDigit.Is("1"), centuryDigit.Is("2"), centuryDigit.Is("3"):
		yearText = "19" + numText(year)
	case centuryDigit.Is("4"), centuryDigit.Is("9"):
		if year < 37 {
			yearText = "20" + numText(year)
		} else {
			yearText = "19" + numText(year)
		}
	default:
		if year < 37 {
			yearText = "20" + numText(year)
		} else if year > 58 {
			yearText = "18" + numText(year)
		} else {
			return false
		}
	}
	if len(yearText) == 3 {
		yearText = yearText[0:2] + "0" + yearText[2:]
	}
	date := yearText + "/" + tin.Slice(2, 4).Go() + "/" + tin.Slice(0, 2).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	checksum := float64(0)
	weight := float64(4)
	for i := 0; i < 9; i++ {
		checksum += at(digits, i) * weight
		weight -= 1
		if weight == 1 {
			weight = 7
		}
	}
	checksum = math.Mod(checksum, 11)
	if checksum == 1 {
		return false
	}
	if checksum == 0 {
		return at(digits, 9) == 0
	}
	return at(digits, 9) == 11-checksum
}

/*
 * el-CY: the tax number ends in a letter that follows from its digits.
 */
func taxElCy(tin jsstr.String) bool {
	digits := digitsOf(tin.Slice(0, 8))
	checksum := float64(0)
	for i := 1; i < len(digits); i += 2 {
		checksum += digits[i]
	}
	for i := 0; i < len(digits); i += 2 {
		if digits[i] < 2 {
			checksum += 1 - digits[i]
		} else {
			checksum += 2*(digits[i]-2) + 5
			if digits[i] > 4 {
				checksum += 2
			}
		}
	}
	code := math.Mod(checksum, 26) + 65
	if math.IsNaN(code) {
		return false
	}
	return jsstr.FromCharCode(uint16(code)).Equal(tin.CharAt(8))
}

/*
 * el-GR: the tax number carries a check digit over powers of two.
 */
func taxElGr(tin jsstr.String) bool {
	digits := digitsOf(tin)
	checksum := float64(0)
	for i := 0; i < 8; i++ {
		checksum += at(digits, i) * math.Pow(2, float64(8-i))
	}
	return math.Mod(math.Mod(checksum, 11), 10) == at(digits, 8)
}

/*
 * en-IE: the personal public service number ends in a check letter.
 */
func taxEnIe(tin jsstr.String) bool {
	chars := tin.Chars()
	head := make([]float64, 0, 7)
	for i := 0; i < 7 && i < len(chars); i++ {
		head = append(head, jsstr.ParseInt(chars[i], 10))
	}
	checksum := reverseMultiplyAndSum(head, 8)
	if len(tin) == 9 && !tin.CharAt(8).Is("W") {
		checksum += (charCodeAt(tin, 8) - 64) * 9
	}
	checksum = math.Mod(checksum, 23)
	if checksum == 0 {
		return tin.CharAt(7).ToUpper().Is("W")
	}
	if math.IsNaN(checksum) {
		return false
	}
	return tin.CharAt(7).ToUpper().Equal(jsstr.FromCharCode(uint16(64 + checksum)))
}

// taxEnUsCampusPrefix holds the prefixes the American employer identification numbers are assigned from, by
// campus, in the order the library lists them.
var taxEnUsCampusPrefix = [][]string{
	{"10", "12"},
	{"60", "67"},
	{"50", "53"},
	{"01", "02", "03", "04", "05", "06", "11", "13", "14", "16", "21", "22", "23", "25", "34", "51",
		"52", "54", "55", "56", "57", "58", "59", "65"},
	{"30", "32", "35", "36", "37", "38", "61"},
	{"15", "24"},
	{"20", "26", "27", "45", "46", "47"},
	{"40", "44"},
	{"94", "95"},
	{"80", "90"},
	{"33", "39", "41", "42", "43", "46", "48", "62", "63", "64", "66", "68", "71", "72", "73", "74",
		"75", "76", "77", "81", "82", "83", "84", "85", "86", "87", "88", "91", "92", "93", "98", "99"},
	{"31"},
}

/*
 * en-US: the employer identification number starts with a campus prefix.
 */
func taxEnUs(tin jsstr.String) bool {
	prefix := tin.Slice(0, 2)
	for _, campus := range taxEnUsCampusPrefix {
		for _, p := range campus {
			if prefix.Is(p) {
				return true
			}
		}
	}
	return false
}

/*
 * es-AR: the taxpayer number ends in a check digit over a repeating multiplier.
 */
func taxEsAr(tin jsstr.String) bool {
	accum := float64(0)
	chars := tin.Chars()
	if len(chars) == 0 {
		return false
	}
	digit := jsstr.ParseInt(chars[len(chars)-1], 10)
	chars = chars[:len(chars)-1]
	for i := 0; i < len(chars); i++ {
		value := math.NaN()
		if j := 9 - i; j >= 0 && j < len(chars) {
			value = jsstr.ToNumber(chars[j])
		}
		accum += value * float64(2+i%6)
	}
	verif := 11 - math.Mod(accum, 11)
	if verif == 11 {
		verif = 0
	} else if verif == 10 {
		verif = 9
	}
	return digit == verif
}

/*
 * es-ES: the national identity number ends in a letter that follows from its digits.
 */
func taxEsEs(tin jsstr.String) bool {
	chars := tin.ToUpper().Chars()
	if math.IsNaN(jsstr.ParseInt(charsAt(chars, 0), 10)) && len(chars) > 1 {
		leadReplace := float64(0)
		switch {
		case charsAt(chars, 0).Is("Y"):
			leadReplace = 1
		case charsAt(chars, 0).Is("Z"):
			leadReplace = 2
		}
		chars[0] = jsstr.Of(numText(leadReplace))
	} else {
		for len(chars) < 9 {
			chars = append([]jsstr.String{jsstr.Of("0")}, chars...)
		}
	}
	lookup := []string{"T", "R", "W", "A", "G", "M", "Y", "F", "P", "D", "X", "B", "N", "J", "Z", "S",
		"Q", "V", "H", "L", "C", "K", "E"}
	joined := jsstr.Join(chars, "")
	checksum := math.Mod(jsstr.ParseInt(joined.Slice(0, 8), 10), 23)
	if math.IsNaN(checksum) || checksum < 0 || int(checksum) >= len(lookup) {
		return false
	}
	return joined.CharAt(8).Is(lookup[int(checksum)])
}

// charsAt reads a character of a split string, which is undefined past its end and so is empty here.
func charsAt(chars []jsstr.String, i int) jsstr.String {
	if i < 0 || i >= len(chars) {
		return jsstr.String{}
	}
	return chars[i]
}

/*
 * et-EE and lt-LT: the personal code holds a birth date and a check digit, calculated twice when needed.
 */
func taxEtEe(tin jsstr.String) bool {
	fullYear := tin.Slice(1, 3).Go()
	centuryDigit := tin.Slice(0, 1)
	switch {
	case centuryDigit.Is("1"), centuryDigit.Is("2"):
		fullYear = "18" + fullYear
	case centuryDigit.Is("3"), centuryDigit.Is("4"):
		fullYear = "19" + fullYear
	default:
		fullYear = "20" + fullYear
	}
	date := fullYear + "/" + tin.Slice(3, 5).Go() + "/" + tin.Slice(5, 7).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	checksum := float64(0)
	weight := float64(1)
	for i := 0; i < 10; i++ {
		checksum += at(digits, i) * weight
		weight += 1
		if weight == 10 {
			weight = 1
		}
	}
	if math.Mod(checksum, 11) == 10 {
		checksum = 0
		weight = 3
		for i := 0; i < 10; i++ {
			checksum += at(digits, i) * weight
			weight += 1
			if weight == 10 {
				weight = 1
			}
		}
		if math.Mod(checksum, 11) == 10 {
			return at(digits, 10) == 0
		}
	}
	return math.Mod(checksum, 11) == at(digits, 10)
}

/*
 * fi-FI: the personal identity code holds a birth date whose century follows from a symbol.
 */
func taxFiFi(tin jsstr.String) bool {
	fullYear := tin.Slice(4, 6).Go()
	centurySymbol := tin.Slice(6, 7)
	switch {
	case centurySymbol.Is("+"):
		fullYear = "18" + fullYear
	case centurySymbol.Is("-"):
		fullYear = "19" + fullYear
	default:
		fullYear = "20" + fullYear
	}
	date := fullYear + "/" + tin.Slice(2, 4).Go() + "/" + tin.Slice(0, 2).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	checksum := math.Mod(jsstr.ParseInt(jsstr.Concat(tin.Slice(0, 6), tin.Slice(7, 10)), 10), 31)
	if checksum < 10 {
		return checksum == jsstr.ParseInt(tin.From(10), 10)
	}
	checksum -= 10
	lettersLookup := []string{"A", "B", "C", "D", "E", "F", "H", "J", "K", "L", "M", "N", "P", "R",
		"S", "T", "U", "V", "W", "X", "Y"}
	if math.IsNaN(checksum) || checksum < 0 || int(checksum) >= len(lettersLookup) {
		return false
	}
	return tin.From(10).Is(lettersLookup[int(checksum)])
}

/*
 * fr-BE and nl-BE: the national number holds a birth date and two check digits.
 */
func taxFrBe(tin jsstr.String) bool {
	if !tin.Slice(2, 4).Is("00") || !tin.Slice(4, 6).Is("00") {
		date := tin.Slice(0, 2).Go() + "/" + tin.Slice(2, 4).Go() + "/" + tin.Slice(4, 6).Go()
		if !taxDate(date, "YY/MM/DD") {
			return false
		}
	}
	checksum := 97 - math.Mod(jsstr.ParseInt(tin.Slice(0, 9), 10), 97)
	checkdigits := jsstr.ParseInt(tin.Slice(9, 11), 10)
	if checksum != checkdigits {
		checksum = 97 - math.Mod(jsstr.ParseInt(jsstr.Concat(jsstr.Of("2"), tin.Slice(0, 9)), 10), 97)
		if checksum != checkdigits {
			return false
		}
	}
	return true
}

/*
 * fr-FR: the reference tax number ends in three check digits modulo five hundred and eleven.
 */
func taxFrFr(tin jsstr.String) bool {
	tin = jsre.Replace(tin, reTaxSpaceAll, jsstr.String{})
	checksum := math.Mod(jsstr.ParseInt(tin.Slice(0, 10), 10), 511)
	checkdigits := jsstr.ParseInt(tin.Slice(10, 13), 10)
	return checksum == checkdigits
}

/*
 * fr-LU and lb-LU: the personal identification number holds a birth date and passes Luhn and Verhoeff.
 */
func taxFrLu(tin jsstr.String) bool {
	date := tin.Slice(0, 4).Go() + "/" + tin.Slice(4, 6).Go() + "/" + tin.Slice(6, 8).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	if !luhnCheck(tin.Slice(0, 12)) {
		return false
	}
	return verhoeffCheck(jsstr.Concat(tin.Slice(0, 11), tin.CharAt(12)))
}

/*
 * hr-HR: the personal identification number carries an ISO 7064 check digit.
 */
func taxHrHr(tin jsstr.String) bool { return iso7064Check(tin) }

/*
 * hu-HU: the tax identification number ends in a check digit.
 */
func taxHuHu(tin jsstr.String) bool {
	digits := digitsOf(tin)
	checksum := float64(8)
	for i := 1; i < 9; i++ {
		checksum += at(digits, i) * float64(i+1)
	}
	return math.Mod(checksum, 11) == at(digits, 9)
}

/*
 * it-IT: the three letters of a name in a fiscal code may only hold vowels after the first vowel, then X.
 */
func taxItItName(name []jsstr.String) bool {
	vowelflag := false
	xflag := false
	for i := 0; i < 3; i++ {
		c := charsAt(name, i)
		switch {
		case !vowelflag && reTaxVowel.Test(c):
			vowelflag = true
		case !xflag && vowelflag && c.Is("X"):
			xflag = true
		case i > 0:
			if vowelflag && !xflag {
				if !reTaxVowel.Test(c) {
					return false
				}
			}
			if xflag {
				if !reTaxX.Test(c) {
					return false
				}
			}
		}
	}
	return true
}

/*
 * it-IT: the fiscal code holds a name, a birth date and a check character.
 */
func taxItIt(tin jsstr.String) bool {
	chars := tin.ToUpper().Chars()
	if !taxItItName(sliceChars(chars, 0, 3)) {
		return false
	}
	if !taxItItName(sliceChars(chars, 3, 6)) {
		return false
	}
	numberLocations := []int{6, 7, 9, 10, 12, 13, 14}
	numberReplace := map[string]string{
		"L": "0", "M": "1", "N": "2", "P": "3", "Q": "4",
		"R": "5", "S": "6", "T": "7", "U": "8", "V": "9",
	}
	for _, i := range numberLocations {
		if i < len(chars) {
			if replacement, ok := numberReplace[chars[i].Go()]; ok {
				chars[i] = jsstr.Of(replacement)
			}
		}
	}
	monthReplace := map[string]string{
		"A": "01", "B": "02", "C": "03", "D": "04", "E": "05", "H": "06",
		"L": "07", "M": "08", "P": "09", "R": "10", "S": "11", "T": "12",
	}
	month, ok := monthReplace[charsAt(chars, 8).Go()]
	if !ok {
		month = "undefined"
	}
	day := jsstr.ParseInt(jsstr.Concat(charsAt(chars, 9), charsAt(chars, 10)), 10)
	if day > 40 {
		day -= 40
	}
	date := charsAt(chars, 6).Go() + charsAt(chars, 7).Go() + "/" + month + "/" + padTwo(day)
	if !taxDate(date, "YY/MM/DD") {
		return false
	}
	checksum := float64(0)
	for i := 1; i < len(chars)-1; i += 2 {
		charToInt := jsstr.ParseInt(chars[i], 10)
		if math.IsNaN(charToInt) {
			charToInt = charCodeAt(chars[i], 0) - 65
		}
		checksum += charToInt
	}
	oddConvert := map[string]float64{
		"A": 1, "B": 0, "C": 5, "D": 7, "E": 9, "F": 13, "G": 15, "H": 17, "I": 19, "J": 21,
		"K": 2, "L": 4, "M": 18, "N": 20, "O": 11, "P": 3, "Q": 6, "R": 8, "S": 12, "T": 14,
		"U": 16, "V": 10, "W": 22, "X": 25, "Y": 24, "Z": 23, "0": 1, "1": 0,
	}
	for i := 0; i < len(chars)-1; i += 2 {
		charToInt := float64(0)
		if value, ok := oddConvert[chars[i].Go()]; ok {
			charToInt = value
		} else {
			multiplier := jsstr.ParseInt(chars[i], 10)
			charToInt = 2*multiplier + 1
			if multiplier > 4 {
				charToInt += 2
			}
		}
		checksum += charToInt
	}
	code := 65 + math.Mod(checksum, 26)
	if math.IsNaN(code) {
		return false
	}
	return jsstr.FromCharCode(uint16(code)).Equal(charsAt(chars, 15))
}

// sliceChars takes a run of characters out of a split string.
func sliceChars(chars []jsstr.String, start, end int) []jsstr.String {
	if start > len(chars) {
		start = len(chars)
	}
	if end > len(chars) {
		end = len(chars)
	}
	return chars[start:end]
}

/*
 * lv-LV: the personal code holds a birth date and a check digit, unless it is of the new format.
 */
func taxLvLv(tin jsstr.String) bool {
	tin = jsre.Replace(tin, reTaxNonWord, jsstr.String{})
	day := tin.Slice(0, 2)
	if !day.Is("32") {
		month := tin.Slice(2, 4)
		if !month.Is("00") {
			fullYear := tin.Slice(4, 6).Go()
			switch {
			case tin.CharAt(6).Is("0"):
				fullYear = "18" + fullYear
			case tin.CharAt(6).Is("1"):
				fullYear = "19" + fullYear
			default:
				fullYear = "20" + fullYear
			}
			date := fullYear + "/" + tin.Slice(2, 4).Go() + "/" + day.Go()
			if !taxDate(date, "YYYY/MM/DD") {
				return false
			}
		}
		checksum := float64(1101)
		multipLookup := []float64{1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
		for i := 0; i < len(tin)-1; i++ {
			weight := math.NaN()
			if i < len(multipLookup) {
				weight = multipLookup[i]
			}
			checksum -= jsstr.ParseInt(tin.CharAt(i), 10) * weight
		}
		return jsstr.ParseInt(tin.CharAt(10), 10) == math.Mod(checksum, 11)
	}
	return true
}

/*
 * mt-MT: the identity card number has a shape that depends on its last letter.
 */
func taxMtMt(tin jsstr.String) bool {
	if len(tin) != 9 {
		chars := tin.ToUpper().Chars()
		for len(chars) < 8 {
			chars = append([]jsstr.String{jsstr.Of("0")}, chars...)
		}
		last := tin.CharAt(7)
		switch {
		case last.Is("A"), last.Is("P"):
			if jsstr.ParseInt(charsAt(chars, 6), 10) == 0 {
				return false
			}
		default:
			joined := jsstr.Join(chars, "")
			firstPart := jsstr.ParseInt(joined.Slice(0, 5), 10)
			if firstPart > 32000 {
				return false
			}
			secondPart := jsstr.ParseInt(joined.Slice(5, 7), 10)
			if firstPart == secondPart {
				return false
			}
		}
	}
	return true
}

/*
 * nl-NL: the citizen service number ends in a check digit over a falling multiplier.
 */
func taxNlNl(tin jsstr.String) bool {
	chars := tin.Chars()
	head := make([]float64, 0, 8)
	for i := 0; i < 8 && i < len(chars); i++ {
		head = append(head, jsstr.ParseInt(chars[i], 10))
	}
	return math.Mod(reverseMultiplyAndSum(head, 9), 11) == jsstr.ParseInt(tin.CharAt(8), 10)
}

/*
 * pl-PL: the tax number is a company number of ten digits or a personal number holding a birth date.
 */
func taxPlPl(tin jsstr.String) bool {
	if len(tin) == 10 {
		lookup := []float64{6, 5, 7, 2, 3, 4, 5, 6, 7}
		checksum := float64(0)
		for i, m := range lookup {
			checksum += jsstr.ParseInt(tin.CharAt(i), 10) * m
		}
		checksum = math.Mod(checksum, 11)
		if checksum == 10 {
			return false
		}
		return checksum == jsstr.ParseInt(tin.CharAt(9), 10)
	}
	fullYear := tin.Slice(0, 2).Go()
	month := jsstr.ParseInt(tin.Slice(2, 4), 10)
	switch {
	case month > 80:
		fullYear = "18" + fullYear
		month -= 80
	case month > 60:
		fullYear = "22" + fullYear
		month -= 60
	case month > 40:
		fullYear = "21" + fullYear
		month -= 40
	case month > 20:
		fullYear = "20" + fullYear
		month -= 20
	default:
		fullYear = "19" + fullYear
	}
	date := fullYear + "/" + padTwo(month) + "/" + tin.Slice(4, 6).Go()
	if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	checksum := float64(0)
	multiplier := float64(1)
	for i := 0; i < len(tin)-1; i++ {
		checksum += math.Mod(jsstr.ParseInt(tin.CharAt(i), 10)*multiplier, 10)
		multiplier += 2
		if multiplier > 10 {
			multiplier = 1
		} else if multiplier == 5 {
			multiplier += 2
		}
	}
	checksum = 10 - math.Mod(checksum, 10)
	return checksum == jsstr.ParseInt(tin.CharAt(10), 10)
}

// cnpjCharToValue is the value of a character of a company number, which is its code point less forty-eight.
func cnpjCharToValue(c jsstr.String) float64 { return charCodeAt(c, 0) - 48 }

// validateCnpj checks the two check digits of a Brazilian company number, in either of its formats.
func validateCnpj(cnpj jsstr.String) bool {
	identifiers := cnpj.Substring(0, 12).ToUpper()
	checkDigits := cnpj.From(12)
	if reTaxAllSame.Test(cnpj.ToUpper()) {
		return false
	}
	sum := float64(0)
	weight := float64(5)
	for i := 0; i < 12; i++ {
		sum += cnpjCharToValue(identifiers.CharAt(i)) * weight
		if weight == 2 {
			weight = 9
		} else {
			weight -= 1
		}
	}
	remainder := math.Mod(sum, 11)
	firstDV := float64(0)
	if remainder >= 2 {
		firstDV = 11 - remainder
	}
	if firstDV != jsstr.ParseInt(checkDigits.CharAt(0), 10) {
		return false
	}
	sum = 0
	weight = 6
	for i := 0; i < 12; i++ {
		sum += cnpjCharToValue(identifiers.CharAt(i)) * weight
		if weight == 2 {
			weight = 9
		} else {
			weight -= 1
		}
	}
	sum += firstDV * 2
	remainder = math.Mod(sum, 11)
	secondDV := float64(0)
	if remainder >= 2 {
		secondDV = 11 - remainder
	}
	return secondDV == jsstr.ParseInt(checkDigits.CharAt(1), 10)
}

/*
 * pt-BR: the tax number is a personal number of eleven digits or a company number.
 */
func taxPtBr(tin jsstr.String) bool {
	tin = jsre.Replace(tin, reTaxCpfPunct, jsstr.String{})
	if len(tin) == 11 {
		for _, bad := range []string{"11111111111", "22222222222", "33333333333", "44444444444",
			"55555555555", "66666666666", "77777777777", "88888888888", "99999999999", "00000000000"} {
			if tin.Is(bad) {
				return false
			}
		}
		sum := float64(0)
		for i := 1; i <= 9; i++ {
			sum += jsstr.ParseInt(tin.Substring(i-1, i), 10) * float64(11-i)
		}
		remainder := math.Mod(sum*10, 11)
		if remainder == 10 {
			remainder = 0
		}
		if remainder != jsstr.ParseInt(tin.Substring(9, 10), 10) {
			return false
		}
		sum = 0
		for i := 1; i <= 10; i++ {
			sum += jsstr.ParseInt(tin.Substring(i-1, i), 10) * float64(12-i)
		}
		remainder = math.Mod(sum*10, 11)
		if remainder == 10 {
			remainder = 0
		}
		if remainder != jsstr.ParseInt(tin.Substring(10, 11), 10) {
			return false
		}
		return true
	}
	return validateCnpj(tin)
}

/*
 * pt-PT: the tax number ends in a check digit over a falling multiplier.
 */
func taxPtPt(tin jsstr.String) bool {
	chars := tin.Chars()
	head := make([]float64, 0, 8)
	for i := 0; i < 8 && i < len(chars); i++ {
		head = append(head, jsstr.ParseInt(chars[i], 10))
	}
	checksum := 11 - math.Mod(reverseMultiplyAndSum(head, 9), 11)
	if checksum > 9 {
		return jsstr.ParseInt(tin.CharAt(8), 10) == 0
	}
	return checksum == jsstr.ParseInt(tin.CharAt(8), 10)
}

/*
 * ro-RO: the personal numeric code holds a birth date and a check digit.
 */
func taxRoRo(tin jsstr.String) bool {
	if !tin.Slice(0, 4).Is("9000") {
		fullYear := tin.Slice(1, 3).Go()
		switch {
		case tin.CharAt(0).Is("1"), tin.CharAt(0).Is("2"):
			fullYear = "19" + fullYear
		case tin.CharAt(0).Is("3"), tin.CharAt(0).Is("4"):
			fullYear = "18" + fullYear
		case tin.CharAt(0).Is("5"), tin.CharAt(0).Is("6"):
			fullYear = "20" + fullYear
		}
		date := fullYear + "/" + tin.Slice(3, 5).Go() + "/" + tin.Slice(5, 7).Go()
		if len(date) == 8 {
			if !taxDate(date, "YY/MM/DD") {
				return false
			}
		} else if !taxDate(date, "YYYY/MM/DD") {
			return false
		}
		digits := digitsOf(tin)
		multipliers := []float64{2, 7, 9, 1, 4, 6, 3, 5, 8, 2, 7, 9}
		checksum := float64(0)
		for i, m := range multipliers {
			checksum += at(digits, i) * m
		}
		if math.Mod(checksum, 11) == 10 {
			return at(digits, 12) == 1
		}
		return at(digits, 12) == math.Mod(checksum, 11)
	}
	return true
}

/*
 * sk-SK: the birth number of nine characters holds a birth date before 1954.
 */
func taxSkSk(tin jsstr.String) bool {
	if len(tin) == 9 {
		tin = jsre.Replace(tin, reTaxNonWord, jsstr.String{})
		if tin.From(6).Is("000") {
			return false
		}
		year := jsstr.ParseInt(tin.Slice(0, 2), 10)
		if year > 53 {
			return false
		}
		fullYear := ""
		if year < 10 {
			fullYear = "190" + numText(year)
		} else {
			fullYear = "19" + numText(year)
		}
		month := jsstr.ParseInt(tin.Slice(2, 4), 10)
		if month > 50 {
			month -= 50
		}
		date := fullYear + "/" + padTwo(month) + "/" + tin.Slice(4, 6).Go()
		if !taxDate(date, "YYYY/MM/DD") {
			return false
		}
	}
	return true
}

/*
 * sl-SI: the tax number ends in a check digit over a falling multiplier.
 */
func taxSlSi(tin jsstr.String) bool {
	chars := tin.Chars()
	head := make([]float64, 0, 7)
	for i := 0; i < 7 && i < len(chars); i++ {
		head = append(head, jsstr.ParseInt(chars[i], 10))
	}
	checksum := 11 - math.Mod(reverseMultiplyAndSum(head, 8), 11)
	if checksum == 10 {
		return jsstr.ParseInt(tin.CharAt(7), 10) == 0
	}
	return checksum == jsstr.ParseInt(tin.CharAt(7), 10)
}

/*
 * sv-SE: the personal number holds a birth date, with a centenarian form, and a Luhn check digit.
 */
func taxSvSe(tin jsstr.String) bool {
	tinCopy := tin.From(0)
	if len(tin) > 11 {
		tinCopy = tinCopy.From(2)
	}
	fullYear := ""
	month := tinCopy.Slice(2, 4).Go()
	day := jsstr.ParseInt(tinCopy.Slice(4, 6), 10)
	if len(tin) > 11 {
		fullYear = tin.Slice(0, 4).Go()
	} else {
		fullYear = tin.Slice(0, 2).Go()
		if len(tin) == 11 && day < 60 {
			currentYearText := numText(float64(jsval.NewDate(float64(nowMilli())).FullYear()))
			currentCentury := jsstr.ParseInt(jsstr.Of(currentYearText).Slice(0, 2), 10)
			currentYear := jsstr.ParseInt(jsstr.Of(currentYearText), 10)
			if tin.CharAt(6).Is("-") {
				if jsstr.ParseInt(jsstr.Of(numText(currentCentury)+fullYear), 10) > currentYear {
					fullYear = numText(currentCentury-1) + fullYear
				} else {
					fullYear = numText(currentCentury) + fullYear
				}
			} else {
				fullYear = numText(currentCentury-1) + fullYear
				if currentYear-jsstr.ParseInt(jsstr.Of(fullYear), 10) < 100 {
					return false
				}
			}
		}
	}
	if day > 60 {
		day -= 60
	}
	date := fullYear + "/" + month + "/" + padTwo(day)
	if len(date) == 8 {
		if !taxDate(date, "YY/MM/DD") {
			return false
		}
	} else if !taxDate(date, "YYYY/MM/DD") {
		return false
	}
	return luhnCheck(jsre.Replace(tin, reTaxNonWord, jsstr.String{}))
}

/*
 * uk-UA: the tax number ends in a check digit over a fixed multiplier.
 */
func taxUkUa(tin jsstr.String) bool {
	digits := digitsOf(tin)
	multipliers := []float64{-1, 5, 7, 9, 4, 6, 10, 5, 7}
	checksum := float64(0)
	for i, m := range multipliers {
		checksum += at(digits, i) * m
	}
	if math.Mod(checksum, 11) == 10 {
		return at(digits, 9) == 0
	}
	return at(digits, 9) == math.Mod(checksum, 11)
}

// taxIDEntry is one locale's tax number: the shape it must have, the check its digits carry, and the characters
// the library strips before looking at the shape.
type taxIDEntry struct {
	locale   string
	format   *jsre.Regexp
	check    func(jsstr.String) bool
	sanitize *jsre.Regexp
}

// taxIDEntries holds the tax numbers of every locale the library knows, in the order it lists them, with the
// aliases that share a shape and a check.
var taxIDEntries = []taxIDEntry{
	{"bg-BG", re("^\\d{10}$", ""), taxBgBg, nil},
	{"cs-CZ", re("^\\d{6}\\/{0,1}\\d{3,4}$", ""), taxCsCz, nil},
	{"de-AT", re("^\\d{9}$", ""), taxDeAt, reTaxAllSymbols},
	{"de-DE", re("^[1-9]\\d{10}$", ""), taxDeDe, reTaxSlashes},
	{"dk-DK", re("^\\d{6}-{0,1}\\d{4}$", ""), taxDkDk, nil},
	{"el-CY", re("^[09]\\d{7}[A-Z]$", ""), taxElCy, nil},
	{"el-GR", re("^([0-4]|[7-9])\\d{8}$", ""), taxElGr, nil},
	{"en-CA", re("^\\d{9}$", ""), taxCanadianSIN, nil},
	{"en-GB", re("^\\d{10}$|^(?!GB|NK|TN|ZZ)(?![DFIQUV])[A-Z](?![DFIQUVO])[A-Z]\\d{6}[ABCD ]$", "i"), nil, nil},
	{"en-IE", re("^\\d{7}[A-W][A-IW]{0,1}$", "i"), taxEnIe, nil},
	{"en-US", re("^\\d{2}[- ]{0,1}\\d{7}$", ""), taxEnUs, nil},
	{"es-AR", re("(20|23|24|27|30|33|34)[0-9]{8}[0-9]", ""), taxEsAr, nil},
	{"es-ES", re("^(\\d{0,8}|[XYZKLM]\\d{7})[A-HJ-NP-TV-Z]$", "i"), taxEsEs, nil},
	{"et-EE", re("^[1-6]\\d{6}(00[1-9]|0[1-9][0-9]|[1-6][0-9]{2}|70[0-9]|710)\\d$", ""), taxEtEe, nil},
	{"fi-FI", re("^\\d{6}[-+A]\\d{3}[0-9A-FHJ-NPR-Y]$", "i"), taxFiFi, nil},
	{"fr-BE", re("^\\d{11}$", ""), taxFrBe, reTaxAllSymbols},
	{"fr-FR", re("^[0-3]\\d{12}$|^[0-3]\\d\\s\\d{2}(\\s\\d{3}){3}$", ""), taxFrFr, nil},
	{"fr-LU", re("^\\d{13}$", ""), taxFrLu, nil},
	{"hr-HR", re("^\\d{11}$", ""), taxHrHr, nil},
	{"hu-HU", re("^8\\d{9}$", ""), taxHuHu, nil},
	{"it-IT", re("^[A-Z]{6}[L-NP-V0-9]{2}[A-EHLMPRST][L-NP-V0-9]{2}[A-ILMZ][L-NP-V0-9]{3}[A-Z]$", "i"), taxItIt, nil},
	{"lv-LV", re("^\\d{6}-{0,1}\\d{5}$", ""), taxLvLv, nil},
	{"mt-MT", re("^\\d{3,7}[APMGLHBZ]$|^([1-8])\\1\\d{7}$", "i"), taxMtMt, nil},
	{"nl-NL", re("^\\d{9}$", ""), taxNlNl, nil},
	{"pl-PL", re("^\\d{10,11}$", ""), taxPlPl, nil},
	{"pt-BR", re("(?:^\\d{3}\\.\\d{3}\\.\\d{3}-\\d{2}$)|(?:^\\d{11}$)|(?:^[A-Z0-9]{12}\\d{2}$)", "i"), taxPtBr, nil},
	{"pt-PT", re("^\\d{9}$", ""), taxPtPt, nil},
	{"ro-RO", re("^\\d{13}$", ""), taxRoRo, nil},
	{"sk-SK", re("^\\d{6}\\/{0,1}\\d{3,4}$", ""), taxSkSk, nil},
	{"sl-SI", re("^[1-9]\\d{7}$", ""), taxSlSi, nil},
	{"sv-SE", re("^(\\d{6}[-+]{0,1}\\d{4}|(18|19|20)\\d{6}[-+]{0,1}\\d{4})$", ""), taxSvSe, nil},
	{"uk-UA", re("^\\d{10}$", ""), taxUkUa, nil},
}

// taxIDAliases are the locales the library points at another locale's shape and check.
var taxIDAliases = map[string]string{
	"lb-LU": "fr-LU",
	"lt-LT": "et-EE",
	"nl-BE": "fr-BE",
	"fr-CA": "en-CA",
}

// IsTaxID reports whether the string is a taxpayer identification number of the locale, which is the American one
// by default.
func IsTaxID(input, locale jsval.Value) jsval.Value {
	s := str(input)
	name := "en-US"
	if !jsval.IsUndefined(locale) {
		name = jsval.ToString(locale).Go()
	}
	if alias, ok := taxIDAliases[name]; ok {
		name = alias
	}
	for _, entry := range taxIDEntries {
		if entry.locale != name {
			continue
		}
		strcopy := s.From(0)
		if entry.sanitize != nil {
			strcopy = jsre.Replace(strcopy, entry.sanitize, jsstr.String{})
		}
		if !entry.format.Test(strcopy) {
			return false
		}
		if entry.check != nil {
			return entry.check(strcopy)
		}
		return true
	}
	jsval.Throw("Error", "Invalid locale '"+name+"'")
	return false
}
