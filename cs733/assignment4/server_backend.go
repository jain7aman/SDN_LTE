package main

import (
	"github.com/cs733-iitb/cluster"
	"log"
	"strings"
	"os"
	"strconv"
	"encoding/gob"
)

func initializeSystem(configFile string) (config *NewConfig) {
	
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
	
	return cfg
}



func main() {
//	done := make(chan bool, 1)
//	//initialize & start the file servers
//	initializeSystem("config1.json")
//
//	<-done
	deleteLogs2(5)
	RegisterEvents2()
	var id int
	id, err := strconv.Atoi(os.Args[1])
	if err != nil { 
		log.Fatal("wrong arguement")
	}
	configFile := os.Args[2]

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

	
	serverMain(id, addressMap, &clusterCfg)
	
}


func deleteLogs2(numServers int) {
	for i := 1; i <= numServers; i++ {
		if _, err := os.Stat("server_" + strconv.Itoa(i) + "_log"); err == nil {
			err = os.RemoveAll("server_" + strconv.Itoa(i) + "_log")
			if err != nil {
				panic(err)
			}
		}
		if _, err := os.Stat(strconv.Itoa(i) + "_state"); err == nil {
			err = os.RemoveAll(strconv.Itoa(i) + "_state")
			if err != nil {
				panic(err)
			}
		}
	}
}

func RegisterEvents2() {
	gob.Register(AppendEntriesReqEvent{})
	gob.Register(AppendEntriesRespEvent{})
	gob.Register(VoteReqEvent{})
	gob.Register(VoteRespEvent{})
	gob.Register(TimeoutEvent{})
	gob.Register(AppendEvent{})
	gob.Register(LogEntry{})
	gob.Register(StateStore{})
}
