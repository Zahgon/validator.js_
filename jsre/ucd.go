// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsre

import "sync"

// The property escapes of the dialect, \p{…} and \P{…}, name sets of code points that the standard library's
// unicode package cannot supply: it has no table for the script extensions of a code point, none for most of
// the binary properties (Alphabetic, Cased, the Changes_When_… family, the Emoji family, Grapheme_Base,
// ID_Start and their companions), and the tables it does have are of a different version of the standard than
// the dialect's. The sets of the dialect are therefore kept here in ucdtable.go, which records, for every name
// the dialect accepts and in the exact spelling it accepts, the code points that name matches.
//
// A set is stored as a string: for each range in order, the gap between the previous range and this one, then
// the length of this one, each as a number of five bits a character with the sixth bit set while the number
// goes on. decodeSet turns that back into ranges the character classes use, once per set.

const ucdAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var (
	ucdOnce  sync.Once
	ucdDigit [128]int8
	ucdCache [len(ucdSets)][]crange
	ucdMutex sync.Mutex
)

func ucdInit() {
	for i := range ucdDigit {
		ucdDigit[i] = -1
	}
	for i, c := range ucdAlphabet {
		ucdDigit[c] = int8(i)
	}
}

// decodeSet reads one stored set into ranges, keeping the result for the next caller.
func decodeSet(index int) []crange {
	ucdOnce.Do(ucdInit)
	ucdMutex.Lock()
	defer ucdMutex.Unlock()
	if ranges := ucdCache[index]; ranges != nil {
		return ranges
	}
	text := ucdSets[index]
	ranges := []crange{}
	prev := -1
	for pos := 0; pos < len(text); {
		numbers := [2]int{}
		for which := range numbers {
			value, shift := 0, 0
			for pos < len(text) {
				digit := int(ucdDigit[text[pos]])
				pos++
				value |= (digit & 31) << shift
				shift += 5
				if digit < 32 {
					break
				}
			}
			numbers[which] = value
		}
		lo := prev + 1 + numbers[0]
		hi := lo + numbers[1]
		ranges = append(ranges, crange{rune(lo), rune(hi)})
		prev = hi
	}
	if len(ranges) == 0 {
		ranges = []crange{}
	}
	ucdCache[index] = ranges
	return ranges
}

// ucdProperty is the set of code points a property name matches, and whether the dialect knows the name at all.
func ucdProperty(name string) ([]crange, bool) {
	index, ok := ucdNames[name]
	if !ok {
		return nil, false
	}
	return decodeSet(index), true
}
