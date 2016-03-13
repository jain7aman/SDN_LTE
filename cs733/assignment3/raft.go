package main

import (
	"encoding/gob"
	"github.com/cs733-iitb/cluster"
	diskLog "github.com/cs733-iitb/log"
	"log"
	"math/rand"
	"strconv"
	"time"
)

var s1 rand.Source = rand.NewSource(time.Now().UnixNano())
var r1 *rand.Rand = rand.New(s1)

func Initialize(numServers int) []Node {
	servers := make([]Node, numServers)
	configs := make([]Config, numServers)
	//	s1 := rand.NewSource(time.Now().UnixNano())
	//	r1 := rand.New(s1)

	peers := GetConfig()

	for i := 1; i <= numServers; i++ {
		configs[i-1].cluster = peers
		configs[i-1].Id = i
		configs[i-1].LogDir = "server_" + strconv.Itoa(i) + "_log"
		if i == 1 {
			configs[i-1].ElectionTimeout = 2
			configs[i-1].HeartbeatTimeout = 10
		} else {
			configs[i-1].ElectionTimeout = r1.Intn(400) + 1600
			configs[i-1].HeartbeatTimeout = r1.Intn(100) + 1600
		}
		servers[i-1] = New(configs[i-1])
	}

	return servers
}

func main() {
	numServers := 5
	nodes := Initialize(numServers)
	log.Printf("Registering Events")
	RegisterEvents()
	log.Printf("Starting channels")
		for i := 0; i < numServers; i++ {
	//		log.Printf("i %v \n", i)
			go handleChannels(nodes[i].(*RaftServer))
		}
//	go handleChannels(nodes[0].(*RaftServer))
//	go handleChannels(nodes[1].(*RaftServer))
//	go handleChannels(nodes[2].(*RaftServer))
//	go handleChannels(nodes[3].(*RaftServer))
//	go handleChannels(nodes[4].(*RaftServer))
	
	log.Printf("DONE\n")

}

func handleChannels(sm *RaftServer) {
	log.Printf(" id = %v method ElectionTimeout = %v HeartbeatTimeout = %v \n", sm.Id(), sm.ElectionTimeout, sm.HeartbeatTimeout)
	for {
		select {
		case ev := <-sm.SendChannel:
			handleActions(sm, ev)
		case event := <-sm.Server.Inbox():
			handleEvents(sm, event)
		}
	}
}

func handleEvents(sm *RaftServer, event *cluster.Envelope) {

	//event := <-sm.Server.Inbox()

	switch event.Msg.(type) {
	case TimeoutEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(TimeoutEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(TimeoutEvent)
	case VoteReqEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(VoteReqEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(VoteReqEvent)
	case AppendEntriesReqEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(AppendEntriesReqEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(AppendEntriesReqEvent)
	case AppendEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(AppendEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(AppendEvent)
	case AppendEntriesRespEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(AppendEntriesRespEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(AppendEntriesRespEvent)
	case VoteRespEvent:
		log.Printf("In Handle Events %v \n", event.Msg.(VoteRespEvent).getEventName())
		sm.ReceiveChannel <- event.Msg.(VoteRespEvent)
	}

}

func handleActions(sm *RaftServer, act Action) {

	//for act := range sm.SendChannel {
	switch act.(type) {
	case Alarm:
		alarmAction := act.(Alarm)
		log.Printf("ALARM Action by %v and time = %v \n", sm.ID, alarmAction.Time)
		if !sm.TimerSet {
			sm.TimerSet = true
			sm.Timer = time.AfterFunc(time.Duration(alarmAction.Time)*time.Millisecond, func() { sm.ReceiveChannel <- TimeoutEvent{} })
		} else {
			sm.Timer.Reset(time.Duration(alarmAction.Time) * time.Millisecond)
		}

	case Send:
		sendAction := act.(Send)
		log.Printf("SEND Action by %v sendto = %v event = %v  \n", sm.ID, sendAction.PeerId, sendAction.Event.getEventName())
		sm.Server.Outbox() <- &cluster.Envelope{Pid: sendAction.PeerId, Msg: sendAction.Event}

	case LogStore:
		logStoreAction := act.(LogStore)
		log.Printf("LOGSTORE Action by %v and index = %v and term = %v \n", sm.ID, logStoreAction.Index, logStoreAction.Term)
		lg, err := diskLog.Open(sm.LogDir)
		if err != nil {
			panic(err)
		}
		err = lg.Append(logStoreAction)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

	case StateStore:
		stateStoreAction := act.(StateStore)
		log.Printf("STATESTORE Action BY %v  term = %v  votedfor = %v \n", sm.ID, stateStoreAction.Term, stateStoreAction.VotedFor)
		lg, err := diskLog.Open(sm.StateStoreDir)
		if err != nil {
			panic(err)
		}
		err = lg.Append(stateStoreAction)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

	case Commit:
		commitAction := act.(Commit)
		log.Printf("COMMIT Action by %v and index = %v \n", sm.ID, commitAction.Index)
		// send the response back to filehandler
		sm.ClientCommitChannel <- commitAction

		//DOUBT do we have store the log entries into the log.. i guess they would already be present in the log
		// when they the append event was raised by client
	case NoAction:
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
}
