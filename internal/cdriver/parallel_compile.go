package cdriver

import (
	"context"
	"sync"
	"sync/atomic"
)

// All jobs inherit one deadline. A failure cancels siblings, and all workers
// finish before their caller can remove the owned source/object directory.
func compileParallel(parent context.Context, workers, count int, compile func(context.Context, int) error) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var next atomic.Int64
	var once sync.Once
	var failure error
	var group sync.WaitGroup
	for range min(max(1, workers), count) {
		group.Add(1)
		go func() {
			defer group.Done()
			for ctx.Err() == nil {
				index := int(next.Add(1) - 1)
				if index >= count {
					return
				}
				if err := compile(ctx, index); err != nil {
					once.Do(func() { failure = err; cancel() })
					return
				}
			}
		}()
	}
	group.Wait()
	if failure != nil {
		return failure
	}
	return parent.Err()
}
