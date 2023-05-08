package promise

import (
	"context"
	"fmt"
	"github.com/tj/assert"
	"strconv"
	"testing"
	"time"
)

func TestChaining(t *testing.T) {
	ctx := context.Background()
	p1 := NewPromise(ctx, 10, func(ctx context.Context, i int) (int, error) {
		return i * 2, nil
	})
	p2 := Then(ctx, p1, func(ctx context.Context, i int) (string, error) {
		return strconv.Itoa(i), nil
	})

	p3 := Then(ctx, p1, func(ctx context.Context, i int) (string, error) {
		return strconv.Itoa(i * 3), nil
	})

	if val, err := p2.Get(); err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, "20", val)
	}

	if val, err := p3.Get(); err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, "60", val)
	}

}

func TestZip(t *testing.T) {
	ctx := context.Background()
	p1 := NewPromise(ctx, 10, func(ctx context.Context, i int) (int, error) {
		return i * 2, nil
	})

	p2 := NewPromise(ctx, 10, func(ctx context.Context, i int) (string, error) {
		return "as promised", nil
	})

	p3 := Zip(ctx, p1, p2, func(ctx context.Context, i1 int, i2 string) (string, error) {
		return strconv.Itoa(i1) + " " + i2, nil
	})

	if val, err := p3.Get(); err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, "20 as promised", val)
	}
}

func TestFanningOut(t *testing.T) {
	ctx := context.Background()
	/*
		                 /-> p4(p3 + p1) - \
		p1 -> p2 -> p3 -<                    > p6 -> result
		                 \-> p5(p2 + p1) - /
	*/

	p1 := NewPromise(ctx, 1, func(ctx context.Context, i int) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return i + 1, nil
	})

	p2 := Then(ctx, p1, func(ctx context.Context, i int) (int, error) {
		time.Sleep(5 * time.Millisecond)
		return i + 1, nil
	})

	p3 := Then(ctx, p2, func(ctx context.Context, i int) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return i * 10, nil
	})

	p4 := Zip(ctx, p3, p1, func(ctx context.Context, i3, i1 int) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return strconv.Itoa(i1 + i3), nil
	})

	p5 := Zip(ctx, p2, p1, func(ctx context.Context, i1, i2 int) (string, error) {
		time.Sleep(25 * time.Millisecond)
		return strconv.Itoa(i2 + i1), nil
	})

	p6 := Zip(ctx, p4, p5, func(ctx context.Context, s4, s5 string) (string, error) {
		return s4 + " and " + s5, nil
	})

	if val, err := p6.Get(); err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, "32 and 5", val)
	}
}

func TestPromiseSlice(t *testing.T) {
	ctx := context.Background()

	var promises []Waiter

	ws1 := WithCancellation(func(ctx context.Context, i int) (int, error) {
		return i * 2, nil
	})

	ws2 := WithCancellation(func(ctx context.Context, i int) (int, error) {
		return i * 3, nil
	})

	p1 := NewPromise(ctx, 10, ws1)
	p2 := NewPromise(ctx, 10, ws2)

	promises = append(promises, p1, p2)

	if err := Wait(promises...); err != nil {
		t.Error(err)
	}

	v1, _ := p1.Get()
	v2, _ := p2.Get()
	fmt.Println(v1, v2)
}

func TestWithCancellation(t *testing.T) {
	ctx := context.Background()
	ws1 := WithCancellation(func(ctx context.Context, i int) (int, error) {
		return i * 2, nil
	})

	ws2 := WithCancellation(func(ctx context.Context, i int) (string, error) {
		return strconv.Itoa(i), nil
	})

	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	p1 := NewPromise(ctx, 10, ws1)
	p2 := NewPromise(ctx, 10, ws2)
	if err := Wait(p1, p2); err != nil {
		t.Error(err)
	}

	type intermediateResults struct {
		result1 int
		result2 string
	}
	ws3 := WithCancellation(func(ctx context.Context, ir intermediateResults) (string, error) {
		return fmt.Sprintf("%d-%s-done", ir.result1, ir.result2), nil
	})

	var ir intermediateResults
	ir.result1, _ = p1.Get()
	ir.result2, _ = p2.Get()
	p3 := NewPromise(ctx, ir, ws3)
	if _, err := p3.Get(); err != nil {
		t.Error(err)
	}
}

func TestPromise_PanicOnWaitAll(t *testing.T) {
	ctx := context.Background()
	panickingPromise := NewPromise(ctx, 10, func(ctx context.Context, x int) (int, error) {
		time.Sleep(time.Second)
		panic(fmt.Errorf("boo"))
	})

	err := Wait(panickingPromise)
	assert.Error(t, err)
}

func TestPromise_PanicOnGet(t *testing.T) {
	ctx := context.Background()
	panickingPromise := NewPromise(ctx, 10, func(ctx context.Context, x int) (int, error) {
		time.Sleep(time.Second)
		panic(fmt.Errorf("boo"))
	})

	_, err := panickingPromise.Get()
	assert.Error(t, err)
}

func TestPromise_PanicOnWait(t *testing.T) {
	ctx := context.Background()
	panickingPromise := NewPromise(ctx, 10, func(ctx context.Context, x int) (int, error) {
		time.Sleep(time.Second)
		panic(fmt.Errorf("boo"))
	})

	err := panickingPromise.Wait()
	assert.Error(t, err)
}
