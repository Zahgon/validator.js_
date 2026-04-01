// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsval

import (
	"math"

	"github.com/validatorjs/validator/jsstr"
)

// This is the date reader of the runtime the library was written for: the date time string format of the standard
// first, and, for whatever the standard does not cover, the reader the runtime keeps for the dates browsers have
// always accepted. The second one is loose: it reads the numbers and words of the string in order and puts the
// date together out of them afterwards, so that "0.2" is a year and a month and "02-01-2020" is a month and a day
// and a year. The library's own behaviour rests on it wherever it hands a string to the date constructor.

const dateNone = -1 << 30 // the value of a part of the date that was not given

// maxSignificantDigits is how many digits of a numeral the reader keeps; the rest only count towards its length.
const maxSignificantDigits = 9

// dateTokenKind tells the kinds of token the reader's scanner gives back.
type dateTokenKind int

const (
	dateEnd dateTokenKind = iota
	dateNumber
	dateSymbol
	dateWord
	dateWhiteSpace
	dateUnrecognized
	dateInvalid
)

// dateKeywordKind tells what a word of the string means.
type dateKeywordKind int

const (
	kwNone dateKeywordKind = iota
	kwMonthName
	kwAMPM
	kwTimeZoneName
	kwTimeSeparator
)

// dateTok is one token of the string.
type dateTok struct {
	kind   dateTokenKind
	number int
	length int
	symbol byte
	kw     dateKeywordKind
	value  int
}

func (t dateTok) isNumber() bool           { return t.kind == dateNumber }
func (t dateTok) isEnd() bool              { return t.kind == dateEnd }
func (t dateTok) isWhiteSpace() bool       { return t.kind == dateWhiteSpace }
func (t dateTok) isKeyword() bool          { return t.kind == dateWord }
func (t dateTok) isInvalid() bool          { return t.kind == dateInvalid }
func (t dateTok) isSymbol(c byte) bool     { return t.kind == dateSymbol && t.symbol == c }
func (t dateTok) isFixedLength(n int) bool { return t.isNumber() && t.length == n }
func (t dateTok) isAsciiSign() bool {
	return t.kind == dateSymbol && (t.symbol == '+' || t.symbol == '-')
}
func (t dateTok) sign() int {
	if t.symbol == '-' {
		return -1
	}
	return 1
}
func (t dateTok) isKeywordType(k dateKeywordKind) bool { return t.kind == dateWord && t.kw == k }
func (t dateTok) isKeywordZ() bool {
	return t.kind == dateWord && t.kw == kwTimeZoneName && t.value == 0 && t.length == 1
}

// dateKeyword is one word the reader knows, matched on its first three letters.
type dateKeyword struct {
	prefix string
	kind   dateKeywordKind
	value  int
}

// dateKeywords are the words the reader knows: the month names, the two halves of the day, and the time zones it
// still carries. A word longer than three letters only matches a month name.
var dateKeywords = []dateKeyword{
	{"jan", kwMonthName, 1}, {"feb", kwMonthName, 2}, {"mar", kwMonthName, 3}, {"apr", kwMonthName, 4},
	{"may", kwMonthName, 5}, {"jun", kwMonthName, 6}, {"jul", kwMonthName, 7}, {"aug", kwMonthName, 8},
	{"sep", kwMonthName, 9}, {"oct", kwMonthName, 10}, {"nov", kwMonthName, 11}, {"dec", kwMonthName, 12},
	{"am", kwAMPM, 0}, {"pm", kwAMPM, 12},
	{"ut", kwTimeZoneName, 0}, {"utc", kwTimeZoneName, 0}, {"z", kwTimeZoneName, 0},
	{"gmt", kwTimeZoneName, 0}, {"cdt", kwTimeZoneName, -5}, {"cst", kwTimeZoneName, -6},
	{"edt", kwTimeZoneName, -4}, {"est", kwTimeZoneName, -5}, {"mdt", kwTimeZoneName, -6},
	{"mst", kwTimeZoneName, -7}, {"pdt", kwTimeZoneName, -7}, {"pst", kwTimeZoneName, -8},
	{"t", kwTimeSeparator, 0},
}

