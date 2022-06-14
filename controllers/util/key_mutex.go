// Taken and modified from https://github.com/containerd/containerd/blob/10c12954828e7c7c9b6e0ea9b0c02b01407d3ae1/pkg/kmutex/kmutex.go
package util

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/semaphore"
)

// KeyedLocker is the interface for acquiring locks based on string.
type KeyedLocker interface {
	Lock(ctx context.Context, key string) error
	TryLock(key string) bool
	Unlock(key string)
}

func NewKeyedLocker() KeyedLocker {
	return newKeyMutex()
}

func newKeyMutex() *keyMutex {
	return &keyMutex{
		locks: make(map[string]*klock),
	}
}

type keyMutex struct {
	mu sync.Mutex

	locks map[string]*klock
}

type klock struct {
	*semaphore.Weighted
	ref int
}

func (km *keyMutex) Lock(ctx context.Context, key string) error {
	km.mu.Lock()

	l, ok := km.locks[key]
	if !ok {
		km.locks[key] = &klock{
			Weighted: semaphore.NewWeighted(1),
		}
		l = km.locks[key]
	}
	l.ref++
	km.mu.Unlock()

	if err := l.Acquire(ctx, 1); err != nil {
		km.mu.Lock()
		defer km.mu.Unlock()

		l.ref--

		if l.ref < 0 {
			panic(fmt.Errorf("kmutex: release of unlocked key %v", key))
		}

		if l.ref == 0 {
			delete(km.locks, key)
		}
		return err
	}
	return nil
}

func (km *keyMutex) TryLock(key string) bool {
	km.mu.Lock()

	l, ok := km.locks[key]
	if !ok {
		km.locks[key] = &klock{
			Weighted: semaphore.NewWeighted(1),
		}
		l = km.locks[key]
	}
	l.ref++
	km.mu.Unlock()

	gotLock := l.TryAcquire(1)
	if !gotLock {
		km.mu.Lock()
		defer km.mu.Unlock()

		l.ref--

		if l.ref < 0 {
			panic(fmt.Errorf("kmutex: release of unlocked key %v", key))
		}

		if l.ref == 0 {
			delete(km.locks, key)
		}
	}

	return gotLock
}

func (km *keyMutex) Unlock(key string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	l, ok := km.locks[key]
	if !ok {
		panic(fmt.Errorf("kmutex: unlock of unlocked key %v", key))
	}
	l.Release(1)

	l.ref--

	if l.ref < 0 {
		panic(fmt.Errorf("kmutex: released of unlocked key %v", key))
	}

	if l.ref == 0 {
		delete(km.locks, key)
	}
}
