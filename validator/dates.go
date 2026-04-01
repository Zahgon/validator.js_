// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"math"
	"time"

	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

var isDateDefaults = jsval.ObjectOf(
	jsval.Of("format"), jsval.Of("YYYY/MM/DD"),
	jsval.Of("delimiters"), []jsval.Value{jsval.Of("/"), jsval.Of("-")},
	jsval.Of("strictMode"), false,
)

var reValidDateFormat = re("(^(y{4}|y{2})[.\\/-](m{1,2})[.\\/-](d{1,2})$)|(^(m{1,2})[.\\/-](d{1,2})[.\\/-]"+
	"((y{4}|y{2})$))|(^(d{1,2})[.\\/-](m{1,2})[.\\/-]((y{4}|y{2})$))", "gi")

// IsDate reports whether the string is a date written in the format the options give. The format may also be
// given on its own, for the sake of older callers.
func IsDate(input, options jsval.Value) jsval.Value {
	var opts *jsval.Object
	if jsval.IsString(options) {
		opts = jsval.Merge(jsval.ObjectOf(jsval.Of("format"), options), isDateDefaults)
	} else {
		opts = jsval.Merge(options, isDateDefaults)
	}
	format := jsval.ToString(opts.Get("format"))
	reValidDateFormat.LastIndex = 0 // the library writes the pattern out at every call, so it never carries
	// a position over from the call before
	if jsval.IsString(input) && reValidDateFormat.Test(format) {
		s := jsval.Str(input)
		if jsval.Truthy(opts.Get("strictMode")) && len(s) != len(format) {
			return false
		}
		delimiters, _ := jsval.Array(opts.Get("delimiters"))
		formatDelimiter := findDelimiter(delimiters, format)
		dateDelimiter := formatDelimiter
		if !jsval.Truthy(opts.Get("strictMode")) {
			dateDelimiter = findDelimiter(delimiters, s)
		}
		dateParts := splitByValue(s, dateDelimiter)
		formatParts := splitByValue(format.ToLower(), formatDelimiter)
		length := len(dateParts)
		if len(formatParts) > length {
			length = len(formatParts)
		}
		fields := map[string]jsstr.String{}
		for i := 0; i < length; i++ {
			var dateWord, formatWord jsstr.String
			present := true
			if i < len(dateParts) {
				dateWord = dateParts[i]
			} else {
				present = false
			}
			if i < len(formatParts) {
				formatWord = formatParts[i]
			} else {
				present = false
			}
			if !present || len(dateWord) == 0 || len(formatWord) == 0 || len(dateWord) != len(formatWord) {
				return false
			}
			fields[formatWord.CharAt(0).Go()] = dateWord
		}
		year, haveYear := fields["y"]
		if !haveYear {
			jsval.Throw("TypeError", "Cannot read properties of undefined (reading 'startsWith')")
		}
		fullYear := year
		if fullYear.HasPrefix("-") {
			return false
		}
		if len(year) == 2 {
			parsed := jsstr.ParseInt(year, 10)
			if math.IsNaN(parsed) {
				return false
			}
			current := float64(time.Now().UTC().Year() % 100)
			if parsed < current {
				fullYear = jsstr.Of("20" + year.Go())
			} else {
				fullYear = jsstr.Of("19" + year.Go())
			}
		}
		month, haveMonth := fields["m"]
		if !haveMonth {
			jsval.Throw("TypeError", "Cannot read properties of undefined (reading 'length')")
		}
		if len(month) == 1 {
			month = jsstr.Of("0" + month.Go())
		}
		day, haveDay := fields["d"]
		if !haveDay {
			jsval.Throw("TypeError", "Cannot read properties of undefined (reading 'length')")
		}
		if len(day) == 1 {
			day = jsstr.Of("0" + day.Go())
		}
		text := jsstr.Of(fullYear.Go() + "-" + month.Go() + "-" + day.Go() + "T00:00:00.000Z")
		parsed := jsval.NewDate(jsval.ParseDate(text))
		if !parsed.Valid() {
			return false
		}
		return float64(parsed.UTCDate()) == jsstr.ToNumber(fields["d"])
	}
	if !jsval.Truthy(opts.Get("strictMode")) {
		date, ok := input.(*jsval.Date)
		return ok && date.Valid()
	}
	return false
}

