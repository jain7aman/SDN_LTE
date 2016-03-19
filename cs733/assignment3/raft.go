package main

import (
	"encoding/gob"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	diskLog "github.com/cs733-iitb/log"
	//	"log"
	"math/rand"
	"strconv"
	"time"
)

var s1 rand.Source = rand.NewSource(time.Now().UnixNano())
var r1 *rand.Rand = rand.New(s1)

func Initialize(configFile string) []Node {
	// read the configuration file
	tmpConfig := GetConfig(configFile)

	numServers := len(tmpConfig.cluster)
	servers := make([]Node, numServers)
	configs := make([]Config, numServers)

	for i := 1; i <= numServers; i++ {
		configs[i-1].cluster = tmpConfig.cluster
		configs[i-1].Id = i
		configs[i-1].LogDir = "server_" + strconv.Itoa(i) + "_log"
		//		if i == 1 {
		//			configs[i-1].ElectionTimeout = 2
		//			configs[i-1].HeartbeatTimeout = 10
		//		} else {
		configs[i-1].ElectionTimeout = r1.Intn(1000) + 400
		configs[i-1].HeartbeatTimeout = r1.Intn(100) + 10
		//		}
		servers[i-1] = New(configs[i-1], true)
	}

	return servers
}

func InitializeMocks(configFile string) (*mock.MockCluster, []Node) {
	// read the configuration file
	tmpConfig, c1 := createMockCluster(configFile)
	var electionTimeout, heartbeatTimeout int
	numServers := len(tmpConfig.Peers)
	servers := make([]Node, numServers)

	for i := 1; i <= numServers; i++ {
		electionTimeout = r1.Intn(1000) + 400
		heartbeatTimeout = r1.Intn(100) + 10

		servers[i-1] = NewMock(c1, tmpConfig.Peers[i-1].Id, numServers, "server_"+strconv.Itoa(tmpConfig.Peers[i-1].Id)+"_log", strconv.Itoa(tmpConfig.Peers[i-1].Id)+"_state", electionTimeout, heartbeatTimeout, true)
	}

	return c1, servers
}

func makeMockRafts(configFile string) (*mock.MockCluster, []Node) {
	c1, nodes := InitializeMocks(configFile)
	//	if Debug {
	//		log.Printf("Registering Events")
	//	}
	RegisterEvents()
	//	if Debug {
	//		log.Printf("Starting channels")
	//	}
	for i := 0; i < len(nodes); i++ {
		go handleChannels(nodes[i].(*RaftServer))
	}
	return c1, nodes
	//	<-done
}

func makeRafts(configFile string) []Node {
	//	done := make(chan bool, 1)
	nodes := Initialize(configFile)
	//	if Debug {
	//		log.Printf("Registering Events")
	//	}
	RegisterEvents()
	//	if Debug {
	//		log.Printf("Starting channels")
	//	}
	for i := 0; i < len(nodes); i++ {
		go handleChannels(nodes[i].(*RaftServer))
	}
	return nodes
	//	<-done
}

func main() {

}

/*
func mmain() {
	done := make(chan bool, 1)
	nodes := Initialize("Cluster_config1.json")
	numServers := len(nodes)
	if Debug {
		log.Printf("Registering Events")
	}
	RegisterEvents()
	if Debug {
		log.Printf("Clearing Logs")
	}
	for i := 0; i < numServers; i++ {
		nodes[i].(*RaftServer).ClearLogs()

	}
	done <- true

	<-done

//		if Debug {
//				log.Printf("Starting channels")}
//
//
//		for i := 0; i < numServers; i++ {
//			//		if Debug {
//				log.Printf("i %v \n", i)}
//			//go method1()
//			if Debug {
//				log.Printf("server = %v and ;length = %v \n", nodes[i].(*RaftServer).ID, len(nodes[i].(*RaftServer).Log))}
//			go handleChannels(nodes[i].(*RaftServer))
//		}
//
//		//time.Sleep(1 * time.Second)
//		for {
//			ldr := getLeader(nodes)
//
//			if ldr.ID != -1 {
//				if Debug {
//				log.Printf("%v:: ##################LEADER ID = %v ##################", time.Now().Nanosecond(), ldr.Id())}
//				ldr.Append([]byte("Msg_ID:1@@Command1"))
//				time.Sleep(1 * time.Second)
//
//				for i := 0; i < numServers; i++ {
//					select {
//					case ci := <-nodes[i].(*RaftServer).CommitChannel():
//						commitAction := ci.(Commit)
//						if commitAction.Err != nil {
//							panic(commitAction.Err)
//						}
//						if string(commitAction.Data) != "Msg_ID:1@@Command1" {
//							log.Fatal("Got different data")
//						}
//					default:
//						log.Fatal("Expected message on all nodes")
//					}
//
//				}
//							done <- true
//				break
//
//			}
//		}
//		//	go handleChannels(nodes[0].(*RaftServer))
//		//	go handleChannels(nodes[1].(*RaftServer))
//		//	go handleChannels(nodes[2].(*RaftServer))
//		//	go handleChannels(nodes[3].(*RaftServer))
//		//	handleChannels(nodes[4].(*RaftServer))
//
//		if Debug {
//				log.Printf("DONE\n")}
//
//		<-done

}
*/
func getACandidate(nodes []Node) *RaftServer {
	for {
		for i := 0; i < len(nodes); i++ {
			if nodes[i].(*RaftServer).State == CANDIDATE {
				return nodes[i].(*RaftServer)
			}
		}
		time.Sleep(1 * time.Second)
	}
	return &RaftServer{ID: -1, LeaderID: -1}
}

