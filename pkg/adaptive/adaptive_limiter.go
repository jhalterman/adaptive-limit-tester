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

func NewConcurrencyLimiter(cpuLimiter *CpuLimiter, clients []string) *ConcurrencyLimiter {
	partitions := make(map[string]*strategy.LookupPartition)

	for _, client := range clients {
		partition := strategy.NewLookupPartitionWithMetricRegistry(
			client,
			1.0/float64(len(clients)),
			1,
			core.EmptyMetricRegistryInstance,
		)
		promauto.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "ad_client_limit",
				ConstLabels: map[string]string{"client": client},
			},
			func() float64 {
				return float64(partition.Limit())
			},
		)
		promauto.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "ad_client_concurrency",
				ConstLabels: map[string]string{"client": client},
			},
			func() float64 {
				return float64(partition.BusyCount())
			},
		)
		partitions[client] = partition
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

func (l *ConcurrencyLimiter) Acquire(client string) int {
	ctx := context.WithValue(context.Background(), matchers.LookupPartitionContextKey, client)
	token, ok := l.concurrencyLimiter.Acquire(ctx)
	if !ok {
		return 430 // 430 indicates a concurrency limiter rejection
	}

	if !l.cpuLimiter.Acquire(client) {
		token.OnDropped()
		return 429 // 429 indicates a CPU limiter rejection
	}
	token.OnSuccess()
	return 200
}