// findDelimiter returns the first of the delimiters that occurs in the text, or undefined when none does.
func findDelimiter(delimiters []jsval.Value, text jsstr.String) jsval.Value {
	for _, delimiter := range delimiters {
		if text.IndexOf(jsval.ToString(delimiter), 0) != -1 {
			return delimiter
		}
	}
	return jsval.Undefined
}

// splitByValue splits the text on a separator that may be undefined, which splits nothing.
func splitByValue(text jsstr.String, separator jsval.Value) []jsstr.String {
	if jsval.IsUndefined(separator) {
		return []jsstr.String{text}
	}
	return text.SplitStr(jsval.ToString(separator))
}

var isTimeDefaults = jsval.ObjectOf(
	jsval.Of("hourFormat"), jsval.Of("hour24"),
	jsval.Of("mode"), jsval.Of("default"),
)

var timeFormats = map[string]map[string]*jsre.Regexp{
	"hour24": {
		"default":             re("^([01]?[0-9]|2[0-3]):([0-5][0-9])$", ""),
		"withSeconds":         re("^([01]?[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])$", ""),
		"withOptionalSeconds": re("^([01]?[0-9]|2[0-3]):([0-5][0-9])(?::([0-5][0-9]))?$", ""),
	},
	"hour12": {
		"default":             re("^(0?[1-9]|1[0-2]):([0-5][0-9]) (A|P)M$", ""),
		"withSeconds":         re("^(0?[1-9]|1[0-2]):([0-5][0-9]):([0-5][0-9]) (A|P)M$", ""),
		"withOptionalSeconds": re("^(0?[1-9]|1[0-2]):([0-5][0-9])(?::([0-5][0-9]))? (A|P)M$", ""),
	},
}

// IsTime reports whether the string is a time of day in the format the options give.
func IsTime(input, options jsval.Value) jsval.Value {
	opts := jsval.Merge(options, isTimeDefaults)
	hourFormat := jsval.ToString(opts.Get("hourFormat")).Go()
	mode := jsval.ToString(opts.Get("mode")).Go()
	// The source reads the input's type before it reaches into its table of formats, so an input that is not
	// a string answers false even when the options name no format the table holds.
	if !jsval.IsString(input) {
		return false
	}
	formats, ok := timeFormats[hourFormat]
	if !ok {
		jsval.Throw("TypeError", "Cannot read properties of undefined (reading '"+mode+"')")
	}
	pattern, ok := formats[mode]
	if !ok {
		jsval.Throw("TypeError", "Cannot read properties of undefined (reading 'test')")
	}
	return pattern.Test(jsval.Str(input))
}

const iso8601Source = "^([\\+-]?\\d{4}(?!\\d{2}\\b))((-?)((0[1-9]|1[0-2])(\\3([12]\\d|0[1-9]|3[01]))?|" +
	"W([0-4]\\d|5[0-3])(-?[1-7])?|(00[1-9]|0[1-9]\\d|[12]\\d{2}|3([0-5]\\d|6[1-6])))([T%s]((([01]\\d|2[0-3])" +
	"((:?)[0-5]\\d)?|24:?00)([\\.,]\\d+(?!:))?)?(\\17[0-5]\\d([\\.,]\\d+)?)?" +
	"([zZ]|([\\+-])([01]\\d|2[0-3]):?([0-5]\\d)?)?)?)?$"

var (
	reISO8601         = re(replaceOnce(iso8601Source, "%s", "\\s"), "")
	reISO8601StrictT  = re(replaceOnce(iso8601Source, "%s", ""), "")
	reOrdinalDate     = re("^(\\d{4})-?(\\d{3})([ T]{1}\\.*|$)", "")
	reISO8601DateOnly = re("(\\d{4})-?(\\d{0,2})-?(\\d*)", "")
)

func replaceOnce(s, from, to string) string {
	for i := 0; i+len(from) <= len(s); i++ {
		if s[i:i+len(from)] == from {
			return s[:i] + to + s[i+len(from):]
		}
	}
	return s
}

