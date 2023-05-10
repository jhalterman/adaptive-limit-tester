package adaptive

import (
	"encoding/json"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"adaptivelimit/pkg/util"
)

var (
	v *viper.Viper
)

type ClientConfigs interface {
	GetClients() []string
	GetCpuTimes() map[string]int
	GetRps(client string) int
}

type Config struct {
	Clients map[string]ClientConfig `mapstructure:"clients"`
	Server  *ServerConfig           `mapstructure:"server"`
}

func (c *Config) GetClients() []string {
	return util.Keys(c.Clients)
}

func (c *Config) GetCpuTimes() map[string]int {
	result := make(map[string]int)
	for client, lc := range c.Clients {
		result[client] = lc.CpuTime
	}
	return result
}

func (c *Config) GetRps(client string) int {
	return c.Clients[client].Rps
}

type ServerConfig struct {
	Fairness         bool `mapstructure:"fairness"`
	AvailableCpuTime int  `mapstructure:"available_cpu_time"`
}

func (c *ServerConfig) String() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}

type ClientConfig struct {
	Rps     int `mapstructure:"rps"`
	CpuTime int `mapstructure:"cpu_time"`
}

func (c *ClientConfig) String() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}

func InitConfig() *Config {
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	config := readConfig()
	return config
}

func readConfig() *Config {
	var config *Config
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config file: %s", err))
	}
	err = v.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %s", err))
	}
	return config
}

func WatchConfig(client *Client, server *Server) {
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Reapplying config")
		config := readConfig()
		client.Apply(config)
		server.Apply(config.Server, config)
	})
	v.WatchConfig()
}
