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
	next unsafe.Pointer
}

// Queue is a goroutine-safe Queue implementation.
// The overall performance of Queue is much better than List+Mutex(standard package).
type Queue[T any] struct {
	head  unsafe.Pointer
	tail  unsafe.Pointer
	dummy node[T]
}

// New is the only way to get a new, ready-to-use LockfreeQueue.
func New[T any]() *Queue[T] {
	var q Queue[T]
	q.head = unsafe.Pointer(&q.dummy)
	q.tail = q.head
	return &q
}

// Pop returns (and removes) an element from the front of the queue and true if the queue is not empty,
// otherwise it returns a default value and false if the queue is empty.
// It performs about 100% better than list.List.Front() and list.List.Remove() with sync.Mutex.
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
// It performs exactly the same as list.List.PushBack() with sync.Mutex.
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
