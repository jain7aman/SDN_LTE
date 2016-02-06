package assignment2

import ()

type State int

const (
	LEADER State = 1 + iota
	FOLLOWER
	CANDIDATE
)

// Structure for a single LOG ENTRY
type LogEntry struct {
	Term    uint   // term in which this log entry was written
	Index   uint   // index of log entry w.r.t whole LOG
	Command []byte // command or data for the state machine
}

// An instance of RAFT SERVER
type RaftServer struct {
	Id          uint       // unique ID of the RAFT Server
	Term        uint       // current term for this server
	State       State      // state of the server, one of LEADER, FOLLOWER or CANDIDATE
	CurrentTerm uint       // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    int        // candidateId that received vote in current term (or null if none)
	Log         []LogEntry // log entries; each entry contains command for state machines, and term when entry was received by leader (first index is 1)
	CommitIndex int        // index of highest log entry known to be committed (initialized to 0 on first boot, increases monotonically)
	LastApplied int        // index of highest log entry applied to state machine (initialized to 0 on first boot, increases monotonically)
	N           uint       // number of RAFT Servers
	LeaderId    uint       // current leader (ID) of the configuration
}

type Event interface {
	eventGenericMethod() // to group all the events into the same category of events
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

func (AppendEntriesReqEvent) eventGenericMethod() {}

// Append Entries RPC Response - Response from another Raft state machine in response to a previous AppendEntriesReq.
type AppendEntriesRespEvent struct {
	Term          uint // currentTerm, for leader to update itself
	Success       bool // true if follower contained entry matching preLogIndex and prevLogTerm
	FollowerId    uint // Id of the follower to let the leader know, which follower to reply to in case append entries response is NOT success
	FollowerIndex uint // index of follower's log. This will help the server to know which entry can be committed if append entries response is successful
}

func (AppendEntriesRespEvent) eventGenericMethod() {}

// Request Vote RPC Request - Message from another Raft state machine to request votes for its candidature.
type VoteReqEvent struct {
	Term         uint // candidate's term
	CandidateId  uint // candidate requesting vote
	LastLogIndex int  // index of candidate's last log entry
	LastLogTerm  uint // term of candidate's last log entry
}

func (VoteReqEvent) eventGenericMethod() {}

// Request Vote RPC Response - Response to a Vote request.
type VoteRespEvent struct {
	Term        uint // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

func (VoteRespEvent) eventGenericMethod() {}

// A timeout event is interpreted according to the state.
// If the state machine is a leader, it is interpreted as a heartbeat timeout,
// and if it is a follower or candidate, it is interpreted as an election timeout.
type TimeoutEvent struct{}

func (TimeoutEvent) eventGenericMethod() {}

// This is a request from the layer above to append the data to the replicated log.
// The response is in the form of an eventual Commit action
type AppendEvent struct {
	Command []byte // command received from the client
}

func (AppendEvent) eventGenericMethod() {}

func (raft *RaftServer) raftMain() {
	raft.State = FOLLOWER // each server starts as a follower
	for {
		switch raft.State {
		case FOLLOWER:
			raft.State = raft.follower()
			break
		case CANDIDATE:
			raft.State = raft.candidate()
			break
		case LEADER:
			raft.State = raft.leader()
			break
		}

	}
}
func Send(peerId uint, event Event) {

}

func Commit(index uint, data []byte, err error) {

}

func Alarm(t uint) {

}

func LogStore(index uint, data []byte) {

}

func (sm *RaftServer) follower() State {
	//Step 1: sets a random timer for the timeout event of follower ****TO BE DONE****
	//Step 2: loops through the events being received from event channel ****TO BE DONE****
	var event Event //some event from event channel (setting Append temporary) ****TO BE DONE****

	switch event.(type) {
	case TimeoutEvent:
		sm.Term++        //increments the term number
		return CANDIDATE //returns to CANDIDATE state

	case VoteReqEvent:
		//Step 1: reset the timer for Timeout event, as the network is alive ****TO BE DONE****
		msg := event.(VoteReqEvent)
		//check if candidates log is as upto to date as follower's term
		if sm.Term <= msg.Term && (sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId)) && len(sm.Log) <= msg.LastLogIndex {
			sm.Term = msg.Term
			sm.VotedFor = int(msg.CandidateId)
			Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: true}) // Give the Vote
		} else {
			Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}) // Reject the Vote
		}

	case AppendEntriesReqEvent:
		//Step 1: reset the timer for Timeout event, as the network is alive ****TO BE DONE****
		msg := event.(AppendEntriesReqEvent)

		if msg.Term < sm.Term { //Followers term is greater than leader's term
			Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))}) //Leader is old and not uptodate
			//continue //continue the events loop ****TO BE DONE****
		}

		if len(sm.Log) > 0 {
			//Reply false if log doesn't contain an entry  at prevLogIndex whos eterm matches prevLogIndex
			if sm.Log[len(sm.Log)-1].Term == msg.PrevLogTerm && len(sm.Log)-1 != msg.PrevLogIndex {
				Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))})
				//continue //continue the events loop ****TO BE DONE****
			}

			//If existing entry conflicts with a new one(same index but different terms),
			// delete the existing entry and all that follows it
			if len(sm.Log)-1 == msg.PrevLogIndex && sm.Log[len(sm.Log)-1].Term != msg.PrevLogTerm {
				// delete the existing entry and all that follows it
				sm.Log = sm.Log[0:msg.PrevLogIndex]
			}
		}
		//Append any new entries not already in the log
		if len(msg.Entries) > 0 { //not a hearbeat message
			for i := 0; i <= len(msg.Entries); i++ {
				sm.Log[i+msg.PrevLogIndex+1] = msg.Entries[i]
				// still need to check if entries already present then need not run the complete loop ****TO BE DONE****
			}
		}
		sm.Term = msg.Term //update our term (Incase leader's term is higher than ours else it remians the same)
		if msg.LeaderCommit > sm.CommitIndex {
			//set commit index = min(leaderCommitt, index of last new entry)
			sm.CommitIndex = minimum(msg.LeaderCommit, len(sm.Log)-1)
		}
		Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: true, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))}) //send a success response back to leader

	case AppendEvent:
		//if a client contacts a follower, the follower redirects it to the leader ****TO BE DONE****

	}
	return FOLLOWER
}

