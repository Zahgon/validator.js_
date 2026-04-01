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

var vatSimple = func() map[string]*jsre.Regexp {
	out := map[string]*jsre.Regexp{}
	for _, e := range vatSimpleEntries {
		out[e.locale] = re(e.source, e.flags)
	}
	return out
}()

var (
	rePortugueseVAT = re("^(PT)?(\\d{9})$", "")
	reAustralianVAT = re("^(AU)?(\\d{11})$", "")
	reAustralianAU  = re("^AU", "")
	reSwissVAT      = re("^(CHE[- ]?)?(\\d{9}|(\\d{3}\\.\\d{3}\\.\\d{3})|(\\d{3} \\d{3} \\d{3})) ?"+
		"(TVA|MWST|IVA)?$", "")
	reSwissDigits = re("\\d", "g")
)

// vatPortugal checks the Portuguese number, whose last digit is a modulus eleven check over the first eight.
func vatPortugal(s jsstr.String) bool {
	_, m := jsre.Matches(s, rePortugueseVAT)
	if m == nil {
		return false
	}
	tin, _ := m.Group(2)
	digits := make([]float64, 0, 8)
	for i := 0; i < 8; i++ {
		digits = append(digits, jsstr.ParseInt(tin.CharAt(i), 10))
	}
	checksum := 11 - math.Mod(reverseMultiplyAndSum(digits, 9), 11)
	if checksum > 9 {
		return jsstr.ParseInt(tin.CharAt(8), 10) == 0
	}
	return checksum == jsstr.ParseInt(tin.CharAt(8), 10)
}

// vatAustralia checks the Australian business number, whose weighted total divides by eighty nine.
func vatAustralia(s jsstr.String) bool {
	_, m := jsre.Matches(s, reAustralianVAT)
	if m == nil {
		return false
	}
	weights := []float64{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	stripped := jsre.Replace(s, reAustralianAU, jsstr.String{})
	abn := jsstr.Concat(jsstr.Of(jsstr.NumberToString(jsstr.ParseInt(stripped.Slice(0, 1), 10)-1)),
		stripped.From(1))
	total := float64(0)
	for i := 0; i < 11; i++ {
		total += weights[i] * jsstr.ToNumber(abn.CharAt(i))
	}
	return total != 0 && math.Mod(total, 89) == 0
}

// vatSwitzerland checks the Swiss number, whose last digit is a weighted modulus eleven check.
func vatSwitzerland(s jsstr.String) bool {
	if !reSwissVAT.Test(s) {
		return false
	}
	found, _ := jsre.Matches(s, reSwissDigits)
	if found == nil {
		jsval.Throw("TypeError", "Cannot read properties of null (reading 'map')")
	}
	digits := make([]float64, len(found))
	for i, d := range found {
		digits[i] = jsstr.ToNumber(d)
	}
	last := digits[len(digits)-1]
	digits = digits[:len(digits)-1]
	weights := []float64{5, 4, 3, 2, 7, 6, 5, 4}
	total := float64(0)
	for i, d := range digits {
		weight := float64(0)
		if i < len(weights) {
			weight = weights[i]
		} else {
			return false
		}
		total += d * weight
	}
	calculated := math.Mod(11-math.Mod(total, 11), 11)
	return last == calculated
}

// IsVAT reports whether the string is a value added tax number of the country.
func IsVAT(input, countryCode jsval.Value) jsval.Value {
	s := str(input)
	code := str(countryCode).Go()
	switch code {
	case "PT":
		return vatPortugal(s)
	case "AU":
		return vatAustralia(s)
	case "CH":
		return vatSwitzerland(s)
	}
	if pattern, ok := vatSimple[code]; ok {
		return pattern.Test(s)
	}
	jsval.Throw("Error", "Invalid country code: '"+code+"'")
	return false
}
