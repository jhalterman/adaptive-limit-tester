package adaptive

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CpuLimiter struct {
	// Static values
	tenantCpuTimes map[string]int
	mtx            sync.Mutex

	// Dynamic values
	configVersion      int // Incremented when a new initial cpu time is set
	initialCpuTime     int
	remainingCpuTime   int
	cpuInUse           int
	concurrentRequests int
}

func NewCpuLimiter(initialCpuTime int, tenantCpuTimes map[string]int) *CpuLimiter {
	cpuLimiter := &CpuLimiter{
		initialCpuTime:   initialCpuTime,
		remainingCpuTime: initialCpuTime,
		tenantCpuTimes:   tenantCpuTimes,
	}
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "ad_cpu_time_remaining",
		},
		func() float64 {
			return float64(cpuLimiter.GetRemainingCpuTime())
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "ad_cpu_time_used",
		},
		func() float64 {
			return float64(cpuLimiter.GetCpuInUse())
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "ad_concurrent_requests",
		},
		func() float64 {
			return float64(cpuLimiter.GetConcurrentRequests())
		},
	)
	return cpuLimiter
}

func (l *CpuLimiter) Acquire(tenant string) bool {
	l.mtx.Lock()
	initialVersion := l.configVersion
	cpuTime := l.tenantCpuTimes[tenant]
	if l.remainingCpuTime-cpuTime < 0 {
		l.mtx.Unlock()
		//	fmt.Println("no cpu time remaining")
		return false
	}

	l.remainingCpuTime -= cpuTime
	l.cpuInUse += cpuTime
	l.concurrentRequests++
	l.mtx.Unlock()

	// Simulate execution by sleeping
	delay := time.Millisecond * time.Duration(cpuTime)
	time.Sleep(delay)
	l.release(initialVersion, cpuTime)
	return true
}

func (l *CpuLimiter) release(configVersion int, cpuTime int) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if configVersion == l.configVersion {
		l.remainingCpuTime += cpuTime
		l.cpuInUse -= cpuTime
		l.concurrentRequests--
	}
}

func (l *CpuLimiter) SetInitialCpuTime(cpuTime int) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.initialCpuTime != cpuTime {
		l.remainingCpuTime = cpuTime
		l.cpuInUse = 0
		l.concurrentRequests = 0
		l.configVersion++
	}
}

func (l *CpuLimiter) GetCpuInUse() int {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.cpuInUse
}

func (l *CpuLimiter) GetRemainingCpuTime() int {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.remainingCpuTime
}

func (l *CpuLimiter) GetConcurrentRequests() int {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.concurrentRequests
}