func (sm *RaftServer) candidate() State {
	sm.VotedFor = int(sm.Id) //votes for self
	votesReceived := 1

	lastLogIndex := -1
	lastLogTerm := uint(0)

	logLength := len(sm.Log)
	if logLength > 0 {
		lastLogTerm = sm.Log[logLength-1].Term
		lastLogIndex = logLength - 1
	}
	//issues vote request RPCs in parallel to other servers in the cluster
	for i := uint(0); i < sm.N; i++ {
		if i != sm.Id {
			go Send(i, VoteReqEvent{CandidateId: sm.Id, LastLogIndex: lastLogIndex, LastLogTerm: lastLogTerm, Term: sm.Term})
		}
	}

	// Sets a random ELECTION timer for the timeout event of candidate ****TO BE DONE****
	// Loops through the events being received from event channel ****TO BE DONE****
	var event Event //some event from event channel (setting Append temporary) ****TO BE DONE****

	switch event.(type) {
	case TimeoutEvent:
		sm.Term++        //increments the term number
		return CANDIDATE //returns to CANDIDATE state

	case VoteRespEvent:
		msg := event.(VoteRespEvent)

		if msg.VoteGranted == true {
			votesReceived++
		}

		if votesReceived > int(sm.N/2) { //received the majority
			sm.LeaderId = sm.Id
			return LEADER //becomes the leader
		}
	case AppendEvent:
		// ---------------------DOUBT-------------------------
		// what to do in this exactly ??
		//can't do anything at this moment, as the State Machine does not know who the current leader is.
	case AppendEntriesReqEvent:
		msg := event.(AppendEntriesReqEvent)

		// If the leader's term is at least as large as the candidates's current term, then
		// the candidate recognizes the leader as legitimate and returns to FOLLOWER state
		if msg.Term >= sm.CurrentTerm {
			sm.LeaderId = msg.LeaderId
			// ---------------------DOUBT-------------------------
			// BUT WHAT ABOUT THE ENTRIES THAT LEADER SENT, DO WE HAVE TO UPDATE THE ENTRIES HERE ITSELF
			// OR LET THE LEADER SEND THE RPC AGAIN TO THIS SERVER (when it has become follower)

			return FOLLOWER //return to follower state
		}

		// If the term in the RPC is smaller than the candidate's term, then the
		// candidate reject the RPC and continues in candidate state
		Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))})
	case VoteReqEvent:
		msg := event.(VoteReqEvent)

		// If the other candidate's term is less than or equal to my term, then Reject the VOTE
		// (For equal case we are rejecting because we have already voted for self)
		if msg.Term <= sm.Term {
			Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}) // Reject the Vote
		} else {
			// In this case since the other candidate is more updated than myself
			// I will update my term and give the candidate my vote if I have not already voted
			// for any other candidate (other than myself) else reject this vote and
			// return to FOLLOWER state
			sm.Term = msg.Term
			// not voted OR already voted for same candidate OR already voted for SELF
			if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) || sm.VotedFor == int(sm.Id) {
				sm.VotedFor = int(msg.CandidateId)
				Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: true}) // Give the vote
			} else {
				Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}) // Reject the Vote
			}
			return FOLLOWER
		}
	}
	return CANDIDATE
}

