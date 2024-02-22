// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"math"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

// RetryCond is a condition that applies only to retry backoff mechanism.
type RetryCond func(resp *http.Response, err error) bool

// RetryFunc takes attemps number, minimal and maximal wait time for backoff.
// Returns duration that mechanism have to wait before making a request.
type RetryFunc func(n int, min, max time.Duration) time.Duration

// Retrier defines general interface for custom retry algo implementations.
type Retrier interface {
	Next() time.Duration
	Reset() int64
	Attempt() int64
}

// backoff is a thread-safe retry backoff mechanism.
// Currently supported only ExponentalBackoff retry algorithm.
type backoff struct {
	minWaitTime time.Duration
	maxWaitTime time.Duration
	maxAttempts int64
	attempts    int64
	f           RetryFunc
}

const stopBackoff time.Duration = -1

func (b *backoff) Next() time.Duration {
	if atomic.LoadInt64(&b.attempts) >= b.maxAttempts {
		return stopBackoff
	}
	atomic.AddInt64(&b.attempts, 1)
	return b.f(int(atomic.LoadInt64(&b.attempts)), b.minWaitTime, b.maxWaitTime)
}

func (b *backoff) Reset() int64 {
	return atomic.SwapInt64(&b.attempts, 0)
}

func (b *backoff) Attempt() int64 {
	return atomic.LoadInt64(&b.attempts)
}

func ExponentalBackoff(attemptNum int, min, max time.Duration) time.Duration {
	const factor = 2.0
	rand.Seed(time.Now().UnixNano())
	delay := time.Duration(math.Pow(factor, float64(attemptNum)) * float64(min))
	jitter := time.Duration(rand.Float64() * float64(min) * float64(attemptNum))

	delay = delay + jitter
	if delay > max {
		delay = max
	}

	return delay
}