// lookupDateKeyword matches the first three letters of a word, with the shorter words only matching words of
// their own length and every word longer than three letters only matching a month name.
func lookupDateKeyword(prefix string, length int) (dateKeywordKind, int) {
	for _, keyword := range dateKeywords {
		padded := keyword.prefix
		for len(padded) < 3 {
			padded += "\x00"
		}
		if padded != prefix {
			continue
		}
		if length <= 3 || keyword.kind == kwMonthName {
			return keyword.kind, keyword.value
		}
	}
	return kwNone, 0
}

// dateScanner reads the tokens of the string one at a time, with one token of lookahead.
type dateScanner struct {
	in     jsstr.String
	pos    int
	next   dateTok
	peeked bool
}

func (s *dateScanner) at(i int) int {
	if i >= len(s.in) {
		return -1
	}
	return int(s.in[i])
}

func isDateDigit(c int) bool    { return c >= '0' && c <= '9' }
func isAlphaOrAbove(c int) bool { return c >= 'A' }
func isDateWhiteSpace(c int) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x00a0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return c >= 0x2000 && c <= 0x200a
}

// scan reads the next token of the string.
func (s *dateScanner) scan() dateTok {
	start := s.pos
	c := s.at(s.pos)
	if c < 0 {
		return dateTok{kind: dateEnd}
	}
	if isDateDigit(c) {
		// The value keeps the first nine significant digits; the length counts every digit, leading zeroes
		// among them.
		for s.at(s.pos) == '0' {
			s.pos++
		}
		number, digits := 0, 0
		for isDateDigit(s.at(s.pos)) {
			if digits < maxSignificantDigits {
				number = number*10 + (s.at(s.pos) - '0')
			}
			digits++
			s.pos++
		}
		return dateTok{kind: dateNumber, number: number, length: s.pos - start}
	}
	switch c {
	case ':', '-', '+', '.', ')':
		s.pos++
		return dateTok{kind: dateSymbol, symbol: byte(c)}
	}
	if isAlphaOrAbove(c) && !isDateWhiteSpace(c) {
		prefix := make([]byte, 0, 3)
		length := 0
		for {
			unit := s.at(s.pos)
			if !isAlphaOrAbove(unit) || isDateWhiteSpace(unit) {
				break
			}
			if length < 3 {
				prefix = append(prefix, byte(unit|32))
			}
			length++
			s.pos++
		}
		for len(prefix) < 3 {
			prefix = append(prefix, 0)
		}
		kind, value := lookupDateKeyword(string(prefix), length)
		if kind == kwNone {
			return dateTok{kind: dateWord, kw: kwNone, length: length}
		}
		return dateTok{kind: dateWord, kw: kind, value: value, length: length}
	}
	if isDateWhiteSpace(c) {
		s.pos++
		return dateTok{kind: dateWhiteSpace, length: s.pos - start}
	}
	if c == '(' {
		// Everything in brackets is a comment, as it is in the dates the runtime itself writes.
		balance := 0
		for {
			unit := s.at(s.pos)
			if unit < 0 {
				break
			}
			if unit == ')' {
				balance--
			} else if unit == '(' {
				balance++
			}
			s.pos++
			if balance <= 0 {
				break
			}
		}
		return dateTok{kind: dateUnrecognized}
	}
	s.pos++
	return dateTok{kind: dateUnrecognized}
}

// Next gives the next token and moves past it.
func (s *dateScanner) Next() dateTok {
	if s.peeked {
		s.peeked = false
		return s.next
	}
	return s.scan()
}

// Peek gives the next token without moving past it.
func (s *dateScanner) Peek() dateTok {
	if !s.peeked {
		s.next = s.scan()
		s.peeked = true
	}
	return s.next
}

// SkipSymbol moves past the next token when it is that symbol.
func (s *dateScanner) SkipSymbol(c byte) bool {
	if s.Peek().isSymbol(c) {
		s.Next()
		return true
	}
	return false
}

// dayComposer collects the numbers of the date and decides afterwards which of them is the year, the month and
// the day.
type dayComposer struct {
	comp       [3]int
	index      int
	namedMonth int
	isISO      bool
}

func (d *dayComposer) isEmpty() bool { return d.index == 0 }

func (d *dayComposer) add(n int) bool {
	if d.index < 3 {
		d.comp[d.index] = n
		d.index++
		return true
	}
	return false
}

func isDateMonth(x int) bool { return x >= 1 && x <= 12 }
func isDateDay(x int) bool   { return x >= 1 && x <= 31 }

