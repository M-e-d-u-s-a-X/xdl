package httpx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type RequestOptionsRuntime struct {
	Method      string
	URI         string
	Params      url.Values
	Headers     http.Header
	Body        []byte
	Timeout     time.Duration
	WithCookies bool
}

type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
}

func DoRequest(ctx context.Context, client *http.Client, opt RequestOptionsRuntime) (*Response, error) {
	if client == nil {
		return nil, errors.New("nil http.Client")
	}
	if opt.Method == "" {
		opt.Method = http.MethodGet
	}
	if opt.Timeout < 0 {
		opt.Timeout = 0
	}

	u, err := url.Parse(opt.URI)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	for k, vals := range opt.Params {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()

	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	var body io.Reader
	if len(opt.Body) > 0 {
		body = bytes.NewReader(opt.Body)
	}

	req, err := http.NewRequestWithContext(ctx, opt.Method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// Headers.
	for k, vals := range opt.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	if !opt.WithCookies {
		req.Header.Del("Cookie")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	out := &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header.Clone(),
		Body:       bodyBytes,
	}
	return out, nil
}

type DebouncedFn[T any] struct {
	mu       sync.Mutex
	fn       func(T)
	interval time.Duration
	leading  bool

	timer   *time.Timer
	pending bool
	lastArg T
}

func NewDebouncedFn[T any](interval time.Duration, leading bool, fn func(T)) *DebouncedFn[T] {
	return &DebouncedFn[T]{
		fn:       fn,
		interval: interval,
		leading:  leading,
	}
}

func (d *DebouncedFn[T]) Call(arg T) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastArg = arg

	if d.timer == nil {
		if d.leading {
			go d.fn(arg)
		}
		d.startTimerLocked()
		return
	}

	d.pending = true
	if !d.timer.Stop() {
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.startTimerLocked()
}

func (d *DebouncedFn[T]) startTimerLocked() {
	d.pending = true
	d.timer = time.AfterFunc(d.interval, func() {
		d.mu.Lock()
		arg := d.lastArg
		d.pending = false
		d.timer = nil
		d.mu.Unlock()

		d.fn(arg)
	})
}
