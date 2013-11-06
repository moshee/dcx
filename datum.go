package main

import (
	"fmt"
	"math/big"
	"unicode/utf8"
)

type Datum interface {
	fmt.Stringer
	Len() int
}

type String []byte

func (s String) String() string { return string(s) }
func (s String) Len() int       { return utf8.RuneCount([]byte(s)) }

var (
	zero = big.NewInt(0)
	one  = big.NewInt(1)
)

type Number struct {
	*big.Rat
}

func intNumber(i int64) Number {
	return Number{big.NewRat(i, 1)}
}

func (n Number) String() string {
	return n.Rat.FloatString(precision)
}

func (n Number) Len() int {
	panic("todo: this")
}

// Rounds the big.Rat to an integer and returns it
func (n Number) Int() int64 {
	num := n.Num()

	if n.IsInt() {
		return num.Int64()
	}

	denom := n.Denom()

	return num.Div(num, denom).Int64()
}

func (n Number) Cmp(m Number) int {
	return n.Rat.Cmp(m.Rat)
}

// Logic stolen from math/big/rat.go. Probably much slower than it could've
// been if some of those unexported big.Rat methods and stuff were exported. We
// have to sidestep using big.nats by using big.Ints.

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

// Compute and set n to the remainder of a and b.
func (n Number) Mod(a, b Number) Number {
	if a.IsInt() && b.IsInt() {
		aNum, bNum := a.Num(), b.Num()
		n.Rat.SetInt(aNum.Mod(aNum, bNum))
	} else {
		if b.Num().Cmp(zero) == 0 {
			panic("remainder by zero")
		}
		a1 := scaleDenom(a.Num(), b.Denom())
		b1 := scaleDenom(b.Num(), a.Denom())

		num := new(big.Int).Mod(a1, b1)
		denom := mulDenom(n.Denom(), a.Denom(), b.Denom())

		n.Rat = a.SetFrac(num, denom)
	}

	return n
}
