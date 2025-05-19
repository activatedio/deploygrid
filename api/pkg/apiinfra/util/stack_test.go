package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStack(t *testing.T) {

	a := assert.New(t)

	s1 := "1"
	s2 := "2"
	s3 := "3"
	s4 := "4"

	unit := NewStack[string]()

	a.True(unit.Empty())
	a.Equal(0, unit.Size())

	unit.Push(&s1)

	a.False(unit.Empty())
	a.Equal(1, unit.Size())

	unit.Push(&s2)
	unit.Push(&s3)

	a.False(unit.Empty())
	a.Equal(3, unit.Size())

	a.Equal(&s3, unit.Pop())
	a.Equal(2, unit.Size())
	a.Equal(&s2, unit.Pop())
	a.Equal(1, unit.Size())
	a.Equal(&s1, unit.Pop())
	a.Equal(0, unit.Size())

	a.True(unit.Empty())

	a.Nil(unit.Pop())

	unit.Push(&s1)

	a.Equal(&s1, unit.Pop())

	unit.Push(&s2)
	unit.Push(&s3)

	a.Equal(&s3, unit.Pop())
	a.Equal(&s2, unit.Pop())

	unit.Push(&s4)

	a.Equal(&s4, unit.Pop())
}