func (sm *RaftServer) leader() State {
	// declaring the Volatile state on leaders
	nextIndex := make([]uint, sm.N) // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex := make([]int, sm.N) // For each server, index of highest log entry known to be replicated on server (initialized to -1 (since log index starts from 0), increases monotonically)

	prevLogTerm := sm.Term
	if len(sm.Log) > 0 {
		prevLogTerm = sm.Log[len(sm.Log)-1].Term
	}

	//Setup the hearbeat timer TIMEOUT event ****TO BE DONE****

	// initialize the Volatile state on leaders and send initial empty AppendEntriesRPC (heartbeat)
	// to each server
	for i := uint(0); i < sm.N; i++ {
		nextIndex[i] = uint(len(sm.Log))
		matchIndex[i] = -1
		if i != sm.Id { //send to all except self
			Send(sm.Id, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommitIndex, LeaderId: sm.Id, PrevLogIndex: int(nextIndex[i]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term})
		}
	}

	// code to send heartbeat messages at periodic intervals if no AppendEntries RPC is there
	// Loops through the events being received from event channel ****TO BE DONE****
	var event Event //some event from event channel (setting Append temporary) ****TO BE DONE****

	switch event.(type) {
	case TimeoutEvent: // heartbeat messages
		for i := uint(0); i < sm.N; i++ {
			if i != sm.Id {
				if int(nextIndex[i]) < (len(sm.Log) - 1) { //receiver log is not uptodate
					prevLogTerm = sm.Term
					if len(sm.Log) > 0 && nextIndex[i] > 0 {
						prevLogTerm = sm.Log[nextIndex[i]-1].Term
					}
					// send the entries not present in follower's log to respective follower
					newEntries := sm.Log[nextIndex[i]:]
					Send(sm.Id, AppendEntriesReqEvent{Entries: newEntries, LeaderCommit: sm.CommitIndex, LeaderId: sm.Id, PrevLogIndex: int(nextIndex[i]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term})
				}
			}
		}

	case VoteReqEvent: // This may be the old leader
		msg := event.(VoteReqEvent)

		// check if candidates log is as upto to date as leader's term, if yes then return to FOLLOWER state
		if sm.Term <= msg.Term && (sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId)) && len(sm.Log) <= msg.LastLogIndex {
			sm.Term = msg.Term
			sm.VotedFor = int(msg.CandidateId)
			Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: true}) // Give the Vote
			return FOLLOWER
		} else {
			Send(msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}) // Reject the Vote
		}

	case AppendEvent:
		msg := event.(AppendEvent)

		index := 0
		if len(sm.Log) > 0 {
			index = len(sm.Log)
		}
		// if received a new command from client, append it to log, it will sent to followers in heartbeat
		// messages or in AppendEntriesRespEvent
		sm.Log = append(sm.Log, LogEntry{Command: msg.Command, Index: uint(index), Term: sm.Term})

	case AppendEntriesReqEvent: // may be this leader is old leader and new leader has alreday been elected
		msg := event.(AppendEntriesReqEvent)

		if sm.Term < msg.Term { // leader is outdated
			sm.Term = msg.Term
			sm.LeaderId = msg.LeaderId
			// send a fail response to new leader, it will automatically bring this server in sync
			// using AppendEntries RPC's when this server changes to FOLLOWER state
			Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))})
			return FOLLOWER
		} else {
			// This case should never occur => two leaders in the system
			// else send a fail AppendEntries response RPC's to let the other leader know it is there
			Send(msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: uint(len(sm.Log))})
		}

	case AppendEntriesRespEvent:
		msg := event.(AppendEntriesRespEvent)

		followerId := msg.FollowerId
		if msg.Success == false {

			if sm.Term < msg.Term { // it implies this leader is outdated to should immediately revert to follower
				sm.Term = msg.Term
				sm.LeaderId = followerId
				return FOLLOWER
			}

			if nextIndex[followerId] > 0 {
				nextIndex[followerId]--
			} else {
				nextIndex[followerId] = 0
			}

			// continue // continue the event loop ****TO BE DONE****
		}

		//update the match index for this follower
		if msg.FollowerIndex > 0 {
			matchIndex[followerId] = int(msg.FollowerIndex)
		}
		// now check to see if an index entry has been replicated on majority of servers
		// if yes then commit the entry and send the response back to client
		min := sm.CommitIndex
		count := 0
		first := true
		for i := uint(0); i < sm.N; i++ {
			if matchIndex[i] > sm.CommitIndex {
				count++
				if first == true {
					first = false
					min = matchIndex[i]
				} else {
					min = minimum(min, matchIndex[i]) //check this later.. if it is correct]
				}
			}
		}

		if count > int(sm.N/2) {
			sm.CommitIndex = min
		}
	}

	return LEADER
}

func minimum(x int, y int) int {
	if x < y {
		return x
	}
	return y
}
