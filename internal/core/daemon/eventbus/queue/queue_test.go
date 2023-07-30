package queue

import (
	"sort"
	"sync"
	"testing"
)

const (
	_goroutines = 10
	_pushingNum = 500000
	_bufferSize = _goroutines * _pushingNum
)

var q = New[int]()

func TestQueue(t *testing.T) {
	var popBuf [_goroutines][]int
	// init popBuf
	for i := 0; i != _goroutines; i++ {
		popBuf[i] = make([]int, 0, _bufferSize)
	}

	var wg sync.WaitGroup
	// Push() simultaneously
	wg.Add(_goroutines)
	for i := 0; i != _goroutines; i++ {
		go func() {
			push()
			wg.Done()
		}()
	}
	wg.Wait()
	// Pop() simultaneously
	wg.Add(_goroutines)
	for i := 0; i != _goroutines; i++ {
		go func() {
			for i := 0; i != _pushingNum; i++ {
				_, ok := q.Pop()
				if !ok {
					t.Error("Should never be nil!")
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// Push() and Pop() simultaneously
	wg.Add(_goroutines * 2)
	for i := 0; i < _goroutines; i++ {
		go func() {
			push()
			wg.Done()
		}()
		go func(n int) { // pop while pushing
			for j := 0; j < _pushingNum*2; j++ {
				v, ok := q.Pop()
				if ok {
					popBuf[n] = append(popBuf[n], v)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	// Verification
	resultBuf := popBuf[0]
	for i := 1; i != _goroutines; i++ {
		resultBuf = append(resultBuf, popBuf[i]...)
	}
	// in case there are some elements left in the queue
	for v, ok := q.Pop(); ok; v, ok = q.Pop() {
		resultBuf = append(resultBuf, v)
	}
	sort.Ints(resultBuf)
	for i := 0; i != _pushingNum; i++ {
		for j := 0; j != _goroutines; j++ {
			if resultBuf[(i*_goroutines)+j] != i {
				t.Error("Invalid result:", i, j, resultBuf[(i*_goroutines)+j])
			}
		}
	}
}

func push() {
	for i := 0; i < _pushingNum; i++ {
		q.Push(i)
	}
}
