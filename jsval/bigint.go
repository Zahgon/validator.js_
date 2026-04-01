// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package jsval

import (
	"math/big"
	"strings"

	"github.com/validatorjs/validator/jsstr"
)

// BigInt is an integer of any size, the kind the runtime writes with a trailing n. The library never makes one,
// but it is handed them, and what it answers for one follows from its type.
type BigInt struct {
	// Int is the value.
	Int *big.Int
}

// NewBigInt reads a value the way the BigInt function does: a string of digits, in decimal or with a base
// prefix, with space around it allowed and the empty string standing for zero.
func NewBigInt(v Value) (*BigInt, error) {
	switch t := v.(type) {
	case jsstr.String:
		text := strings.TrimSpace(t.Go())
		if text == "" {
			return &BigInt{Int: big.NewInt(0)}, nil
		}
		base := 10
		body := text
		if len(body) > 2 {
			switch strings.ToLower(body[:2]) {
			case "0x":
				base, body = 16, body[2:]
			case "0o":
				base, body = 8, body[2:]
			case "0b":
				base, body = 2, body[2:]
			}
		}
		value, ok := new(big.Int).SetString(body, base)
		if !ok {
			return nil, &Thrown{Name: "SyntaxError", Message: "Cannot convert " + t.Go() + " to a BigInt"}
		}
		return &BigInt{Int: value}, nil
	case float64:
		if t != float64(int64(t)) {
			return nil, &Thrown{Name: "RangeError",
				Message: "The number " + jsstr.NumberToString(t) + " cannot be converted to a BigInt because it is not an integer"}
		}
		return &BigInt{Int: big.NewInt(int64(t))}, nil
	case bool:
		if t {
			return &BigInt{Int: big.NewInt(1)}, nil
		}
		return &BigInt{Int: big.NewInt(0)}, nil
	}
	return nil, &Thrown{Name: "TypeError", Message: "Cannot convert " + TypeOf(v) + " to a BigInt"}
}

// Text writes the integer the way the runtime writes it, without the trailing n.
func (b *BigInt) Text() string { return b.Int.String() }
