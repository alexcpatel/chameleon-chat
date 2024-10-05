package history

import (
	"errors"
	"sync"
)

// CircularBuffer is a generic circular buffer implementation
type CircularBuffer[T any] struct {
	buffer []T
	size   int
	head   int
	tail   int
	count  int
	mutex  sync.Mutex
}

// NewCircularBuffer creates a new circular buffer with the given size
func NewCircularBuffer[T any](size int) *CircularBuffer[T] {
	return &CircularBuffer[T]{
		buffer: make([]T, size),
		size:   size,
	}
}

// Push adds an item to the buffer, overwriting the oldest item if the buffer is full
func (cb *CircularBuffer[T]) Push(item T) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.count == cb.size {
		cb.head = (cb.head + 1) % cb.size
	} else {
		cb.count++
	}
	cb.buffer[cb.tail] = item
	cb.tail = (cb.tail + 1) % cb.size
}

// Pop removes and returns the oldest item from the buffer
func (cb *CircularBuffer[T]) Pop() (T, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.count == 0 {
		var zero T
		return zero, errors.New("buffer is empty")
	}
	item := cb.buffer[cb.head]
	cb.head = (cb.head + 1) % cb.size
	cb.count--
	return item, nil
}

// Peek returns the oldest item without removing it
func (cb *CircularBuffer[T]) Peek() (T, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.count == 0 {
		var zero T
		return zero, errors.New("buffer is empty")
	}
	return cb.buffer[cb.head], nil
}

// IsFull returns true if the buffer is full
func (cb *CircularBuffer[T]) IsFull() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.count == cb.size
}

// IsEmpty returns true if the buffer is empty
func (cb *CircularBuffer[T]) IsEmpty() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.count == 0
}

// Count returns the number of items in the buffer
func (cb *CircularBuffer[T]) Count() int {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.count
}

// Clear empties the buffer
func (cb *CircularBuffer[T]) Clear() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.head = 0
	cb.tail = 0
	cb.count = 0
}

// LastN returns a slice of the last n items in the buffer, from oldest to newest
// If n is greater than the number of items in the buffer, it returns all items
func (cb *CircularBuffer[T]) LastN(n int) []T {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.count == 0 {
		return []T{}
	}

	// Determine the number of items to return
	itemsToReturn := n
	if itemsToReturn > cb.count {
		itemsToReturn = cb.count
	}

	result := make([]T, itemsToReturn)
	startIndex := (cb.tail - itemsToReturn + cb.size) % cb.size

	for i := 0; i < itemsToReturn; i++ {
		index := (startIndex + i) % cb.size
		result[i] = cb.buffer[index]
	}
	return result
}