// write decides the year, the month and the day. The parts that were not given default to one, which is why a
// string of two numbers still reads as a date.
func (d *dayComposer) write() (year, month, day int, ok bool) {
	if d.index < 1 {
		return 0, 0, 0, false
	}
	for d.index < 3 {
		d.comp[d.index] = 1
		d.index++
	}
	year, month, day = 0, dateNone, dateNone
	if d.namedMonth == dateNone {
		if d.isISO || !isDateDay(d.comp[0]) {
			year, month, day = d.comp[0], d.comp[1], d.comp[2]
		} else {
			month, day, year = d.comp[0], d.comp[1], d.comp[2]
		}
	} else {
		month = d.namedMonth
		if !isDateDay(d.comp[0]) {
			year, day = d.comp[0], d.comp[1]
		} else {
			day, year = d.comp[0], d.comp[1]
		}
	}
	if !d.isISO {
		if year >= 0 && year <= 49 {
			year += 2000
		} else if year >= 50 && year <= 99 {
			year += 1900
		}
	}
	if year < -(1<<30) || year > 1<<30 || !isDateMonth(month) || !isDateDay(day) {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

// timeComposer collects the numbers of the time of day.
type timeComposer struct {
	comp       [4]int
	index      int
	hourOffset int
}

func (t *timeComposer) isEmpty() bool { return t.index == 0 }

func isDateMinute(x int) bool      { return x >= 0 && x <= 59 }
func isDateHour(x int) bool        { return x >= 0 && x <= 23 }
func isDateHour12(x int) bool      { return x >= 0 && x <= 12 }
func isDateSecond(x int) bool      { return x >= 0 && x <= 59 }
func isDateMillisecond(x int) bool { return x >= 0 && x <= 999 }

// isExpecting reports whether the number can only be the next part of the time of day.
func (t *timeComposer) isExpecting(n int) bool {
	return (t.index == 1 && isDateMinute(n)) || (t.index == 2 && isDateSecond(n)) ||
		(t.index == 3 && isDateMillisecond(n))
}

func (t *timeComposer) add(n int) bool {
	if t.index < 4 {
		t.comp[t.index] = n
		t.index++
		return true
	}
	return false
}

func (t *timeComposer) addFinal(n int) bool {
	if !t.add(n) {
		return false
	}
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	return true
}

// write decides the time of day, with the half of the day moving the hour and a twenty-fourth hour allowed when
// nothing else is set.
func (t *timeComposer) write() (hour, minute, second, ms int, ok bool) {
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	hour, minute, second, ms = t.comp[0], t.comp[1], t.comp[2], t.comp[3]
	if t.hourOffset != dateNone {
		if !isDateHour12(hour) {
			return 0, 0, 0, 0, false
		}
		hour %= 12
		hour += t.hourOffset
	}
	if !isDateHour(hour) || !isDateMinute(minute) || !isDateSecond(second) || !isDateMillisecond(ms) {
		if hour != 24 || minute != 0 || second != 0 || ms != 0 {
			return 0, 0, 0, 0, false
		}
	}
	return hour, minute, second, ms, true
}

// zoneComposer collects the time zone the string carries.
type zoneComposer struct {
	sign   int
	hour   int
	minute int
}

func (z *zoneComposer) set(offsetInHours int) {
	z.sign = 1
	if offsetInHours < 0 {
		z.sign = -1
		offsetInHours = -offsetInHours
	}
	z.hour = offsetInHours
	z.minute = 0
}

func (z *zoneComposer) setSign(sign int) {
	z.sign = 1
	if sign < 0 {
		z.sign = -1
	}
}

func (z *zoneComposer) isEmpty() bool { return z.hour == dateNone }
func (z *zoneComposer) isUTC() bool   { return z.hour == 0 && z.minute == 0 }

// isExpecting reports whether the number can only be the minutes of the time zone.
func (z *zoneComposer) isExpecting(n int) bool {
	return z.hour != dateNone && z.minute == dateNone && isDateMinute(n)
}

// write gives the offset of the time zone in milliseconds, and false when the string carried none, which makes
// the date a local one.
func (z *zoneComposer) write() (float64, bool) {
	if z.sign == dateNone {
		return 0, false
	}
	hour, minute := z.hour, z.minute
	if hour == dateNone {
		hour = 0
	}
	if minute == dateNone {
		minute = 0
	}
	offset := float64(hour)*msPerHour + float64(minute)*msPerMinute
	if z.sign < 0 {
		offset = -offset
	}
	return offset, true
}

// readMilliseconds reads the first three digits of a numeral as milliseconds, going by its length so that the
// leading zeroes count.
func readMilliseconds(tok dateTok) int {
	number, length := tok.number, tok.length
	switch {
	case length == 1:
		number *= 100
	case length == 2:
		number *= 10
	case length > 3:
		if length > maxSignificantDigits {
			length = maxSignificantDigits
		}
		factor := 1
		for length > 3 {
			factor *= 10
			length--
		}
		number /= factor
	}
	return number
}

// ParseDate reads a date the way Date.parse does: the extended format of the standard first, and the reader the
// runtime keeps for everything else after that, carrying over whatever the first one had already read.
func ParseDate(s jsstr.String) float64 {
	scanner := &dateScanner{in: s}
	day := &dayComposer{namedMonth: dateNone}
	time := &timeComposer{hourOffset: dateNone}
	zone := &zoneComposer{sign: dateNone, hour: dateNone, minute: dateNone}

	token := parseStandardDate(scanner, day, time, zone)
	if token.isInvalid() {
		return math.NaN()
	}
	hasReadNumber := !day.isEmpty()

	for ; !token.isEnd(); token = scanner.Next() {
		switch {
		case token.isNumber():
			hasReadNumber = true
			n := token.number
			if scanner.SkipSymbol(':') {
				if scanner.SkipSymbol(':') {
					// An hour followed by two colons leaves the minutes out.
					if !time.isEmpty() {
						return math.NaN()
					}
					time.add(n)
					time.add(0)
				} else {
					if !time.add(n) {
						return math.NaN()
					}
					if scanner.Peek().isSymbol('.') {
						scanner.Next()
					}
				}
			} else if scanner.SkipSymbol('.') && time.isExpecting(n) {
				time.add(n)
				if !scanner.Peek().isNumber() {
					return math.NaN()
				}
				ms := readMilliseconds(scanner.Next())
				if ms < 0 {
					return math.NaN()
				}
				time.addFinal(ms)
			} else if zone.isExpecting(n) {
				zone.minute = n
			} else if time.isExpecting(n) {
				time.addFinal(n)
				// The seconds must be followed by the end of the string, a space, a half of the day or a
				// time zone.
				peek := scanner.Peek()
				if !peek.isEnd() && !peek.isWhiteSpace() && !peek.isKeywordType(kwAMPM) &&
					!peek.isKeywordType(kwTimeZoneName) && !peek.isAsciiSign() {
					return math.NaN()
				}
			} else {
				if !day.add(n) {
					return math.NaN()
				}
				scanner.SkipSymbol('-')
			}
		case token.isKeyword():
			switch {
			case token.kw == kwAMPM && !time.isEmpty():
				time.hourOffset = token.value
			case token.kw == kwMonthName:
				day.namedMonth = token.value
				scanner.SkipSymbol('-')
			case token.kw == kwTimeZoneName:
				zone.set(token.value)
			default:
				// A word the reader does not know is only allowed before the first number, and must be
				// kept apart from it.
				if hasReadNumber {
					return math.NaN()
				}
				if scanner.Peek().isNumber() {
					return math.NaN()
				}
			}
		case token.isAsciiSign() && (zone.isUTC() || !time.isEmpty()):
			// An offset from universal time, which only follows a time or the name of a zone.
			zone.setSign(token.sign())
			n, length := 0, 0
			if scanner.Peek().isNumber() {
				next := scanner.Next()
				n, length = next.number, next.length
			}
			hasReadNumber = true
			switch {
			case scanner.Peek().isSymbol(':'):
				zone.hour = n
				zone.minute = dateNone
			case length == 1 || length == 2:
				zone.hour = n
				zone.minute = 0
			case length > 4:
				// An offset is at most four digits: hours and minutes written together.
				return math.NaN()
			default:
				zone.hour = n / 100
				zone.minute = n % 100
			}
		case (token.isAsciiSign() || token.isSymbol(')')) && hasReadNumber:
			return math.NaN()
		}
	}

	year, month, dayOfMonth, ok := day.write()
	if !ok {
		return math.NaN()
	}
	hour, minute, second, ms, ok := time.write()
	if !ok {
		return math.NaN()
	}
	t := MakeTime(year, month-1, dayOfMonth, hour, minute, second, ms)
	if offset, ok := zone.write(); ok {
		t -= offset
	}
	return clipTime(t)
}

// parseStandardDate reads the date time string format of the standard, giving back the token it stopped at, which
// the loose reader goes on from. Whatever it read before stopping stays in the parts it was given.
func parseStandardDate(scanner *dateScanner, day *dayComposer, time *timeComposer, zone *zoneComposer) dateTok {
	if scanner.Peek().isAsciiSign() {
		// The sign is kept so that the loose reader sees it again when the year that should follow does not.
		signToken := scanner.Next()
		if !scanner.Peek().isFixedLength(6) {
			return signToken
		}
		sign := signToken.sign()
		year := scanner.Next().number
		if sign < 0 && year == 0 {
			return signToken
		}
		day.add(sign * year)
	} else if scanner.Peek().isFixedLength(4) {
		day.add(scanner.Next().number)
	} else {
		return scanner.Next()
	}
	if scanner.SkipSymbol('-') {
		if !scanner.Peek().isFixedLength(2) || !isDateMonth(scanner.Peek().number) {
			return scanner.Next()
		}
		day.add(scanner.Next().number)
		if scanner.SkipSymbol('-') {
			if !scanner.Peek().isFixedLength(2) || !isDateDay(scanner.Peek().number) {
				return scanner.Next()
			}
			day.add(scanner.Next().number)
		}
	}
	if !scanner.Peek().isKeywordType(kwTimeSeparator) {
		if !scanner.Peek().isEnd() {
			return scanner.Next()
		}
	} else {
		scanner.Next()
		if !scanner.Peek().isFixedLength(2) || scanner.Peek().number < 0 || scanner.Peek().number > 24 {
			return dateTok{kind: dateInvalid}
		}
		// A twenty-fourth hour is only a date when nothing follows it.
		hourIs24 := scanner.Peek().number == 24
		time.add(scanner.Next().number)
		if !scanner.SkipSymbol(':') {
			return dateTok{kind: dateInvalid}
		}
		if !scanner.Peek().isFixedLength(2) || !isDateMinute(scanner.Peek().number) ||
			(hourIs24 && scanner.Peek().number > 0) {
			return dateTok{kind: dateInvalid}
		}
		time.add(scanner.Next().number)
		if scanner.SkipSymbol(':') {
			if !scanner.Peek().isFixedLength(2) || !isDateSecond(scanner.Peek().number) ||
				(hourIs24 && scanner.Peek().number > 0) {
				return dateTok{kind: dateInvalid}
			}
			time.add(scanner.Next().number)
			if scanner.SkipSymbol('.') {
				if !scanner.Peek().isNumber() || (hourIs24 && scanner.Peek().number > 0) {
					return dateTok{kind: dateInvalid}
				}
				ms := readMilliseconds(scanner.Next())
				if ms < 0 {
					return dateTok{kind: dateInvalid}
				}
				time.addFinal(ms)
			}
		}
		if scanner.Peek().isKeywordZ() {
			scanner.Next()
			zone.set(0)
		} else if scanner.Peek().isSymbol('+') || scanner.Peek().isSymbol('-') {
			zone.setSign(scanner.Next().sign())
			if scanner.Peek().isFixedLength(4) {
				hourmin := scanner.Next().number
				zone.hour = hourmin / 100
				zone.minute = hourmin % 100
			} else {
				if !scanner.Peek().isFixedLength(2) {
					return dateTok{kind: dateInvalid}
				}
				zone.hour = scanner.Next().number
				if !scanner.SkipSymbol(':') {
					return dateTok{kind: dateInvalid}
				}
				if !scanner.Peek().isFixedLength(2) {
					return dateTok{kind: dateInvalid}
				}
				zone.minute = scanner.Next().number
			}
		}
		if !scanner.Peek().isEnd() {
			return dateTok{kind: dateInvalid}
		}
	}
	// A date without a time of day and without a zone is a universal one; so is a date with a time of day
	// here, because the runtime's own zone is universal time.
	if zone.isEmpty() && time.isEmpty() {
		zone.set(0)
	}
	day.isISO = true
	return scanner.Next()
}
