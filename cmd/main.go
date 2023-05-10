package main

import (
	"sync"

	"adaptivelimit/pkg/adaptive"
)

func main() {
	config := adaptive.InitConfig()
	client := adaptive.NewClient(config.Client)
	server := adaptive.NewServer(config.Server)
	adaptive.WatchConfig(client, server)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go server.Start()
	go client.Start()
	wg.Wait()
}
