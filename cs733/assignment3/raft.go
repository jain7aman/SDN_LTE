package main

import (
	"strconv"
	"time"
	"math/rand"
)


func hh(){
	
	var config Config
	var numServers int = 5
	var startingPort int = 8080
	servers := make([]RaftServer, numServers)
	s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)
	
	for i := 0; i < numServers; i++ {
		config.cluster = append(config.cluster, NetConfig{Id:i, Host:"localhost", Port:startingPort})
		config.Id = i
		config.LogDir = "server_" + strconv.Itoa(i) + "_log"
		config.ElectionTimeout = r1.Intn(100)
		config.HeartbeatTimeout = r1.Intn(100)
		startingPort++
		servers[i] = New(config)
	}
	
	for i := 0; i < 5; i++ {
		go servers[i].NodeStart()
	}

}

