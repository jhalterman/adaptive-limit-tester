package adaptive

import (
	"context"

	"github.com/platinummonkey/go-concurrency-limits/core"
	"github.com/platinummonkey/go-concurrency-limits/limit"
	"github.com/platinummonkey/go-concurrency-limits/limiter"
	"github.com/platinummonkey/go-concurrency-limits/strategy"
	"github.com/platinummonkey/go-concurrency-limits/strategy/matchers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ConcurrencyLimiter struct {
	concurrencyLimiter *limiter.DefaultLimiter
	cpuLimiter         *CpuLimiter
}

func NewConcurrencyLimiter(cpuLimiter *CpuLimiter, tenants []string) *ConcurrencyLimiter {
	partitions := make(map[string]*strategy.LookupPartition)

	for _, tenant := range tenants {
		partition := strategy.NewLookupPartitionWithMetricRegistry(
			tenant,
			1.0/float64(len(tenants)),
			1,
			core.EmptyMetricRegistryInstance,
		)
		promauto.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "ad_tenant_limit",
				ConstLabels: map[string]string{"tenant": tenant},
			},
			func() float64 {
				return float64(partition.Limit())
			},
		)
		promauto.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "ad_tenant_busy_count",
				ConstLabels: map[string]string{"tenant": tenant},
			},
			func() float64 {
				return float64(partition.BusyCount())
			},
		)
		partitions[tenant] = partition
	}

	partition, _ := strategy.NewLookupPartitionStrategyWithMetricRegistry(partitions, nil, 10, core.EmptyMetricRegistryInstance)
	//adaptiveLimit := limit.NewDefaultAIMDLimit("adaptive", nil)
	adaptiveLimit := limit.NewDefaultVegasLimit("vegas", limit.BuiltinLimitLogger{}, core.EmptyMetricRegistryInstance)
	partitionedLimiter, _ := limiter.NewDefaultLimiter(
		adaptiveLimit,
		int64(1e9),
		int64(1e9),
		int64(1e5),
		100,
		partition,
		limit.BuiltinLimitLogger{},
		core.EmptyMetricRegistryInstance)

	return &ConcurrencyLimiter{
		concurrencyLimiter: partitionedLimiter,
		cpuLimiter:         cpuLimiter,
	}
}

func (l *ConcurrencyLimiter) Acquire(tenant string) int {
	ctx := context.WithValue(context.Background(), matchers.LookupPartitionContextKey, tenant)
	token, ok := l.concurrencyLimiter.Acquire(ctx)
	if !ok {
		return 430 // 430 indicates a concurrency limiter rejection
	}

	if !l.cpuLimiter.Acquire(tenant) {
		token.OnDropped()
		return 429 // 429 indicates a CPU limiter rejection
	}
	token.OnSuccess()
	return 200
}
