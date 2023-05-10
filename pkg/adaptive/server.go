package adaptive

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"adaptivelimit/pkg/util"
)

type Server struct {
	configMtx          sync.Mutex
	config             *ServerConfig
	cpuLimiter         *CpuLimiter
	concurrencyLimiter *ConcurrencyLimiter
}

func NewServer(config *ServerConfig) *Server {
	cpuLimiter := NewCpuLimiter(config.InitialCpuTime, config.TenantCpuTimes)
	server := &Server{
		config:             config,
		cpuLimiter:         cpuLimiter,
		concurrencyLimiter: NewConcurrencyLimiter(cpuLimiter, util.Keys(config.TenantCpuTimes)),
	}
	for tenant := range config.TenantCpuTimes {
		http.HandleFunc("/"+tenant, server.cpuLimitedHandler(tenant))
	}
	return server
}

func (s *Server) Start() {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Listening on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func (s *Server) cpuLimitedHandler(tenant string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.Fairness {
			result := s.concurrencyLimiter.Acquire(tenant)
			if result != 200 {
				http.Error(w, http.StatusText(result), result)
			}
		} else {
			if !s.cpuLimiter.Acquire(tenant) {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			}
		}
	}
}

func (s *Server) Apply(config *ServerConfig) {
	s.configMtx.Lock()
	defer s.configMtx.Unlock()
	s.config = config
	s.concurrencyLimiter.cpuLimiter.SetInitialCpuTime(config.InitialCpuTime)
	fmt.Println(fmt.Sprintf("Reloaded server config: %v", config))
}

func (s *Server) getCpuTime(tenant string) int {
	s.configMtx.Lock()
	defer s.configMtx.Unlock()
	return s.config.TenantCpuTimes[tenant]
}
