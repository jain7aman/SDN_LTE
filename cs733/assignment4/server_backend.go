package main

import (
	"github.com/cs733-iitb/cluster"
	"log"
	"strings"
)

func initializeServer(configFile string) {
	// read the configuration file
	cfg, err := ToConfig(configFile)

	if err != nil {
		log.Fatal(err)
	}
	var addressMap = make(map[int]string)
	numServers := len(cfg.Peers)

	peersConfig := make([]cluster.PeerConfig, numServers)
	for i := 0; i < numServers; i++ {
		peersConfig[i] = cluster.PeerConfig{Id: cfg.Peers[i].Id, Address: cfg.Peers[i].Address}
		addressMap[i+1] = strings.Split(cfg.Peers[i].Address, ":")[0] + cfg.Peers[i].ClientPort
	}

	clusterCfg := cluster.Config{Peers: peersConfig, InboxSize: cfg.InboxSize, OutboxSize: cfg.OutboxSize}

	for i := 1; i <= numServers; i++ {
		go serverMain(i, addressMap, &clusterCfg)
	}
}

type T struct {
	A string
	B float64
}

func main() {
	done := make(chan bool, 1)
	//initialize & start the file servers
	initializeServer("config1.json")

	<-done
}
