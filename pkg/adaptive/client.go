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
	clientConfigs   ClientConfigs
	responseCounter *prometheus.CounterVec
	rttCounter      *prometheus.CounterVec
}

func NewClient(clientConfigs ClientConfigs) *Client {
	client := &Client{
		clientConfigs: clientConfigs,
		responseCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ad_http_response",
			},
			[]string{"client", "response"},
		),
		rttCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ad_http_rtt",
			},
			[]string{"client"},
		),
	}
	return client
}

func (c *Client) Start() {
	for _, client := range c.clientConfigs.GetClients() {
		go c.doRequest(client)
	}
}

func (c *Client) doRequest(client string) {
	for {
		rps := c.getRps(client)
		if rps == 0 {
			time.Sleep(time.Second)
			continue
		}

		go func() {
			httpClient := http.Client{
				Timeout: 10 * time.Second,
			}
			startTime := time.Now()
			resp, err := httpClient.Get("http://localhost:8080/" + client)
			elapsed := time.Now().Sub(startTime).Milliseconds()
			c.rttCounter.WithLabelValues(client).Add(float64(elapsed))
			if err == nil {
				defer resp.Body.Close()
			}
			statusCode := "500"
			if resp != nil {
				statusCode = strconv.Itoa(resp.StatusCode)
			}
			c.responseCounter.WithLabelValues(client, statusCode).Inc()
		}()

		delay := time.Second / time.Duration(rps)
		time.Sleep(delay)
	}
}

func (c *Client) Apply(clientConfigs ClientConfigs) {
	c.configMtx.Lock()
	defer c.configMtx.Unlock()
	c.clientConfigs = clientConfigs
	fmt.Println(fmt.Sprintf("Reloaded client clientConfigs: %v", clientConfigs))
}

func (c *Client) getRps(client string) int {
	c.configMtx.Lock()
	defer c.configMtx.Unlock()
	return c.clientConfigs.GetRps(client)
}
