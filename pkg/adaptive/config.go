package adaptive

import (
	"encoding/json"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	v *viper.Viper
)

type Config struct {
	Client *ClientConfig `mapstructure:"client"`
	Server *ServerConfig `mapstructure:"server"`
}

type ServerConfig struct {
	Fairness       bool           `mapstructure:"fairness"`
	InitialCpuTime int            `mapstructure:"initial_cpu_time"`
	TenantCpuTimes map[string]int `mapstructure:"cpu_time"`
}

func (c *ServerConfig) String() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}

type ClientConfig struct {
	TenantRps map[string]int `mapstructure:"rps"`
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
		client.Apply(config.Client)
		server.Apply(config.Server)
	})
	v.WatchConfig()
}
