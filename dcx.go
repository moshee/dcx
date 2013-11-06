package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	//"math/big"
	"os"
	"path/filepath"
	//"runtime/debug"
	"strconv"
	"unicode"
	//"unicode/utf8"
)

const VERSION = "α"

var (
	PROG_NAME = filepath.Base(os.Args[0])

	// Global states
	stack     *Stack
	registers = make(map[rune]*Stack)
	precision = 0
	//input      = new(reader)
	prog        = new(program)
	macroDepth  int
	targetDepth = -1

	flag_file = flag.String("f", "", "Evaluate file and then stay running")
	flag_expr = flag.String("e", "", "Evaluate string")
	flag_v    = flag.Bool("v", false, "Print version info and exit")
)

func init() {
	stack = newStack()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION] [file [...]]\n", PROG_NAME)
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *flag_v {
		fmt.Fprintf(os.Stderr, "dcx %s\n", VERSION)
		return
	}

	if len(*flag_expr) > 0 {
		evalString([]byte(*flag_expr))
		return
	}
	if len(*flag_file) > 0 {
		evalFile(*flag_file)
	}

	if flag.NArg() > 0 {
		for _, file := range flag.Args() {
			evalFile(file)
		}
		return
	}

	//input.readFrom(os.Stdin)
	prog.init(os.Stdin)
	prog.eval()
}

func evalFile(name string) {
	file, err := os.Open(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	prog.init(file)
	prog.eval()
}

func evalString(expr []byte) {
	//fmt.Printf("<<%v>>\n", string(expr))
	p := new(program)
	p.depth = macroDepth
	p.init(bytes.NewReader(expr))
	p.eval()
}

/*
func evalFile(name string) {
	file, err := os.Open(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	input.readFrom(file)
	input.eval()
}

func evalString(expr []byte) {
	stringInput := &reader{
		scanner: bufio.NewScanner(bytes.NewReader(expr)),
		depth:   macroDepth,
	}
	stringInput.eval()
}
*/

const eof = -1

type program struct {
	r       *bufio.Reader
	col     int
	line    int
	lastCol int
	depth   int
}

func (self *program) init(r io.Reader) {
	self.r = bufio.NewReader(r)
	self.col = 0
	self.line = 1
	self.depth = macroDepth
}

func (self *program) next() rune {
	ch, _, err := self.r.ReadRune()
	if err != nil {
		if err == io.EOF {
			return eof
		}
		panic(err)
	}
	if ch == '\n' {
		self.lastCol = self.col
		self.nextLine()
	} else {
		self.col++
	}
	return ch
}

func (self *program) nextByte() (byte, error) {
	ch, err := self.r.ReadByte()
	if ch == '\n' {
		self.nextLine()
	}

	return ch, err
}

func (self *program) back() {
	err := self.r.UnreadRune()
	if err != nil {
		panic(err)
	}

	if self.col <= 1 {
		self.col = self.lastCol
		self.lastCol = -1
	} else {
		self.col--
	}
}

func (self *program) finishLine() []byte {
	line := make([]byte, 0, 256)

	for {
		ch, err := self.nextByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			panic(err)
		}

		line = append(line, ch)

		if ch == '\n' {
			break
		}
	}

	self.nextLine()

	return line
}

func (self *program) nextLine() {
	self.col = 0
	self.line++
}

func (self *program) error(err interface{}) {
	fmt.Fprintf(os.Stderr, "%s[%d:%d]: %v\n", PROG_NAME, self.line, self.col, err)
}

func (self *program) errorf(format string, a ...interface{}) {
	self.error(fmt.Errorf(format, a...))
}

func (self *program) eval() {
	for !func() bool {
		defer func() {
			if err := recover(); err != nil {
				self.error(err)
				//debug.PrintStack()
				//os.Exit(1)
			}
		}()

	scan:
		for {
			if targetDepth > 0 {
				targetDepth--
				break scan
			}

			ch := self.next()

			switch {
			case ch == eof:
				break scan

			case unicode.IsSpace(ch):
				continue

			case isNumeric(ch):
				self.scanNumeric(ch)

			case ch == '[':
				self.scanString()

			default:
				if cmd, ok := commands[ch]; ok {
					cmd(self)
					continue
				}
				self.error(fmt.Errorf("%c: unimplemented", ch))
			}
		}

		return true
	}() {
		// just keep looping
	}
}

func isNumeric(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ch == '_' || ch == '.'
}

func (self *program) scanNumeric(ch rune) {
	// place the number in here.
	buf := make([]rune, 0, 16)
	seenDot := false

	switch ch {
	case '_':
		buf = append(buf, '-')
	case '.':
		seenDot = true
		fallthrough
	default:
		buf = append(buf, ch)
	}

scan:
	for {
		ch = self.next()
		switch ch {
		case eof:
			break scan
		case '.':
			if seenDot {
				self.back()
				break scan
			}
			seenDot = true
		case '_':
			self.back()
			break scan
		default:
			if !isNumeric(ch) {
				self.back()
				break scan
			}
		}

		buf = append(buf, ch)
	}

	str := string(buf)
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		self.errorf("failed to parse numeric '%s'", str)
	} else {
		stack.Push(Number(f))
	}
}

