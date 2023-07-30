package queue

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	_goroutines = 10
	_pushingNum = 500000
	_bufferSize = _goroutines * _pushingNum
)

func TestManyPushPop(t *testing.T) {
	t.Parallel()

	cnt := [_pushingNum]int{}
	q := New[int]()

	var wg sync.WaitGroup
	// Push() and Pop() simultaneously
	wg.Add(_goroutines * 2)
	for i := 0; i < _goroutines; i++ {
		go func() {
			push(q)
			wg.Done()
		}()
		go func() {
			for j := 0; j < _pushingNum*2; j++ {
				if v, ok := q.Pop(); ok {
					cnt[v]++
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// in case there are some elements left in the queue
	for v, ok := q.Pop(); ok; v, ok = q.Pop() {
		cnt[v]++
	}

	for _, x := range cnt {
		assert.Equal(t, _goroutines, x)
	}
}

func TestManyPushManyPop(t *testing.T) {
	t.Parallel()

	q := New[int]()

	var wg sync.WaitGroup
	// Push() simultaneously
	wg.Add(_goroutines)
	for i := 0; i < _goroutines; i++ {
		go func() {
			push(q)
			wg.Done()
		}()
	}
	wg.Wait()

	// Pop() simultaneously
	wg.Add(_goroutines)
	for i := 0; i < _goroutines; i++ {
		go func() {
			for i := 0; i < _pushingNum; i++ {
				_, ok := q.Pop()
				assert.True(t, ok)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func push(q *Queue[int]) {
	for i := 0; i < _pushingNum; i++ {
		q.Push(i)
	}
}