func getLeader(nodes []Node) *RaftServer {
	for {
		for i := 0; i < len(nodes); i++ {
			if nodes[i].(*RaftServer).State == LEADER {
				return nodes[i].(*RaftServer)
			}
		}
		time.Sleep(1 * time.Second)
	}
	return &RaftServer{ID: -1, LeaderID: -1}
}

func getNumberOfLeaders(nodes []Node) int {
	cnt := 0
	for i := 0; i < len(nodes); i++ {
		if nodes[i].(*RaftServer).State == LEADER {
			cnt++
		}
	}
	return cnt
}

func getAFollower(nodes []Node, leaderId int) *RaftServer {
	for i := 1; i <= len(nodes); i++ {
		if i != leaderId {
			return nodes[i].(*RaftServer)
		}
	}
	return &RaftServer{ID: -1, LeaderID: -1}
}

func checkLeader(nodes []Node) *RaftServer {
	for i := 0; i < len(nodes); i++ {
		if nodes[i].(*RaftServer).State == LEADER {
			return nodes[i].(*RaftServer)
		}
	}
	return &RaftServer{ID: -1, LeaderID: -1}
}

func handleChannels(sm *RaftServer) {
	//	if Debug {
	//		log.Printf(" id = %v method ElectionTimeout = %v HeartbeatTimeout = %v \n", sm.Id(), sm.ElectionTimeout, sm.HeartbeatTimeout)
	//	}
	done := false
	for {
		select {
		case ev := <-sm.SendChannel:
			handleActions(sm, ev)
		case event := <-sm.Server.Inbox():
			handleEvents(sm, event)
		case quit := <-sm.QuitChannel:
			if quit {
				done = true
			}
			break
		}
		if done {
			//			if Debug {
			//				log.Printf("Stopping Handling routines of Server = %v \n", sm.Id())
			//			}
			break
		}
	}
}

func reviveServer(id int, configFile string) *RaftServer {
	tmpConfig := GetConfig(configFile)
	var conf Config
	conf.cluster = tmpConfig.cluster
	conf.Id = id
	conf.LogDir = "server_" + strconv.Itoa(id) + "_log"
	conf.ElectionTimeout = r1.Intn(1000) + 100
	conf.HeartbeatTimeout = r1.Intn(100) + 10
	node := New(conf, false)

	go handleChannels(node.(*RaftServer))

	return node.(*RaftServer)

}

