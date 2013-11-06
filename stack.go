package main

import (
	"errors"
	"fmt"
)

var (
	errNotNumber      = errors.New("not a number")
	errStackEmpty     = errors.New("stack empty")
	errNotEnoughStack = "less than %d values on stack"
)

type Stack struct {
	Data []Datum
}

func newStack() *Stack {
	return &Stack{make([]Datum, 0)}
}

func (s *Stack) Push(d Datum) {
	//fmt.Printf("before push %v\n", s.Data)
	s.Data = append(s.Data, d)
	//fmt.Printf("pushed %v (size now %d, %v)\n", d, s.Len(), s.Data)
}

func (s *Stack) Pop() (d Datum) {
	if len(s.Data) == 0 {
		panic(errStackEmpty)
	}

	d = s.Data[len(s.Data)-1]
	s.Data = s.Data[:len(s.Data)-1]
	//fmt.Printf("popped %v (size now %d, %v)\n", d, s.Len(), s.Data)

	return
}

func (s *Stack) PopNumber() (n Number) {
	d := stack.Pop()

	if num, ok := d.(Number); ok {
		return num
	}

	s.Push(d)
	panic(errNotNumber)
}

func (s *Stack) Set(d Datum) {
	if len(s.Data) == 0 {
		s.Push(d)
	} else {
		s.Data[len(s.Data)-1] = d
	}
}

func (s *Stack) Peek() (d Datum) {
	if len(s.Data) == 0 {
		panic(errStackEmpty)
	}

	return s.Data[len(s.Data)-1]
}

func (s *Stack) Len() int {
	return len(s.Data)
}

func (s *Stack) Clear() {
	s.Data = s.Data[:0]
}

func (s *Stack) Show() {
	if s.Len() == 0 {
		return
	}

	for i := len(s.Data) - 1; i >= 0; i-- {
		fmt.Println(s.Data[i])
	}
}
