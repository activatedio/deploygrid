package util

import (
	"sync"
)

// FROM https://stackoverflow.com/a/28542256/19580930

type Stack[E any] struct {
	lock *sync.Mutex // you don't have to do this if you don't want thread safety
	// Made public so json can serialize
	S []*E
}

func NewStack[E any]() *Stack[E] {
	return &Stack[E]{&sync.Mutex{}, make([]*E, 0)}
}

func (s *Stack[E]) Push(v *E) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.S = append(s.S, v)
}

func (s *Stack[E]) Empty() bool {
	return len(s.S) == 0
}

func (s *Stack[E]) Size() int {
	return len(s.S)
}

func (s *Stack[E]) Pop() *E {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := len(s.S)
	if l == 0 {
		return nil
	}

	res := s.S[l-1]
	s.S = s.S[:l-1]
	return res
}
