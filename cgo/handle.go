package main

import (
	"sync"
	"sync/atomic"
)

// Handle is an opaque pointer to a Go object, safe to pass across the CGO boundary.
// Handles are uint64 values that can be cast to/from C pointers.
type Handle uint64

// handleTable stores Go objects and maps them to opaque handles.
// This allows safe passage of Go objects across the CGO boundary.
type handleTable struct {
	counter uint64
	mu      sync.RWMutex
	objects map[Handle]interface{}
}

var handles = &handleTable{
	objects: make(map[Handle]interface{}),
}

// NewHandle stores an object and returns a handle to it.
// The handle can be safely passed to C code.
func NewHandle(obj interface{}) Handle {
	id := Handle(atomic.AddUint64(&handles.counter, 1))
	handles.mu.Lock()
	handles.objects[id] = obj
	handles.mu.Unlock()
	return id
}

// Get retrieves the object associated with a handle.
// Returns nil if the handle is invalid or has been deleted.
func (h Handle) Get() interface{} {
	handles.mu.RLock()
	obj := handles.objects[h]
	handles.mu.RUnlock()
	return obj
}

// Delete removes the object associated with a handle.
// After deletion, the handle is no longer valid.
func (h Handle) Delete() {
	handles.mu.Lock()
	delete(handles.objects, h)
	handles.mu.Unlock()
}

// Valid returns true if the handle is associated with an object.
func (h Handle) Valid() bool {
	handles.mu.RLock()
	_, exists := handles.objects[h]
	handles.mu.RUnlock()
	return exists
}

// TypedHandle provides type-safe handle operations.
type TypedHandle[T any] Handle

// NewTypedHandle creates a new handle for a typed object.
func NewTypedHandle[T any](obj T) TypedHandle[T] {
	return TypedHandle[T](NewHandle(obj))
}

// Get retrieves the typed object, returning the zero value if not found or wrong type.
func (h TypedHandle[T]) Get() (T, bool) {
	obj := Handle(h).Get()
	if obj == nil {
		var zero T
		return zero, false
	}
	typed, ok := obj.(T)
	return typed, ok
}

// Delete removes the typed handle.
func (h TypedHandle[T]) Delete() {
	Handle(h).Delete()
}

// Handle returns the underlying Handle value.
func (h TypedHandle[T]) Handle() Handle {
	return Handle(h)
}