func handleEvents(sm *RaftServer, event *cluster.Envelope) {

	//event := <-sm.Server.Inbox()

	switch event.Msg.(type) {
	case TimeoutEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(TimeoutEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(TimeoutEvent)
	case VoteReqEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(VoteReqEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(VoteReqEvent)
	case AppendEntriesReqEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(AppendEntriesReqEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(AppendEntriesReqEvent)
	case AppendEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(AppendEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(AppendEvent)
	case AppendEntriesRespEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(AppendEntriesRespEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(AppendEntriesRespEvent)
	case VoteRespEvent:
		//		if Debug {
		//				log.Printf("In Handle Events %v \n", event.Msg.(VoteRespEvent).getEventName())}
		sm.ReceiveChannel <- event.Msg.(VoteRespEvent)
	}

}

func handleActions(sm *RaftServer, act Action) {

	//for act := range sm.SendChannel {
	switch act.(type) {
	case Alarm:
		alarmAction := act.(Alarm)
		//		if Debug {
		//			log.Printf("%v:: ALARM Action by %v and state = %v \n", time.Now().Nanosecond(), sm.ID, sm.getState())
		//		}
		if !sm.TimerSet {
			sm.TimerSet = true
			if sm.State == LEADER {
				//				sm.Timer = time.AfterFunc(time.Duration(alarmAction.Time+uint(r1.Intn(0)))*time.Millisecond, func() { sm.ReceiveChannel <- TimeoutEvent{} })
				sm.Timer = time.AfterFunc(time.Duration(alarmAction.Time)*time.Millisecond, func() { sm.ReceiveChannel <- TimeoutEvent{} })
			} else {
				sm.Timer = time.AfterFunc(time.Duration(alarmAction.Time+uint(r1.Intn(1000)))*time.Millisecond, func() { sm.ReceiveChannel <- TimeoutEvent{} })
			}
		} else {
			if alarmAction.Time == HighTimeout {
				sm.Timer.Stop()
				sm.TimerSet = false
			} else {
				if sm.State == LEADER {
					//					sm.Timer.Reset(time.Duration(alarmAction.Time+uint(r1.Intn(0))) * time.Millisecond)
					sm.Timer.Reset(time.Duration(alarmAction.Time) * time.Millisecond)
				} else {
					sm.Timer.Reset(time.Duration(alarmAction.Time+uint(r1.Intn(1000))) * time.Millisecond)
				}

			}
		}

	case Send:
		sendAction := act.(Send)
		//		if Debug {
		//				log.Printf("SEND Action by %v sendto = %v event = %v  \n", sm.ID, sendAction.PeerId, sendAction.Event.getEventName())}
		//		if Debug {
		//			log.Printf("%v:: SEND %v -> %v (%v)", time.Now().Nanosecond(), sm.ID, sendAction.PeerId, sendAction.Event.getEventName())
		//		}
		sm.Server.Outbox() <- &cluster.Envelope{Pid: sendAction.PeerId, Msg: sendAction.Event}

	case LogStore:
		logStoreAction := act.(LogStore)
		//		if Debug {
		//			log.Printf("%v:: LOGSTORE Action by %v and index = %v and term = %v  sm.LogDir = %v \n", time.Now().Nanosecond(), sm.ID, logStoreAction.Index, logStoreAction.Term, sm.LogDir)
		//		}
		lg, err := diskLog.Open(sm.LogDir)
		lg.RegisterSampleEntry(LogStore{})
		if err != nil {
			panic(err)
		}

		defer lg.Close()

		err = lg.Append(logStoreAction)
		if err != nil {
			panic(err)
		}

		//
		//		lg, err := diskLog.Open(sm.LogDir)
		//		if err != nil {
		//			panic(err)
		//		}
		//		lg.RegisterSampleEntry(LogEntry{})
		//		err = lg.Append(logStoreAction)
		//		if err != nil {
		//			panic(err)
		//		}
		//		defer lg.Close()

	case StateStore:
		stateStoreAction := act.(StateStore)
		//		if Debug {
		//		log.Printf("STATESTORE Action BY %v  term = %v  votedfor = %v \n", sm.ID, stateStoreAction.Term, stateStoreAction.VotedFor)
		lg, err := diskLog.Open(sm.StateStoreDir)
		if err != nil {
			panic(err)
		}
		lg.RegisterSampleEntry(StateStore{})
		err = lg.Append(stateStoreAction)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

	case Commit:
		commitAction := act.(Commit)
		//		if Debug {
		//			log.Printf("%v:: %%^^^&&&**################################## COMMIT Action by %v and index = %v \n", time.Now().Nanosecond(), sm.ID, commitAction.Index)
		//		}
		// send the response back to filehandler
		sm.ClientCommitChannel <- commitAction

		//DOUBT do we have store the log entries into the log.. i guess they would already be present in the log
		// when they the append event was raised by client
	}
	//}
}

func RegisterEvents() {
	gob.Register(AppendEntriesReqEvent{})
	gob.Register(AppendEntriesRespEvent{})
	gob.Register(VoteReqEvent{})
	gob.Register(VoteRespEvent{})
	gob.Register(TimeoutEvent{})
	gob.Register(AppendEvent{})
	gob.Register(LogEntry{})
	gob.Register(StateStore{})
}
