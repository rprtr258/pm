// Package queue offers goroutine-safe Queue implementations such as LockfreeQueue(Lock free queue).
// Took from here https://github.com/antigloss/go/blob/master/container/concurrent/queue/lockfree_queue.go
package queue

import (
	"sync/atomic"
	"unsafe"

	"github.com/rprtr258/fun"
)

type node[T any] struct {
	val  T
	next unsafe.Pointer // *node[T]
}

// Queue is a goroutine-safe Queue implementation.
type Queue[T any] struct {
	head  unsafe.Pointer // *node[T]
	tail  unsafe.Pointer // *node[T]
	dummy node[T]
}

// New is the only way to get a new, ready-to-use Queue.
func New[T any]() *Queue[T] {
	var q Queue[T]
	q.head = unsafe.Pointer(&q.dummy)
	q.tail = q.head
	return &q
}

// Pop returns (and removes) an element from the front of the queue and true if the queue is not empty,
// otherwise it returns a default value and false if the queue is empty.
func (q *Queue[T]) Pop() (T, bool) {
	for {
		headPtr := atomic.LoadPointer(&q.head)
		head := (*node[T])(headPtr)
		if next := (*node[T])(atomic.LoadPointer(&head.next)); next != nil {
			if atomic.CompareAndSwapPointer(&q.head, headPtr, head.next) {
				return next.val, true
			}

			continue
		}

		return fun.Zero[T](), false
	}
}

// Push inserts an element to the back of the queue.
func (q *Queue[T]) Push(val T) {
	newNode := unsafe.Pointer(&node[T]{val: val})
	for {
		tail := (*node[T])(atomic.LoadPointer(&q.tail))
		if atomic.CompareAndSwapPointer(&tail.next, nil, newNode) {
			atomic.StorePointer(&q.tail, newNode)
			return
		}
	}
}
