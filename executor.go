package main

import (
	"context"
	"sync"
	"time"

	"github.com/platinummonkey/go-concurrency-limits/core"
	"github.com/platinummonkey/go-concurrency-limits/limiter"
)

type CpuLimitedExecutor struct {
	version            int // Incremented each time initial CpuMS is refreshed
	initialCpuTime     int
	remainingCpuTime   int
	cpuInUse           int
	concurrentRequests int
	mu                 sync.Mutex
	limiter            *limiter.DefaultLimiter
}

func (e *CpuLimitedExecutor) Execute(ctx context.Context, cpuTime int) bool {
	v := e.version
	token, adjustedCpuTime, ok := e.acquire(ctx, cpuTime)
	if !ok {
		return false
	}

	// Simulate execution by sleeping
	delay := time.Millisecond * time.Duration(adjustedCpuTime)
	time.Sleep(delay)
	if v == e.version {
		e.release(cpuTime)
		if token != nil {
			token.OnSuccess()
		}
	}
	return true
}

func (e *CpuLimitedExecutor) GetRemainingCpuTime() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.remainingCpuTime
}

func (e *CpuLimitedExecutor) GetCpuInUse() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cpuInUse
}

func (e *CpuLimitedExecutor) GetConcurrentRequests() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.concurrentRequests
}

func (e *CpuLimitedExecutor) SetInitialCpuTime(cpuTime int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.initialCpuTime != cpuTime {
		e.remainingCpuTime = cpuTime
		e.cpuInUse = 0
		e.concurrentRequests = 0
		e.version++
	}
}

func (e *CpuLimitedExecutor) acquire(ctx context.Context, cpuTime int) (core.Listener, int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	token, ok := e.limiter.Acquire(ctx)
	if !ok {
		return nil, cpuTime, false
	}
	if e.remainingCpuTime-cpuTime < 0 {
		// We are out of CPU, so let the limiter know thta we're dropping the execution attempt
		token.OnDropped()
		return nil, cpuTime, false
	}
	e.remainingCpuTime -= cpuTime
	e.cpuInUse += cpuTime
	e.concurrentRequests++
	return token, cpuTime, true
	//return nil, cpuTime, true
}

func (e *CpuLimitedExecutor) release(cpuMS int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.remainingCpuTime += cpuMS
	e.cpuInUse -= cpuMS
	e.concurrentRequests--
}
