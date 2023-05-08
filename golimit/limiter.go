package golimit

type Limiter interface {
	Reserve()
	Release()
	Destroy()
}

type limiter struct {
	buffChan chan struct{}
}

func NewLimiter(limit int) Limiter {
	return &limiter{buffChan: make(chan struct{}, limit)}
}

func (l *limiter) Destroy() {
	close(l.buffChan)
}

func (l *limiter) Reserve() {
	l.buffChan <- struct{}{}
}

func (l *limiter) Release() {
	<-l.buffChan
}
