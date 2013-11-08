I liked `dc` so much that I tried reimplementing it in Go. Sadly I can't say this was another evening's hack—it took me almost a week to get it to this semi-working state. But here it is.

It was meant be mostly backwards-compatible with GNU `dc` with some notable changes and incompletenesses:

- The input is interpreted as UTF-8. The most obvious consequence of this is that you are no longer limited to 256 registers. You can use any Unicode codepoint for registers (that's a lot).
- It uses simple `float64`s instead of real arbitrary precision numbers (yeah, good enough for me, for now). I started out using `big.Rat` from the standard package `math/big` but that turned out to be far too slow and doesn't work with `dc`'s idea of "arbitrary precision". The precision with `big.Rat` is already "infinite" in all cases, which makes for easier dealing with but (constantly) slow arithmetic. Also, `dc` (`bc`) numbers are something like decimal numbers each with individual scales, which means printing a number will always print out its full precision. Not so with rationals. This needs a custom number implementation that I'm not feeling up to quite yet.
- `e`: Pop a value and print it to stderr with a new line. Nice for debugging, kinda.

My "good enough" point was using `dcx` to [generate a Mandelbrot set](https://github.com/moshee/mandel.dc). It runs at about half the speed of `dc` for the same parameters, but the output is identical, so at least that's a win.

```
⚡ time dc mandel.min.dc > test1.pgm <<<'32 256 384'

real    0m7.011s
user    0m6.988s
sys     0m0.024s
⚡ time ./dcx mandel.min.dc > test2.pgm <<<'32 256 384'

real    0m15.745s
user    0m14.169s
sys     0m2.352s
⚡ diff *.pgm
⚡
```

#### TODO

- reimplement number (`math/big` is kind of worthless for this)
  - `i`, `o`, `I`, `O`
  - `~`, `|` (not in `math`)
- `Z`
- `X`
- when I'm feeling adventurous this is a nice project to study optimization
- special functions (sin, cos, etc.)?
