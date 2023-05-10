package adaptive

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	configMtx          sync.Mutex
	serverConfig       *ServerConfig
	clientConfigs      ClientConfigs
	cpuLimiter         *CpuLimiter
	concurrencyLimiter *ConcurrencyLimiter
}

func NewServer(serverConfig *ServerConfig, clientConfigs ClientConfigs) *Server {
	cpuLimiter := NewCpuLimiter(!serverConfig.AdaptiveLimiting, serverConfig.AvailableCpuTime, clientConfigs.GetCpuTimes())
	server := &Server{
		serverConfig:       serverConfig,
		clientConfigs:      clientConfigs,
		cpuLimiter:         cpuLimiter,
		concurrencyLimiter: NewConcurrencyLimiter(cpuLimiter, clientConfigs.GetClients()),
	}
	for _, client := range clientConfigs.GetClients() {
		http.HandleFunc("/"+client, server.cpuLimitedHandler(client))
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

func (s *Server) cpuLimitedHandler(client string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.serverConfig.AdaptiveLimiting {
			result := s.concurrencyLimiter.Acquire(client)
			if result != 200 {
				http.Error(w, http.StatusText(result), result)
			}
		} else {
			if !s.cpuLimiter.Acquire(client) {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			}
		}
	}
}

func (s *Server) Apply(serverConfig *ServerConfig, clientConfigs ClientConfigs) {
	s.configMtx.Lock()
	defer s.configMtx.Unlock()
	s.serverConfig = serverConfig
	s.clientConfigs = clientConfigs
	s.concurrencyLimiter.cpuLimiter.SetInitialCpuTime(serverConfig.AvailableCpuTime)
	fmt.Println(fmt.Sprintf("Reloaded server serverConfig: %v", serverConfig))
}
