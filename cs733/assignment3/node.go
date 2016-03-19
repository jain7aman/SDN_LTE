package main

import (
	"fmt"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	diskLog "github.com/cs733-iitb/log"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type State int

var HighTimeout uint = 9999999
var NumChannels uint = 1000
var mutex = &sync.Mutex{}

//var Debug bool = false
//var Debug2 bool = false

//var RandomWaitTime uint = 4

const (
	LEADER State = 1 + iota
	FOLLOWER
	CANDIDATE
	POWEROFF
)

type Config struct {
	cluster          []NetConfig // Information about all servers, including this.
	Id               int         // This node's id. One of the cluster's entries should match.
	LogDir           string      // Log file directory for this node
	ElectionTimeout  int
	HeartbeatTimeout int
	InboxSize        int
	OutboxSize       int
}

type Node interface {
	// Client's message to Raft node
	Append([]byte)
	// A channel for client to listen on. What goes into Append must come out of here at some point.
	CommitChannel() <-chan CommitInfo
	// Last known committed index in the log.
	CommittedIndex() int //This could be -1 until the system stabilizes.
	// Returns the data at a log index, or an error.
	Get(index int) (error, []byte)
	// Node's id
	Id() int // Id
	// Id of leader. -1 if unknown
	LeaderId() int
	// Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
	Shutdown()
}

func createMockCluster(configFile string) (*cluster.Config, *mock.MockCluster) {
	cfg, err := cluster.ToConfig(configFile)
	if err != nil {
		panic(err)
	}
	c1, err := mock.NewCluster(configFile)
	if err != nil {
		panic(err)
	}
	return cfg, c1
}

func NewMock(c1 *mock.MockCluster, serverId int, numServers int, logDir string, stateLogDir string, electionTimeout int, heartbeatTimeout int, startServer bool) Node {
	//	 inits the cluster

	server := c1.Servers[serverId]

	// inits the log
	lg, err := diskLog.Open(logDir)
	if err != nil {
		panic(err)
	}
	defer lg.Close()
	lg.RegisterSampleEntry(LogStore{})

	//read entries from the log
	numPrevLogs := lg.GetLastIndex() // should return a int64 value

	var logArray []LogEntry
	var i int64
	if numPrevLogs != -1 {
		for i = 0; i <= numPrevLogs; i++ {
			data, err := lg.Get(i) // should return the Foo instance appended above
			if err != nil {
				panic(err)
			}
			logg, ok := data.(LogStore)
			if !ok {
				log.Fatal("Failed")
			}
			logArray = append(logArray, LogEntry{Index: logg.Index, Term: logg.Term, Command: logg.Data}) // creates the node's log
		}
	}

	// reads the node-specific file that stores lastVotedFor and term
	stateLog, err := diskLog.Open(stateLogDir)
	if err != nil {
		panic(err)
	}
	defer stateLog.Close()
	stateLog.RegisterSampleEntry(StateStore{}) //registers the data structure to store

	//read entries from the log
	stateIndex := stateLog.GetLastIndex() // should return a int64 value

	var lastVotedFor int = -1
	var term uint = 0

	// if previous state has been stored for this node, it will be at last index
	if stateIndex != -1 {
		data, err := stateLog.Get(i) // should return the Foo instance appended above
		if err != nil {
			panic(err)
		}
		state, ok := data.(StateStore)
		if !ok {
			log.Fatal("Failed")
		}
		lastVotedFor = state.VotedFor
		term = state.Term
	}
	//////////////////////////////////////////////////
	//	term = 0
	//	lastVotedFor = -1
	//////////////////////////////////////////////////
	sm := RaftServer{
		State:            FOLLOWER,
		ID:               serverId,
		ElectionTimeout:  electionTimeout,
		HeartbeatTimeout: heartbeatTimeout,
		Server:           server,
		N:                uint(numServers),
		Term:             term,
		VotedFor:         lastVotedFor,
		LogDir:           logDir,
		StateStoreDir:    stateLogDir,
		Log:              logArray, //logArray, //[]LogEntry{}
		TimerSet:         false,
		VotesArray:       createIntArray(numServers, -1),
		LeaderID:         -1,
		CommitIndex:      -1, //If the node doesn’t store the committed index to a file, it has to be
		//-inferred at startup. That will have to wait until the first few AppendEntry heartbeats have been responded to.
		ReceiveChannel:      make(chan Event, NumChannels),
		SendChannel:         make(chan Action, NumChannels),
		ClientCommitChannel: make(chan CommitInfo, NumChannels),
		QuitChannel:         make(chan bool, 1)}

	if startServer {
		go sm.NodeStart()
	}

	return &sm
}

func GetConfig(configFile string) Config {
	cfg, err := cluster.ToConfig(configFile)
	if err != nil {
		panic(err)
	}

	peers := make([]NetConfig, len(cfg.Peers))
	for i := 0; i < len(cfg.Peers); i++ {
		tmpArray := strings.Split(cfg.Peers[i].Address, ":")
		port, err := strconv.Atoi(tmpArray[1])
		if err != nil {
			panic(err)
		}
		peers[i] = NetConfig{Id: cfg.Peers[i].Id, Host: tmpArray[0], Port: port}
	}

	return Config{cluster: peers, InboxSize: cfg.InboxSize, OutboxSize: cfg.OutboxSize}
}

func New(config Config, startServer bool) Node {
	//	 inits the cluster
	peersArray := make([]cluster.PeerConfig, len(config.cluster))
	for i := 0; i < len(config.cluster); i++ {
		peersArray[i] = cluster.PeerConfig{Id: config.cluster[i].Id, Address: config.cluster[i].Host + ":" + strconv.Itoa(config.cluster[i].Port)}
	}

	server, err := cluster.New(config.Id, cluster.Config{Peers: peersArray, InboxSize: config.InboxSize, OutboxSize: config.OutboxSize})
	//	server, err := cluster.New(config.Id, "src/ss/Cluster_config.json")
	if err != nil {
		panic(err)
	}

	// inits the log
	lg, err := diskLog.Open(config.LogDir)
	if err != nil {
		panic(err)
	}
	defer lg.Close()
	lg.RegisterSampleEntry(LogStore{})

	//read entries from the log
	numPrevLogs := lg.GetLastIndex() // should return a int64 value

	var logArray []LogEntry
	var i int64
	if numPrevLogs != -1 {
		for i = 0; i <= numPrevLogs; i++ {
			data, err := lg.Get(i) // should return the Foo instance appended above
			if err != nil {
				panic(err)
			}
			logg, ok := data.(LogStore)
			if !ok {
				log.Fatal("Failed")
			}
			logArray = append(logArray, LogEntry{Index: logg.Index, Term: logg.Term, Command: logg.Data}) // creates the node's log
		}
	}

	// reads the node-specific file that stores lastVotedFor and term
	stateLog, err := diskLog.Open(strconv.Itoa(config.Id) + "_state")
	if err != nil {
		panic(err)
	}
	defer stateLog.Close()
	stateLog.RegisterSampleEntry(StateStore{}) //registers the data structure to store

	//read entries from the log
	stateIndex := stateLog.GetLastIndex() // should return a int64 value

	var lastVotedFor int = -1
	var term uint = 0

	// if previous state has been stored for this node, it will be at last index
	if stateIndex != -1 {
		data, err := stateLog.Get(i) // should return the Foo instance appended above
		if err != nil {
			panic(err)
		}
		state, ok := data.(StateStore)
		if !ok {
			log.Fatal("Failed")
		}
		lastVotedFor = state.VotedFor
		term = state.Term
	}
	//////////////////////////////////////////////////
	//	term = 0
	//	lastVotedFor = -1
	//////////////////////////////////////////////////
	sm := RaftServer{
		State:            FOLLOWER,
		ID:               config.Id,
		ElectionTimeout:  config.ElectionTimeout,
		HeartbeatTimeout: config.HeartbeatTimeout,
		Server:           server,
		N:                uint(len(config.cluster)),
		Term:             term,
		VotedFor:         lastVotedFor,
		LogDir:           config.LogDir,
		StateStoreDir:    strconv.Itoa(config.Id) + "_state",
		Log:              logArray, //[]LogEntry{}
		TimerSet:         false,
		VotesArray:       createIntArray(len(config.cluster), -1),
		LeaderID:         -1,
		CommitIndex:      -1, //If the node doesn’t store the committed index to a file, it has to be
		//-inferred at startup. That will have to wait until the first few AppendEntry heartbeats have been responded to.
		ReceiveChannel:      make(chan Event, NumChannels),
		SendChannel:         make(chan Action, NumChannels),
		ClientCommitChannel: make(chan CommitInfo, NumChannels),
		QuitChannel:         make(chan bool, 1)}

	if startServer {
		go sm.NodeStart()
	}

	return &sm
}

type NetConfig struct {
	Id   int
	Host string
	Port int
}

// Structure for a single LOG ENTRY
type LogEntry struct {
	Term    uint   // term in which this log entry was written
	Index   uint   // index of log entry w.r.t whole LOG
	Command []byte // command or data for the state machine
}

// An instance of RAFT SERVER
type RaftServer struct {
	ID    int   // unique ID of the RAFT Server
	Term  uint  // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	State State // state of the server, one of LEADER, FOLLOWER or CANDIDATE
	//CurrentTerm		uint // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    int        // candidateId that received vote in current term (or null if none)
	Log         []LogEntry // log entries; each entry contains command for state machines, and term when entry was received by leader (first index is 1)
	CommitIndex int        // index of highest log entry known to be committed (initialized to 0 on first boot, increases monotonically)
	LastApplied int        // index of highest log entry applied to state machine (initialized to 0 on first boot, increases monotonically)
	N           uint       // number of RAFT Servers
	LeaderID    int        // current leader (ID) of the configuration
	VotesArray  []int      // keep tracks of received votes. votesArray -1 => NOT voted, 0 => Voted NO and 1=> Voted Yes
	//	NextIndex		[]uint // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	//	MatchIndex		[]int // For each server, index of highest log entry known to be replicated on server (initialized to -1 (since log index starts from 0), increases monotonically)
	ElectionTimeout     int // ElectionTimeout for the state machine
	HeartbeatTimeout    int // HeartbeatTimeout for the state machine
	Server              cluster.Server
	LogDir              string // Log file directory for this node
	StateStoreDir       string // state store file directory for this node
	Timer               *time.Timer
	TimerSet            bool        // a check to see if the timer is already set or not
	SendChannel         chan Action // An channel for sending events to state machines ( raft servers )
	ReceiveChannel      chan Event  // An channel for receiving events from state machines ( raft servers )
	ClientCommitChannel chan CommitInfo
	QuitChannel         chan bool
}

func (sm *RaftServer) Append(command []byte) {
	sm.ReceiveChannel <- AppendEvent{Command: command}
}

func (sm *RaftServer) CommitChannel() <-chan CommitInfo {
	return sm.ClientCommitChannel
}

func (sm *RaftServer) CommittedIndex() int {
	mutex.Lock()
	index := sm.CommitIndex
	mutex.Unlock()
	return index
}

func (sm *RaftServer) setCommittedIndex(commitIndex int) {
	mutex.Lock()
	sm.CommitIndex = commitIndex
	mutex.Unlock()
}

func (sm *RaftServer) Get(index int) (error, []byte) {
	var data []byte
	if (len(sm.Log) - 1) < index {
		return &RetrieveError{Prob: "ERR_INDEX_OUT_OF_RANGE"}, data
	}

	return nil, sm.Log[index].Command
}

func (sm *RaftServer) Id() int {
	return sm.ID
}

func (sm *RaftServer) LeaderId() int {
	return sm.LeaderID
}

func (sm *RaftServer) ClearLogs() {
	if _, err := os.Stat(sm.LogDir); err == nil {
		err = os.RemoveAll(sm.LogDir)
		if err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(sm.StateStoreDir); err == nil {
		err = os.RemoveAll(sm.StateStoreDir)
		if err != nil {
			panic(err)
		}
	}
	/*
		// truncate log store dir
		lg, err := diskLog.Open(sm.LogDir)
		if err != nil {
			panic(err)
		}
		defer lg.Close()
		err = lg.TruncateToEnd(0)
		if err != nil {
			panic(err)
		}

		// truncate state store dir
		stateLog, err := diskLog.Open(sm.StateStoreDir)
		if err != nil {
			panic(err)
		}
		defer stateLog.Close()
		err = stateLog.TruncateToEnd(0)
		if err != nil {
			panic(err)
		}
	*/
}

func (sm *RaftServer) Shutdown() {
	//shut down all the go routines and cancel timers
	sm.ReceiveChannel <- ShutDownEvent{Id: sm.ID}
	time.Sleep(1 * time.Second)

	//get the last log index and flush the log upto the commit index
	//	lg, err := diskLog.Open(sm.LogDir)
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer lg.Close()
	//	lg.RegisterSampleEntry(LogStore{})
	//
	//	//read entries from the log
	//	numPrevLogs := lg.GetLastIndex() // should return a int64 value
	//	data, err := lg.Get(numPrevLogs) // should return the Foo instance appended above
	//	if err != nil {
	//		log.Printf("Node ID = %v\n",sm.Id())
	//		panic(err)
	//	}
	//	log, ok := data.(LogStore)
	//	assert(ok)
	//
	//	if sm.CommittedIndex() > int(log.Index) {
	//		for i := int(log.Index) + 1; i <= sm.CommittedIndex(); i++ {
	//			lg.Append(LogEntry{Index: sm.Log[i].Index, Term: sm.Log[i].Term, Command: sm.Log[i].Command})
	//		}
	//	}

	//close the socket
	sm.Server.Close()
	return
}

func (sm *RaftServer) getState() string {
	if sm.State == LEADER {
		return "LEADER"
	}
	if sm.State == CANDIDATE {
		return "CANDIDATE"
	}
	if sm.State == FOLLOWER {
		return "FOLLOWER"
	}
	return "POWEROFF"
}

func (sm *RaftServer) NodeStart() {
	done := false
	for {
		//		if Debug {
		//			log.Printf("Server ID: %v State : %v Term: %v \n", sm.Id(), sm.getState(), sm.Term)
		//		}
		switch sm.State {
		case FOLLOWER:
			sm.State = sm.follower()
			break
		case CANDIDATE:
			sm.State = sm.candidate()
			break
		case LEADER:
			sm.State = sm.leader()
			break
		case POWEROFF:
			done = true
			break
		}

		if done {
			break
		}
	}
}

type Event interface {
	getEventName() string // to group all the events into the same category of events
}

// Append Entries RPC Request - Message from another Raft state machine.
type AppendEntriesReqEvent struct {
	Term         uint       // leader's term
	LeaderId     int        // current leader so followers can redirect clients
	PrevLogIndex int        // index of the log entry immediately preceeding new ones
	PrevLogTerm  uint       // term of prevLogIndex entry
	Entries      []LogEntry // log entries to store (empty for heartbeat; may send more than one for efficiency
	LeaderCommit int        // leader's commitIndex
}

func (AppendEntriesReqEvent) getEventName() string {
	return "AppendEntriesReqEvent"
}

// Append Entries RPC Response - Response from another Raft state machine in response to a previous AppendEntriesReq.
type AppendEntriesRespEvent struct {
	Term          uint // currentTerm, for leader to update itself
	Success       bool // true if follower contained entry matching preLogIndex and prevLogTerm
	FollowerId    int  // Id of the follower to let the leader know, which follower to reply to in case append entries response is NOT success
	FollowerIndex int  // index of follower's log. This will help the server to know which entry can be committed
	// if append entries response is successful. Making it int instead of uint to allow -1 indicating no entry in log
}

func (AppendEntriesRespEvent) getEventName() string {
	return "AppendEntriesRespEvent"
}

// Request Vote RPC Request - Message from another Raft state machine to request votes for its candidature.
type VoteReqEvent struct {
	Term         uint // candidate's term
	CandidateId  int  // candidate requesting vote
	LastLogIndex int  // index of candidate's last log entry
	LastLogTerm  uint // term of candidate's last log entry
}

func (VoteReqEvent) getEventName() string {
	return "VoteReqEvent"
}

// Request Vote RPC Response - Response to a Vote request.
type VoteRespEvent struct {
	Id          int  // Id of the server who is responding to this vote request msg
	Term        uint // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

func (VoteRespEvent) getEventName() string {
	return "VoteRespEvent"
}

// A timeout event is interpreted according to the state.
// If the state machine is a leader, it is interpreted as a heartbeat timeout,
// and if it is a follower or candidate, it is interpreted as an election timeout.
type TimeoutEvent struct {
	Time uint //time after which timeout should occur
}

func (TimeoutEvent) getEventName() string {
	return "TimeoutEvent"
}

// This is a request from the layer above to append the data to the replicated log.
// The response is in the form of an eventual Commit action
type AppendEvent struct {
	Command []byte // command received from the client
}

func (AppendEvent) getEventName() string {
	return "AppendEvent"
}

type ShutDownEvent struct {
	Id int // Id of the server
}

func (ShutDownEvent) getEventName() string {
	return "ShutDownEvent"
}

func (sm *RaftServer) ProcessEvent(ev Event) (actions []Action) {
	sm.ReceiveChannel <- ev

	switch sm.State {
	case FOLLOWER:
		sm.State = sm.follower()
		break
	case CANDIDATE:
		sm.State = sm.candidate()
		break
	case LEADER:
		sm.State = sm.leader()
		break
	}

	//	done := false
	//	for act := range sm.SendChannel {
	//		switch act.(type) {
	//		case NoAction:
	//			done = true
	//		}
	//		if done == true {
	//			break
	//		}
	//		actions = append(actions, act)
	//	}

	return actions
}

type AppendError struct {
	LeaderId int
	Prob     string
}

func (e *AppendError) Error() string {
	return fmt.Sprintf("%d - %s", e.LeaderId, e.Prob)
}

type RetrieveError struct {
	Prob string
}

func (e *RetrieveError) Error() string {
	return fmt.Sprintf("%s", e.Prob)
}

type Action interface {
	getActionName() string // to group all the actions into the same category of actions
}

type CommitInfo interface {
	commitGenericMethod() // to group all the actions into the same category of actions
}

type Send struct {
	PeerId int
	Event  Event
}

func (Send) getActionName() string {
	return "Send"
}

type Commit struct {
	Index uint
	Data  []byte
	Err   error
}

func (Commit) commitGenericMethod() {}

func (Commit) getActionName() string {
	return "Commit"
}

type Alarm struct {
	Time uint
}

func (Alarm) getActionName() string {
	return "Alarm"
}

type LogStore struct {
	Index uint
	Term  uint
	Data  []byte
}

func (LogStore) getActionName() string {
	return "LogStore"
}

type StateStore struct {
	Term     uint
	VotedFor int
}

func (StateStore) getActionName() string {
	return "StateStore"
}

func minimum(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func createIntArray(N int, initialValue int) []int {
	var tmpArray []int
	for i := 0; i < N; i++ {
		tmpArray = append(tmpArray, initialValue)
	}
	return tmpArray
}
