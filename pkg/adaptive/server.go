package adaptive

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"adaptivelimit/pkg/resource"
	"adaptivelimit/pkg/util"
)

type Server struct {
	configMtx       sync.Mutex
	config          *ServerConfig
	adaptiveLimiter *AdaptiveLimiter
}

func NewServer(config *ServerConfig) *Server {
	server := &Server{
		config:          config,
		adaptiveLimiter: NewAdaptiveLimiter(resource.NewCpuLimiter(config.InitialCpuTime, config.TenantCpuTimes), util.Keys(config.TenantCpuTimes)),
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
			result := s.adaptiveLimiter.Acquire(tenant)
			if result != 2 {
				http.Error(w, http.StatusText(result), result)
			}
		} else {
			if !s.adaptiveLimiter.limitedResource.Acquire(tenant) {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			}
		}
	}
}

func (s *Server) Apply(config *ServerConfig) {
	s.configMtx.Lock()
	defer s.configMtx.Unlock()
	s.config = config
	s.adaptiveLimiter.limitedResource.SetInitialCpuTime(config.InitialCpuTime)
	fmt.Println(fmt.Sprintf("Reloaded server config: %v", config))
}

func (s *Server) getCpuTime(tenant string) int {
	s.configMtx.Lock()
	defer s.configMtx.Unlock()
	return s.config.TenantCpuTimes[tenant]
}
