package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"unicode/utf8"
)

var (
	commands map[rune]func(*program)
	Pbuf     = make([]byte, 4)
)

func init() {
	commands = map[rune]func(*program){

		// Quit
		'q': func(p *program) {
			// if this means the depth is -1, that means the program
			// will exit
			targetDepth = 2
		},

		'Q': func(p *program) {
			delta := int(stack.PopNumber())

			// Q will never exit the program
			if delta > macroDepth {
				targetDepth = macroDepth - 1
			} else {
				targetDepth = delta
			}
		},

		// Stack control

		// Clear stack
		'c': func(p *program) {
			stack.Clear()
		},

		// Duplicate value on top of stack
		'd': func(p *program) {
			stack.Push(stack.Peek())
		},

		// Reverse top two values on the stack
		'r': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
				return
			}

			a := stack.Pop()
			b := stack.Pop()

			stack.Push(a)
			stack.Push(b)
		},

		// Push the stack size
		'z': func(p *program) {
			depth := int64(stack.Len())
			stack.Push(intNumber(depth))
		},

		// Parameters

		// Pop a number and use it to set the precision
		'k': func(p *program) {
			num := stack.PopNumber()

			if num < 0 {
				p.error("cannot set a negative precision")
			} else {
				precision = int(num)
			}
		},

		// Push the current precision onto the stack
		'K': func(p *program) {
			stack.Push(intNumber(int64(precision)))
		},

		'o': func(p *program) {
			panic("set output radix")
		},

		'O': func(p *program) {
			panic("load output radix")
		},

		// Registers

		'l': func(p *program) {
			ch := p.next()
			reg, ok := registers[ch]

			if !ok {
				p.errorf("register '%c' is empty", ch)
			} else {
				stack.Push(reg.Peek())
			}
		},

		'L': func(p *program) {
			ch := p.next()
			reg, ok := registers[ch]

			if !ok || reg.Len() == 0 {
				p.errorf("stack register '%c' is empty", ch)
			} else {
				stack.Push(reg.Pop())
			}
		},

		's': func(p *program) {
			ch := p.next()
			reg, ok := registers[ch]
			if !ok {
				reg = newStack()
				registers[ch] = reg
			}

			reg.Set(stack.Pop())
		},

		'S': func(p *program) {
			ch := p.next()
			reg, ok := registers[ch]
			if !ok {
				reg = newStack()
				registers[ch] = reg
			}

			reg.Push(stack.Pop())
		},

		// Printing

		// Pop a value and print it to stderr
		'e': func(p *program) {
			fmt.Fprintln(os.Stderr, stack.Pop())
		},

		// Show stack without modifying it
		'f': func(p *program) {
			stack.Show()
		},

		'n': func(p *program) {
			fmt.Print(stack.Pop())
		},

		// Print the value on top of the stack without modifying it
		'p': func(p *program) {
			fmt.Println(stack.Peek())
		},

		'P': func(p *program) {
			switch v := stack.Pop().(type) {
			case String:
				fmt.Print(v)
			case Number:
				n := uint32(v)
				sz := 0
				switch {
				case n > 1<<24-1:
					sz = 4
				case n > 1<<16-1:
					sz = 3
				case n > 1<<8-1:
					sz = 2
				default:
					sz = 1
				}
				binary.LittleEndian.PutUint32(Pbuf, n)
				os.Stdout.Write(Pbuf[:sz])
			}
		},

		// Arithmetic

		// Pop two values, add them, and push the result
		'+': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				rhs := stack.PopNumber()
				lhs := stack.PopNumber()

				stack.Push(lhs + rhs)
			}
		},

		// Pop two values, subtract the top from the second-to-top, and push the
		// result
		'-': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				rhs := stack.PopNumber()
				lhs := stack.PopNumber()

				stack.Push(lhs - rhs)
			}
		},

		// Pop two values, multiply them, and push the result
		'*': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				rhs := stack.PopNumber()
				lhs := stack.PopNumber()

				stack.Push(lhs * rhs)
			}
		},

		// Pop two values, divide the second-to-top by the top, and push the
		// result
		'/': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				rhs := stack.PopNumber()
				lhs := stack.PopNumber()

				stack.Push(lhs / rhs)
			}
		},

		// Pop two values and push the remainder of the second divided by the
		// first
		'%': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				rhs := stack.PopNumber()
				lhs := stack.PopNumber()

				lhs.Mod(lhs, rhs)
				stack.Push(lhs)
			}
		},

		'~': func(p *program) {
			panic("divmod")

		},

		'^': func(p *program) {
			pow := stack.PopNumber()
			base := stack.PopNumber()
			stack.Push(Number(math.Pow(float64(base), float64(pow))))
		},

		'|': func(p *program) {
			panic("modexp")
		},

		'v': func(p *program) {
			num := stack.PopNumber()
			stack.Push(Number(math.Sqrt(float64(num))))
		},

		// Strings and macros

		// Pop a Unicode codepoint and push the string representation
		'a': func(p *program) {
			var ch rune

			switch v := stack.Pop().(type) {
			case String:
				ch, _ = utf8.DecodeRuneInString(string(v))
			case Number:
				ch = rune(v)
			}

			stack.Push(String(string(ch)))
		},

		// Pop and execute as macro
		'x': func(p *program) {
			d := stack.Pop()

			if macro, ok := d.(String); ok {
				macroDepth++
				evalString([]byte(macro))
				macroDepth--
			} else {
				stack.Push(d)
			}
		},

		'>': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				lhs := stack.PopNumber()
				rhs := stack.PopNumber()
				ch := p.next()

				if lhs.Cmp(rhs) > 0 {
					execMacro(ch)
				}
			}
		},

		'<': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				lhs := stack.PopNumber()
				rhs := stack.PopNumber()
				ch := p.next()

				if lhs.Cmp(rhs) < 0 {
					execMacro(ch)
				}
			}
		},

		'=': func(p *program) {
			if stack.Len() < 2 {
				p.errorf(errNotEnoughStack, 2)
			} else {
				lhs := stack.PopNumber()
				rhs := stack.PopNumber()
				ch := p.next()

				if lhs.Cmp(rhs) == 0 {
					execMacro(ch)
				}
			}
		},

		// misc

		'Z': func(p *program) {
			stack.Push(intNumber(int64(stack.Pop().Len())))
		},

		'X': func(p *program) {
			switch v := stack.Pop().(type) {
			case String:
				stack.Push(Number(0))
			case Number:
				stack.Push(v)
				panic("figure out the fractional digits")
			}
		},

		'#': func(p *program) {
			p.finishLine()
		},

		'?': func(p *program) {
			// since this is generally for user input the buffer shouldn't
			// need to be that big
			buf := bufio.NewReaderSize(os.Stdin, 64)
			line, err := buf.ReadBytes('\n')
			if err != nil && err != io.EOF {
				p.error(err)
			} else {
				evalString(line)
			}
		},

		'!': func(p *program) {
			ch := p.next()
			switch ch {
			case '<', '>', '=':
				if stack.Len() < 2 {
					p.errorf(errNotEnoughStack, 2)
					return
				}
				cmp := 0
				switch ch {
				case '<':
					cmp = -1
				case '>':
					cmp = 1
				}

				lhs := stack.PopNumber()
				rhs := stack.PopNumber()
				ch := p.next()

				if ch == eof {
					return
				}

				if lhs.Cmp(rhs) != cmp {
					execMacro(ch)
				}

			default:
				line := string(p.finishLine())
				cmd := exec.Command("sh", "-c", line)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					p.error(err)
				}
			}
		},

		':': func(p *program) {
			ch := p.next()
			if ch == eof {
				return
			}

			reg, ok := registers[ch]
			if !ok {
				reg = newStack()
				registers[ch] = reg
			}

			index := stack.PopNumber()
			value := stack.Pop()
			reg.ArraySet(index, value)
		},

		';': func(p *program) {
			ch := p.next()
			if ch == eof {
				return
			}

			reg, ok := registers[ch]
			if ok {
				index := stack.PopNumber()
				stack.Push(reg.ArrayGet(index))
			} else {
				stack.Push(Number(0))
			}
		},
	}
}

func execMacro(ch rune) {
	reg, ok := registers[ch]
	if !ok {
		return
	}
	d := reg.Peek()
	if macro, ok := d.(String); ok {
		macroDepth++
		evalString([]byte(macro))
		macroDepth--
	}
}
