package main

import (
	"fmt"
	"math"
	//"math/big"
	"unicode/utf8"
)

type Datum interface {
	fmt.Stringer
	Len() int
}

type String []byte

func (s String) String() string { return string(s) }
func (s String) Len() int       { return utf8.RuneCount([]byte(s)) }

type Number float64

func intNumber(i int64) Number {
	return Number(i)
}

func (n Number) String() string {
	return fmt.Sprintf("%g", n)
}

func (n Number) Len() int {
	panic("todo: this")
}

// Rounds the big.Rat to an integer and returns it
func (n Number) Int() int64 {
	return int64(n)
}

func (n Number) Cmp(m Number) int {
	if n < m {
		return -1
	} else if n > m {
		return 1
	}
	return 0
}

// Logic stolen from math/big/rat.go. Probably much slower than it could've
// been if some of those unexported big.Rat methods and stuff were exported. We
// have to sidestep using big.nats by using big.Ints.

/*
func mulDenom(z, x, y *big.Int) *big.Int {
	switch {
	case x.Cmp(zero) == 0:
		return z.Set(y)
	case y.Cmp(zero) == 0:
		return z.Set(x)
	}
	return z.Mul(x, y)
}

func scaleDenom(num, denom *big.Int) *big.Int {
	var z big.Int
	if denom.Cmp(zero) == 0 {
		return z.Set(num)
	}
	z.Set(z.Mul(num, denom))
	return &z
}
*/

// Compute and set n to the remainder of a and b.
func (n Number) Mod(a, b Number) Number {
	return Number(math.Mod(float64(a), float64(b)))
}
