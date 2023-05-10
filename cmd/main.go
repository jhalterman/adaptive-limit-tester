package main

import (
	"sync"

	"adaptivelimit/pkg/adaptive"
)

func main() {
	config := adaptive.InitConfig()
	client := adaptive.NewClient(config)
	server := adaptive.NewServer(config.Server, config)
	adaptive.WatchConfig(client, server)

	go server.Start()
	go client.Start()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
