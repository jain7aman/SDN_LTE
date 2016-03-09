package main

import (
	"fmt"
)

type State int

var NumChannels uint = 20
//var RandomWaitTime uint = 4

const (
	LEADER State = 1 + iota
	FOLLOWER
	CANDIDATE
)

type Config struct {
	cluster          []NetConfig // Information about all servers, including this.
	Id               int         // This node's id. One of the cluster's entries should match.
	LogDir           string      // Log file directory for this node
	ElectionTimeout  int
	HeartbeatTimeout int
}

func New(config Config) RaftServer {
	sm := RaftServer{State: FOLLOWER, Id: uint(config.Id), Configuration: config, N:uint(len(config.cluster)), Term: 0, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	return sm
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
	Id    uint  // unique ID of the RAFT Server
	Term  uint  // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	State State // state of the server, one of LEADER, FOLLOWER or CANDIDATE
	//CurrentTerm 	uint		// latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    int        // candidateId that received vote in current term (or null if none)
	Log         []LogEntry // log entries; each entry contains command for state machines, and term when entry was received by leader (first index is 1)
	CommitIndex int        // index of highest log entry known to be committed (initialized to 0 on first boot, increases monotonically)
	LastApplied int        // index of highest log entry applied to state machine (initialized to 0 on first boot, increases monotonically)
	N           uint       // number of RAFT Servers
	LeaderId    uint       // current leader (ID) of the configuration
	VotesArray  []int      // keep tracks of received votes. votesArray -1 => NOT voted, 0 => Voted NO and 1=> Voted Yes
	//	NextIndex   []uint     // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	//	MatchIndex  []int      // For each server, index of highest log entry known to be replicated on server (initialized to -1 (since log index starts from 0), increases monotonically)
	Configuration  Config      // configuration for the state machine
	SendChannel    chan Action // An channel for sending events to state machines ( raft servers )
	ReceiveChannel chan Event  // An channel for receiving events from state machines ( raft servers )
}

type Event interface {
	getEventName() string // to group all the events into the same category of events
}

// Append Entries RPC Request - Message from another Raft state machine.
type AppendEntriesReqEvent struct {
	Term         uint       // leader's term
	LeaderId     uint       // current leader so followers can redirect clients
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
	FollowerId    uint // Id of the follower to let the leader know, which follower to reply to in case append entries response is NOT success
	FollowerIndex int  // index of follower's log. This will help the server to know which entry can be committed
	// if append entries response is successful. Making it int instead of uint to allow -1 indicating no entry in log
}

func (AppendEntriesRespEvent) getEventName() string {
	return "AppendEntriesRespEvent"
}

// Request Vote RPC Request - Message from another Raft state machine to request votes for its candidature.
type VoteReqEvent struct {
	Term         uint // candidate's term
	CandidateId  uint // candidate requesting vote
	LastLogIndex int  // index of candidate's last log entry
	LastLogTerm  uint // term of candidate's last log entry
}

func (VoteReqEvent) getEventName() string {
	return "VoteReqEvent"
}

// Request Vote RPC Response - Response to a Vote request.
type VoteRespEvent struct {
	Id          uint // Id of the server who is responding to this vote request msg
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

func (sm *RaftServer) NodeStart() {
	for {
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
	}
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

	done := false
	for act := range sm.SendChannel {
		switch act.(type) {
		case NoAction:
			done = true
		}
		if done == true {
			break
		}
		actions = append(actions, act)
	}

	return actions
}

type AppendError struct {
	LeaderId uint
	Prob     string
}

func (e *AppendError) Error() string {
	return fmt.Sprintf("%d - %s", e.LeaderId, e.Prob)
}

type Action interface {
	getActionName() string // to group all the actions into the same category of actions
}

type Send struct {
	PeerId uint
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

type NoAction struct{}

func (NoAction) getActionName() string {
	return "NoAction"
}

func minimum(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func main() {

}
