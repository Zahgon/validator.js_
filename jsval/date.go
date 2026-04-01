// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsval

import (
	"fmt"
	"math"
)

// Date is a point in time held as milliseconds since the epoch, not a number when the date is invalid.
type Date struct {
	// Time is the number of milliseconds since the epoch.
	Time float64
}

// The time zone of the runtime the library is ported from is set to UTC, so local time and universal time are the
// same here.

const (
	msPerSecond = 1000
	msPerMinute = 60 * msPerSecond
	msPerHour   = 60 * msPerMinute
	msPerDay    = 24 * msPerHour
)

var (
	monthNames = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	dayNames   = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	zoneNames  = map[string]int{
		"gmt": 0, "ut": 0, "utc": 0, "z": 0,
		"est": -5 * 60, "edt": -4 * 60, "cst": -6 * 60, "cdt": -5 * 60,
		"mst": -7 * 60, "mdt": -6 * 60, "pst": -8 * 60, "pdt": -7 * 60,
	}
)

// NewDate makes a date out of milliseconds since the epoch.
func NewDate(ms float64) *Date { return &Date{Time: clipTime(ms)} }

func clipTime(ms float64) float64 {
	if math.IsNaN(ms) || math.Abs(ms) > 8.64e15 {
		return math.NaN()
	}
	return math.Trunc(ms) + 0
}

// Valid reports whether the date is a real point in time.
func (d *Date) Valid() bool { return !math.IsNaN(d.Time) }

// daysFromCivil is the number of days between the epoch and the given date, for any year.
func daysFromCivil(year, month, day int) int {
	y := year
	if month <= 2 {
		y--
	}
	era := y / 400
	if y < 0 && y%400 != 0 {
		era--
	}
	yoe := y - era*400
	m := month
	if m > 2 {
		m -= 3
	} else {
		m += 9
	}
	doy := (153*m+2)/5 + day - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// civilFromDays is the date a number of days after the epoch falls on.
func civilFromDays(z int) (year, month, day int) {
	z += 719468
	era := z / 146097
	if z < 0 && z%146097 != 0 {
		era--
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if mp >= 10 {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}

// MakeTime builds a time value out of the parts of a date, all of them in universal time.
func MakeTime(year, month, day, hour, minute, second, ms int) float64 {
	year += month / 12
	month %= 12
	if month < 0 {
		month += 12
		year--
	}
	days := daysFromCivil(year, month+1, 1) + day - 1
	return float64(days)*msPerDay + float64(hour)*msPerHour + float64(minute)*msPerMinute +
		float64(second)*msPerSecond + float64(ms)
}

func (d *Date) parts() (year, month, day, hour, minute, second, ms, weekday int) {
	t := int64(d.Time)
	days := int(floorDiv(t, msPerDay))
	rest := int(t - int64(days)*msPerDay)
	year, m, day := civilFromDays(days)
	weekday = int((days%7 + 11) % 7)
	return year, m - 1, day, rest / msPerHour, rest % msPerHour / msPerMinute, rest % msPerMinute / msPerSecond,
		rest % msPerSecond, weekday
}

func floorDiv(a int64, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// UTCFullYear is the year of the date in universal time.
func (d *Date) UTCFullYear() int { y, _, _, _, _, _, _, _ := d.parts(); return y }

// UTCMonth is the month of the date in universal time, counted from zero.
func (d *Date) UTCMonth() int { _, m, _, _, _, _, _, _ := d.parts(); return m }

// UTCDate is the day of the month of the date in universal time.
func (d *Date) UTCDate() int { _, _, day, _, _, _, _, _ := d.parts(); return day }

// FullYear is the year of the date in local time, which is universal time here.
func (d *Date) FullYear() int { return d.UTCFullYear() }

// Month is the month of the date in local time, counted from zero.
func (d *Date) Month() int { return d.UTCMonth() }

// Day is the day of the month of the date in local time.
func (d *Date) Day() int { return d.UTCDate() }

// ToISOString writes the date in the extended format the toISOString method writes.
func (d *Date) ToISOString() string {
	year, month, day, hour, minute, second, ms, _ := d.parts()
	yearText := fmt.Sprintf("%04d", year)
	if year < 0 {
		yearText = fmt.Sprintf("-%06d", -year)
	} else if year > 9999 {
		yearText = fmt.Sprintf("+%06d", year)
	}
	return fmt.Sprintf("%s-%02d-%02dT%02d:%02d:%02d.%03dZ", yearText, month+1, day, hour, minute, second, ms)
}

// String writes the date the way the toString method writes it.
func (d *Date) String() string {
	if !d.Valid() {
		return "Invalid Date"
	}
	year, month, day, hour, minute, second, _, weekday := d.parts()
	return fmt.Sprintf("%s %s %02d %04d %02d:%02d:%02d GMT+0000 (Coordinated Universal Time)",
		dayNames[weekday], monthNames[month], day, year, hour, minute, second)
}
