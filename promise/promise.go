package promise

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type Waiter interface {
	Wait() error
}

type Promise[V any] struct {
	val  V
	err  error
	done <-chan struct{}
}

type MapFunc[T, V any] func(T) V

type Func[T, V any] func(context.Context, T) (V, error)

type Func2[T, K, V any] func(context.Context, T, K) (V, error)

func NewPromise[T, V any](ctx context.Context, t T, f Func[T, V]) *Promise[V] {
	done := make(chan struct{})
	p := Promise[V]{
		done: done,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.err = fmt.Errorf("recovered: %#v, stacktrace: %s", r, string(debug.Stack()))
			}
			close(done)
		}()
		p.val, p.err = f(ctx, t)
	}()
	return &p
}

func (p *Promise[V]) Get() (V, error) {
	<-p.done
	return p.val, p.err
}

func (p *Promise[V]) Wait() error {
	<-p.done
	return p.err
}

func Wait(ws ...Waiter) error {
	var wg sync.WaitGroup
	wg.Add(len(ws))
	errChan := make(chan error, len(ws))
	done := make(chan struct{})
	for _, w := range ws {
		go func(w Waiter) {
			defer wg.Done()
			err := w.Wait()
			if err != nil {
				errChan <- err
			}
		}(w)
	}
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case err := <-errChan:
		return err
	case <-done:
	}
	return nil
}

func WithCancellation[T, V any](f Func[T, V]) Func[T, V] {
	return func(ctx context.Context, t T) (V, error) {
		done := make(chan struct{})
		var val V
		var err error
		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("recovered: %#v, stacktrace: %s", r, string(debug.Stack()))
				}
				close(done)
			}()
			val, err = f(ctx, t)
		}()
		select {
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		case <-done:
		}
		return val, err
	}
}

func MapAndThen[T, V, K any](ctx context.Context, p *Promise[T], mf MapFunc[T, K], f Func[K, V]) *Promise[V] {
	done := make(chan struct{})
	out := Promise[V]{
		done: done,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.err = fmt.Errorf("recovered: %#v, stacktrace: %s", r, string(debug.Stack()))
			}
			close(done)
		}()
		val, err := p.Get()
		if err != nil {
			out.err = err
			return
		}
		transVal := mf(val)
		out.val, out.err = f(ctx, transVal)
	}()
	return &out
}

func Then[T, V any](ctx context.Context, p *Promise[T], f Func[T, V]) *Promise[V] {
	done := make(chan struct{})
	out := Promise[V]{
		done: done,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.err = fmt.Errorf("recovered: %#v, stacktrace: %s", r, string(debug.Stack()))
			}
			close(done)
		}()
		val, err := p.Get()
		if err != nil {
			out.err = err
			return
		}
		out.val, out.err = f(ctx, val)
	}()
	return &out
}

func Zip[T, K, V any](ctx context.Context, pt *Promise[T], pk *Promise[K], f Func2[T, K, V]) *Promise[V] {
	done := make(chan struct{})
	out := Promise[V]{
		done: done,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				out.err = fmt.Errorf("recovered: %#v, stacktrace: %s", r, string(debug.Stack()))
			}
			close(done)
		}()
		val1, err1 := pt.Get()
		if err1 != nil {
			out.err = err1
			return
		}

		val2, err2 := pk.Get()
		if err2 != nil {
			out.err = err2
			return
		}
		out.val, out.err = f(ctx, val1, val2)
	}()

	return &out
}

func WaitForAll[T any](promises []*Promise[T]) (err error) {
	var wg sync.WaitGroup
	wg.Add(len(promises))
	errChan := make(chan error, len(promises))
	allDone := make(chan struct{})
	for _, promise := range promises {
		go func(p *Promise[T]) {
			defer wg.Done()
			lErr := p.Wait()
			if lErr != nil {
				errChan <- lErr
			}
		}(promise)
	}
	go func() {
		defer close(allDone)
		wg.Wait()
	}()

	select {
	case err = <-errChan:
		return
	case <-allDone:
		return
	}
}