// isValidISODate catches the dates the pattern lets through but the calendar does not have, such as the thirty
// first of February.
func isValidISODate(s jsstr.String) bool {
	if _, m := jsre.Matches(s, reOrdinalDate); m != nil {
		yearText, _ := m.Group(1)
		dayText, _ := m.Group(2)
		year := jsstr.ToNumber(yearText)
		day := jsstr.ToNumber(dayText)
		if int(year)%4 == 0 && int(year)%100 != 0 || int(year)%400 == 0 {
			return day <= 366
		}
		return day <= 365
	}
	_, m := jsre.Matches(s, reISO8601DateOnly)
	if m == nil {
		return true
	}
	group := func(i int) float64 {
		text, ok := m.Group(i)
		if !ok {
			return math.NaN()
		}
		return jsstr.ToNumber(text)
	}
	year, month, day := group(1), group(2), group(3)
	monthString := formatTwo(month)
	dayString := formatTwo(day)
	if monthString == "" {
		monthString = "01"
	}
	if dayString == "" {
		dayString = "01"
	}
	date := jsval.NewDate(jsval.ParseDate(jsstr.Of(
		jsstr.NumberToString(year) + "-" + monthString + "-" + dayString)))
	if month != 0 && !math.IsNaN(month) && day != 0 && !math.IsNaN(day) {
		if !date.Valid() {
			return false
		}
		return float64(date.UTCFullYear()) == year && float64(date.UTCMonth()+1) == month &&
			float64(date.UTCDate()) == day
	}
	return true
}

// formatTwo writes a number with a leading zero and keeps the last two characters, the way the library does; a
// zero or a value that is not a number writes as nothing.
func formatTwo(n float64) string {
	if n == 0 || math.IsNaN(n) {
		return ""
	}
	text := "0" + jsstr.NumberToString(n)
	if len(text) > 2 {
		return text[len(text)-2:]
	}
	return text
}

// IsISO8601 reports whether the string is a date in the international format, optionally insisting on the T
// separator and on the date being one the calendar has.
func IsISO8601(input, options jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	check := reISO8601.Test(s)
	if jsval.Truthy(prop(options, "strictSeparator")) {
		check = reISO8601StrictT.Test(s)
	}
	if check && jsval.Truthy(prop(options, "strict")) {
		return isValidISODate(s)
	}
	return check
}

const (
	rfcDateFullYear = "[0-9]{4}"
	rfcDateMonth    = "(0[1-9]|1[0-2])"
	rfcDateMDay     = "([12]\\d|0[1-9]|3[01])"
	rfcTimeHour     = "([01][0-9]|2[0-3])"
	rfcTimeMinute   = "[0-5][0-9]"
	rfcTimeSecond   = "([0-5][0-9]|60)"
	rfcTimeSecFrac  = "(\\.[0-9]+)?"
	rfcTimeOffset   = "([zZ]|[-+]" + rfcTimeHour + ":" + rfcTimeMinute + ")"
	rfcPartialTime  = rfcTimeHour + ":" + rfcTimeMinute + ":" + rfcTimeSecond + rfcTimeSecFrac
	rfcFullDate     = rfcDateFullYear + "-" + rfcDateMonth + "-" + rfcDateMDay
)

var reRFC3339 = re("^"+rfcFullDate+"[ tT]"+rfcPartialTime+rfcTimeOffset+"$", "")

// IsRFC3339 reports whether the string is a timestamp in the format RFC 3339 describes.
func IsRFC3339(input jsval.Value) jsval.Value { return reRFC3339.Test(str(input)) }

// comparisonDate reads the date to compare against, which may be given on its own or inside the options.
func comparisonDate(options jsval.Value) jsval.Value {
	if jsval.TypeOf(options) == "object" {
		return or(prop(options, "comparisonDate"), jsval.Of(nowText()))
	}
	return or(options, jsval.Of(nowText()))
}

// nowText is the current time written the way the Date function writes it, which is what the library falls back
// to when it is given no date to compare against.
func nowText() string { return jsval.NewDate(float64(nowMilli())).String() }

// nowMilli is the current time in milliseconds since the epoch, which is what the Date constructor of no
// arguments reads.
func nowMilli() int64 { return time.Now().UnixMilli() }

// IsAfter reports whether the date the string holds comes after the date to compare against, the current time by
// default.
func IsAfter(date, options jsval.Value) jsval.Value {
	comparison := ToDate(comparisonDate(options))
	original := ToDate(date)
	left, leftOK := original.(*jsval.Date)
	right, rightOK := comparison.(*jsval.Date)
	return leftOK && rightOK && left.Time > right.Time
}

// IsBefore reports whether the date the string holds comes before the date to compare against, the current time
// by default.
func IsBefore(date, options jsval.Value) jsval.Value {
	comparison := ToDate(comparisonDate(options))
	original := ToDate(date)
	left, leftOK := original.(*jsval.Date)
	right, rightOK := comparison.(*jsval.Date)
	return leftOK && rightOK && left.Time < right.Time
}
