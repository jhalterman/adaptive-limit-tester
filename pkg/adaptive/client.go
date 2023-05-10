package adaptive

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Client struct {
	configMtx       sync.Mutex
	config          *ClientConfig
	responseCounter *prometheus.CounterVec
}

func NewClient(config *ClientConfig) *Client {
	client := &Client{
		config: config,
		responseCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ad_http_response",
			},
			[]string{"tenant", "response"},
		),
	}
	//promauto.NewGaugeFunc(
	//	prometheus.GaugeOpts{
	//		Name: "ad_outstanding_client_requests",
	//	},
	//	func() float64 {
	//		return client.getOutstandingRequests()
	//	},
	//)
	return client
}

func (c *Client) Start() {
	for tenant := range c.config.TenantRps {
		go c.doRequest(tenant)
	}
}

func (c *Client) doRequest(tenant string) {
	for {
		rps := c.getRps(tenant)
		if rps == 0 {
			time.Sleep(time.Second)
			continue
		}

		go func() {
			client := http.Client{
				Timeout: 10 * time.Second,
			}
			resp, err := client.Get("http://localhost:8080/" + tenant)
			if err == nil {
				defer resp.Body.Close()
			}
			statusCode := "500"
			if resp != nil {
				statusCode = strconv.Itoa(resp.StatusCode)
			}
			c.responseCounter.WithLabelValues(tenant, statusCode).Inc()
		}()

		delay := time.Second / time.Duration(rps)
		time.Sleep(delay)
	}
}

func (c *Client) Apply(config *ClientConfig) {
	c.configMtx.Lock()
	defer c.configMtx.Unlock()
	c.config = config
	fmt.Println(fmt.Sprintf("Reloaded client config: %v", config))
}

func (c *Client) getRps(tenant string) int {
	c.configMtx.Lock()
	defer c.configMtx.Unlock()
	return c.config.TenantRps[tenant]
}