func (self *program) scanString() {
	depth := 1
	buf := make([]byte, 0, 1024)

	for {
		ch, err := self.nextByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		switch ch {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 1 {
				stack.Push(String(buf))
				return
			}
		}

		buf = append(buf, ch)
	}
}

/*
type reader struct {
	// A scanner to read lines from.
	scanner *bufio.Scanner

	// The current line
	buf []byte
	ptr int
	// For backing up the read pointer
	lastWidth int

	// the last line, for string reading
	//oldbuf []byte

	// Keeps track of the line index so error messages might be a little more
	// helpful
	line int

	// The execution depth of this (be it a macro or file or whatever).
	// Zero is top-level.
	depth int
}

// Reset the reader with a new source.
func (self *reader) readFrom(r io.Reader) {
	self.scanner = bufio.NewScanner(r)
	self.buf = nil
	self.ptr = 0
}

// Advance to the next rune, swapping the next line of input if necessary.
func (self *reader) next() rune {
	if self.ptr == len(self.buf) {
		// this is needed to signal that a new buffer is being read
		// pretending that the scanner left the \n on
		self.lastWidth = 1
		self.ptr++
		return '\n'
	} else if self.buf == nil || self.ptr > len(self.buf) {
		if self.scanner == nil || !self.scanner.Scan() {
			return eof
		}

		buf := self.scanner.Bytes()
		self.line++

		for len(buf) == 0 {
			if !self.scanner.Scan() {
				return eof
			}
			buf = self.scanner.Bytes()
			self.line++
		}

		//self.oldbuf = self.buf
		self.buf = buf
		self.ptr = 0
	}

	ch, width := utf8.DecodeRune(self.buf[self.ptr:])
	self.lastWidth = width
	self.ptr += width
	return ch
}

func (self *reader) back() {
	self.ptr -= self.lastWidth
}

func (self *reader) finishLine() {
	self.ptr = len(self.buf) + 1
}

func (self *reader) error(err interface{}) {
	fmt.Fprintf(os.Stderr, "%s[%d]: %v\n", PROG_NAME, self.line, err)
}

func (self *reader) errorf(format string, a ...interface{}) {
	self.error(fmt.Errorf(format, a...))
}

// Main eval loop. If any panic occurs, it will be recovered and printed, and
// execution will begin again where it left off. No program state is lost.
func (self *reader) eval() {
	for !func() bool {
		defer func() {
			if err := recover(); err != nil {
				self.error(err)
				//debug.PrintStack()
				//os.Exit(1)
			}
		}()

	scan:
		for {
			if self.depth > macroDepth || macroDepth < 0 {
				break scan
			}
			ch := self.next()

			switch true {
			case ch == eof:
				break scan
			case unicode.IsSpace(ch):
				continue
			case isNumeric(ch):
				self.back()
				scanNumeric(self)
			case ch == '[':
				scanString(self)
			default:
				if cmd, ok := commands[ch]; ok {
					cmd(self)
					continue
				}
				self.error(fmt.Errorf("%c: unimplemented", ch))
			}
		}

		return true
	}() {
		// just keep looping
	}
}

func scanNumeric(r *reader) {
	i := r.ptr
	ch := r.next()
	if ch == '_' {
		r.buf[r.ptr-r.lastWidth] = '-'
	}

	seenDot := false
	reachedEnd := false

scan:
	for {
		ch = r.next()
		switch true {
		case ch == eof:
			reachedEnd = true
			break scan
		case ch == '_':
			break scan
		case ch == '.':
			if seenDot {
				break scan
			}
			seenDot = true
		case !isNumeric(ch):
			break scan
		}
	}

	if !reachedEnd {
		r.back()
	}

	str := string(r.buf[i:r.ptr])

	if rat, ok := new(big.Rat).SetString(str); ok {
		stack.Push(Number{rat})
	} else {
		r.error(fmt.Errorf("error parsing numeric '%s'", str))
	}
}

func scanString(r *reader) {
	depth := 1
	strstart := r.ptr
	// a buffer to hold lines
	strbuf := make([]byte, 0)

	for {
		ch := r.next()
		switch ch {
		case '[':
			depth++
		case ']', eof:
			depth--
			if depth == 0 {
				strbuf = append(strbuf, r.buf[strstart:r.ptr-1]...)
				//r.errorf("pushing %s\n", string(strbuf))
				stack.Push(String(strbuf))
				return
			}
		case '\n':
			// add the whole line to strbuf and add on the newline.
			// since the reader has already advanced to the next line and
			// replaced the internal buffer, we get to use the previous line
			// which it conveniently provides just for this purpose
			strbuf = append(strbuf, r.buf[strstart:]...)
			strbuf = append(strbuf, '\n')
			strstart = 0
			continue
		}
	}
}
*/
