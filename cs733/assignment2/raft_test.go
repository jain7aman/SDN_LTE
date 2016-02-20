package main

import (
	"testing"
	"fmt"
)

/*************************************************************************************
*																					 *
*								FOLLOWER TEST CASES									 *
*																					 *
**************************************************************************************/

// Testing of timeout event in case of follower
func Test_FollowerTimeoutEvent(t *testing.T) {
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 0, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = TimeoutEvent{}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_FollowerTimeoutEvent_1")
	expect(t, int(sm.Term), 1, "Test_FollowerTimeoutEvent_2")
	expectActions(t, actions, []Action{Alarm{}, StateStore{Term: sm.Term, VotedFor: -1}}, true, false, "Test_FollowerTimeoutEvent_3")
}

func Test_FollowerVoteReqEvent(t *testing.T) {
	//Candidate's term is less than follower's term
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 2, VotedFor: -1, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = VoteReqEvent{CandidateId: 2, Term: 1}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_1")
	expect(t, int(sm.Term), 2, "Test_FollowerVoteReqEvent_2")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_FollowerVoteReqEvent_3")

	// Candidate's term is more than follower's term and Candidates log is shorter than follower's log
	// Vote should be rejected in this scenario
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 2, Command: []byte("Msg_ID:2@@Command2")})
	ev = VoteReqEvent{CandidateId: 2, Term: 3, LastLogIndex: 0, LastLogTerm: 1}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_4")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteReqEvent_5")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, StateStore{Term: 3, VotedFor: -1}, Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_FollowerVoteReqEvent_6")

	// Candidate's term is equal to follower's term and Candidates log is longer than follower's log
	// Vote should be given in this scenario
	ev = VoteReqEvent{CandidateId: 2, Term: 3, LastLogIndex: 2, LastLogTerm: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_7")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteReqEvent_8")
	expect(t, int(sm.VotedFor), 2, "Test_FollowerVoteReqEvent_9")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, StateStore{Term: 3, VotedFor: 2}, Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}}}, true, true, "Test_FollowerVoteReqEvent_10")

	//same candidate asking for vote again, should be given vote
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_7")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteReqEvent_8")
	expect(t, int(sm.VotedFor), 2, "Test_FollowerVoteReqEvent_9")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, StateStore{Term: 3, VotedFor: 2}, Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}}}, true, true, "Test_FollowerVoteReqEvent_10")

	//different candidate with same term asking for vote, should not be given vote as follower has already voted
	ev = VoteReqEvent{CandidateId: 3, Term: 3, LastLogIndex: 2, LastLogTerm: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_11")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteReqEvent_12")
	expect(t, int(sm.VotedFor), 2, "Test_FollowerVoteReqEvent_13")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, Send{3, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_FollowerVoteReqEvent_14")

	//different candidate with higher term asking for vote, should not be given vote as follower has already voted
	// but term should be updated and votedfor should be -1 because term has changed
	ev = VoteReqEvent{CandidateId: 3, Term: 4, LastLogIndex: 3, LastLogTerm: 4}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteReqEvent_15")
	expect(t, int(sm.Term), 4, "Test_FollowerVoteReqEvent_16")
	expect(t, int(sm.VotedFor), -1, "Test_FollowerVoteReqEvent_17")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, StateStore{Term: 4, VotedFor: -1}, Send{3, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_FollowerVoteReqEvent_18")
}

func Test_FollowerAppendEntriesReqEvent(t *testing.T) {
	// Followers term is greater than leader's term, reject the RPC
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 2, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	ev := AppendEntriesReqEvent{Term: 1, LeaderId: 2}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesReqEvent_1")
	expect(t, int(sm.Term), 2, "Test_FollowerAppendEntriesReqEvent_2")
	expectActions(t, actions, []Action{Alarm{}, Send{2, AppendEntriesRespEvent{Term: 2, FollowerId: sm.Id, Success: false, FollowerIndex: -1}}}, true, true, "Test_FollowerAppendEntriesReqEvent_3")

	// Followers term is less than leader's term but followers log index is less than PrevLogIndex of Leader, reject the RPC
	ev = AppendEntriesReqEvent{Term: 3, LeaderId: 2, PrevLogIndex: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesReqEvent_4")
	expect(t, int(sm.Term), 3, "Test_FollowerAppendEntriesReqEvent_5")
	expectActions(t, actions, []Action{Alarm{}, StateStore{Term: 3, VotedFor: -1}, Alarm{}, Send{2, AppendEntriesRespEvent{Term: 3, FollowerId: sm.Id, Success: false, FollowerIndex: -1}}}, true, true, "Test_FollowerAppendEntriesReqEvent_6")

	// Followers term is same as leader's term but followers previous log term does not match leader's PrevLogTerm, reject the RPC
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	ev = AppendEntriesReqEvent{Term: 3, LeaderId: 2, PrevLogIndex: 1, PrevLogTerm: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesReqEvent_7")
	expect(t, int(sm.Term), 3, "Test_FollowerAppendEntriesReqEvent_8")
	expectActions(t, actions, []Action{Alarm{}, Alarm{}, Send{2, AppendEntriesRespEvent{Term: 3, FollowerId: sm.Id, Success: false, FollowerIndex: 1}}}, true, true, "Test_FollowerAppendEntriesReqEvent_9")

	// Followers term is less than leader's term but followers previous log term matches leader's PrevLogTerm
	// check if the entries to be appended is already present in the log at PrevLogIndex+1+i
	// if present then ignore the entry else delete that entry and all that follows it
	// and starts appending the new entries send by leader
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 2, Command: []byte("Msg_ID:3@@Command3")})
	sm.Log = append(sm.Log, LogEntry{Index: 3, Term: 2, Command: []byte("Msg_ID:4@@Command4")})
	sm.Log = append(sm.Log, LogEntry{Index: 4, Term: 2, Command: []byte("Msg_ID:5@@Command5")})
	sm.CommitIndex = 2

	var leadersAppendEntries []LogEntry
	//leadersAppendEntries = append(leadersAppendEntries, LogEntry{Index:2, Term:2, Command:[]byte("Msg_ID:3@@Command3")})
	leadersAppendEntries = append(leadersAppendEntries, LogEntry{Index: 3, Term: 2, Command: []byte("Msg_ID:4@@Command4")})
	leadersAppendEntries = append(leadersAppendEntries, LogEntry{Index: 4, Term: 3, Command: []byte("Msg_ID:51@@Command51")})
	leadersAppendEntries = append(leadersAppendEntries, LogEntry{Index: 5, Term: 4, Command: []byte("Msg_ID:6@@Command6")})

	ev = AppendEntriesReqEvent{Term: 4, LeaderId: 2, PrevLogIndex: 2, PrevLogTerm: 2, Entries: leadersAppendEntries, LeaderCommit: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesReqEvent_10")
	expect(t, int(sm.Term), 4, "Test_FollowerAppendEntriesReqEvent_11")
	expect(t, sm.CommitIndex, 3, "Test_FollowerAppendEntriesReqEvent_12")
	expectActions(t, actions, []Action{Alarm{}, StateStore{Term: 4, VotedFor: -1}, Alarm{},
		LogStore{Index: 4, Term: 3, Data: []byte("Msg_ID:51@@Command51")},
		LogStore{Index: 5, Term: 4, Data: []byte("Msg_ID:6@@Command6")},
		Send{2, AppendEntriesRespEvent{Term: 4, FollowerId: sm.Id, Success: true, FollowerIndex: 5}}}, true, true, "Test_FollowerAppendEntriesReqEvent_13")

}

func Test_FollowerAppendEvent(t *testing.T) {
	//if a client contacts a follower, it redirects it to the leader by sending commit action with error ERR_REDIRECTION
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 2, LeaderId: 2, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	clientAppendEntry := []byte("Msg_ID:3@@Command3")
	ev := AppendEvent{Command: clientAppendEntry}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEvent_1")
	expect(t, int(sm.Term), 2, "Test_FollowerAppendEvent_2")
	expectActions(t, actions, []Action{Alarm{}, Commit{Data: clientAppendEntry, Err: &AppendError{LeaderId: sm.LeaderId, Prob: "ERR_REDIRECTION"}}}, true, true, "Test_FollowerAppendEvent_3")
}

func Test_FollowerAppendEntriesRespEvent(t *testing.T) {
	// If follower receives AppendEntriesRespEvent and event term is higher than its own
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 2, LeaderId: 2, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	ev := AppendEntriesRespEvent{Term: 3}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesRespEvent_1")
	expect(t, int(sm.Term), 3, "Test_FollowerAppendEntriesRespEvent_2")
	expect(t, int(sm.VotedFor), -1, "Test_FollowerAppendEntriesRespEvent_3")
	expectActions(t, actions, []Action{Alarm{}, StateStore{Term: 3, VotedFor: -1}}, true, true, "Test_FollowerAppendEntriesRespEvent_4")

	// If follower receives AppendEntriesRespEvent and event term is NOT higher than its own
	ev = AppendEntriesRespEvent{Term: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerAppendEntriesRespEvent_5")
	expect(t, int(sm.Term), 3, "Test_FollowerAppendEntriesRespEvent_6")
	expect(t, int(sm.VotedFor), -1, "Test_FollowerAppendEntriesRespEvent_7")
	expectActions(t, actions, []Action{Alarm{}}, true, true, "Test_FollowerAppendEntriesRespEvent_8")
}

func Test_FollowerVoteRespEvent(t *testing.T) {
	// If follower receives VoteRespEvent and event term is higher than its own
	var sm = &RaftServer{State: FOLLOWER, Id: 0, Term: 2, LeaderId: 2, ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	ev := VoteRespEvent{Term: 3}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteRespEvent_1")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteRespEvent_2")
	expect(t, int(sm.VotedFor), -1, "Test_FollowerVoteRespEvent_3")
	expectActions(t, actions, []Action{Alarm{}, StateStore{Term: 3, VotedFor: -1}}, true, true, "Test_FollowerVoteRespEvent_4")

	// If follower receives VoteRespEvent and event term is NOT higher than its own
	ev = VoteRespEvent{Term: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_FollowerVoteRespEvent_5")
	expect(t, int(sm.Term), 3, "Test_FollowerVoteRespEvent_6")
	expect(t, int(sm.VotedFor), -1, "Test_FollowerVoteRespEvent_7")
	expectActions(t, actions, []Action{Alarm{}}, true, true, "Test_FollowerVoteRespEvent_8")
}

/*************************************************************************************
*																					 *
*								CANDIDATE TEST CASES								 *
*																					 *
**************************************************************************************/

// Testing of timeout event in case of Candidate
func Test_CandidateTimeoutEvent(t *testing.T) {
	var sm = &RaftServer{State: CANDIDATE, Id: 0, N: 5, Term: 1, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	var ev = TimeoutEvent{}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_CandidateTimeoutEvent_1")
	expect(t, int(sm.Term), 2, "Test_CandidateTimeoutEvent_2") // term was increamented in follower state before it changed to candidate
	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: 1, LastLogTerm: 1}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: 1, LastLogTerm: 1}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: 1, LastLogTerm: 1}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: 1, LastLogTerm: 1}},
		StateStore{Term: 2, VotedFor: -1}}, true, true, "Test_CandidateTimeoutEvent_3")
}

func Test_VoteRespEvent_1(t *testing.T) {
	// Vote Rejected because the Follower's term is higher, Candidate should become follower
	var sm = &RaftServer{State: CANDIDATE, Id: 0, N: 5, Term: 1, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = VoteRespEvent{Id: 1, Term: 2, VoteGranted: false}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_VoteRespEvent_1_1")
	expect(t, int(sm.Term), 2, "Test_VoteRespEvent_1_2") // term was increamented in follower state before it changed to candidate
	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		StateStore{Term: 2, VotedFor: -1}}, true, true, "Test_VoteRespEvent_1_3")
}

// In this function CANDIDATE receives NO vote from majority, it should turn to FOLLOWER
func Test_VoteRespEvent_2(t *testing.T) {
	// Vote Rejected but the term is same, Candidate should remain Candidate
	var sm = &RaftServer{State: CANDIDATE, Id: 0, N: 5, Term: 1, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = VoteRespEvent{Id: 1, Term: 1, VoteGranted: false}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_VoteRespEvent_2_1")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_2_2") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_2_3")  // Should have voted for self
	expect(t, sm.VotesArray[1], 0, "Test_VoteRespEvent_2_4")  // Vote should be rejected by follower with id 1
	expect(t, sm.VotesArray[2], -1, "Test_VoteRespEvent_2_5") // Vote should not have been casted for follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_2_6") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_2_7") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_2_8")

	//Receives another Rejection Vote from same Follower, Votes reject count should not increase
	ev = VoteRespEvent{Id: 1, Term: 1, VoteGranted: false}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_VoteRespEvent_2_9")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_2_10") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_2_11")  // Should have voted for self
	expect(t, sm.VotesArray[1], 0, "Test_VoteRespEvent_2_12")  // Vote should be rejected by follower with id 1
	expect(t, sm.VotesArray[2], -1, "Test_VoteRespEvent_2_13") // Vote should not have been casted for follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_2_14") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_2_15") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_2_16")

	//Receives another Rejection Vote from different Follower, Should remain CANDIDATE
	ev = VoteRespEvent{Id: 2, Term: 1, VoteGranted: false}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_VoteRespEvent_2_17")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_2_18") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_2_19")  // Should have voted for self
	expect(t, sm.VotesArray[1], 0, "Test_VoteRespEvent_2_20")  // Vote should be rejected by follower with id 1
	expect(t, sm.VotesArray[2], 0, "Test_VoteRespEvent_2_21")  // Vote should be rejected by follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_2_22") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_2_23") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_2_24")

	//Receives another Rejection Vote from different Follower, Lost the majority, should become FOLLOWER
	ev = VoteRespEvent{Id: 3, Term: 1, VoteGranted: false}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_VoteRespEvent_2_25")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_2_26") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_2_27")  // Should have voted for self
	expect(t, sm.VotesArray[1], 0, "Test_VoteRespEvent_2_28")  // Vote should be rejected by follower with id 1
	expect(t, sm.VotesArray[2], 0, "Test_VoteRespEvent_2_29")  // Vote should be rejected by follower with id 2
	expect(t, sm.VotesArray[3], 0, "Test_VoteRespEvent_2_30")  // Vote should be rejected by follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_2_31") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_2_32")

}

// In this function CANDIDATE receives YES vote from majority, it should turn to LEADER
func Test_VoteRespEvent_3(t *testing.T) {
	// Vote Accepted but the term is same, Candidate should remain Candidate
	var sm = &RaftServer{State: CANDIDATE, Id: 0, N: 5, Term: 1, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = VoteRespEvent{Id: 1, Term: 1, VoteGranted: true}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_VoteRespEvent_3_1")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_3_2") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_3_3")  // Should have voted for self
	expect(t, sm.VotesArray[1], 1, "Test_VoteRespEvent_3_4")  // Vote by follower with id 1
	expect(t, sm.VotesArray[2], -1, "Test_VoteRespEvent_3_5") // Vote should not have been casted for follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_3_6") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_3_7") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_3_8")

	//Receives another Acceptance Vote from same Follower, Votes accept count should not increase
	ev = VoteRespEvent{Id: 1, Term: 1, VoteGranted: true}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_VoteRespEvent_3_9")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_2_10") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_3_11")  // Should have voted for self
	expect(t, sm.VotesArray[1], 1, "Test_VoteRespEvent_3_12")  // Vote by follower with id 1
	expect(t, sm.VotesArray[2], -1, "Test_VoteRespEvent_3_13") // Vote should not have been casted for follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_3_14") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_3_15") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_3_16")

	//Receives another Acceptance Vote from different Follower, Should change to LEADER
	ev = VoteRespEvent{Id: 2, Term: 1, VoteGranted: true}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_VoteRespEvent_3_17")
	expect(t, int(sm.Term), 1, "Test_VoteRespEvent_3_18") // term was increamented in follower state before it changed to candidate
	//0 -> Vote Reject 1-> Vote Accept -1-> NOT Received Vote
	expect(t, sm.VotesArray[0], 1, "Test_VoteRespEvent_3_19")  // Should have voted for self
	expect(t, sm.VotesArray[1], 1, "Test_VoteRespEvent_3_20")  // Vote by follower with id 1
	expect(t, sm.VotesArray[2], 1, "Test_VoteRespEvent_3_21")  // Vote by follower with id 2
	expect(t, sm.VotesArray[3], -1, "Test_VoteRespEvent_3_22") // Vote should not have been casted for follower with id 3
	expect(t, sm.VotesArray[4], -1, "Test_VoteRespEvent_3_23") // Vote should not have been casted for follower with id 4

	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_VoteRespEvent_3_24")
}

func Test_CandidateAppendEvent(t *testing.T) {
	//if a client contacts a Candidate, it redirects by sending commit action with error ERR_REDIRECTION
	var sm = &RaftServer{State: CANDIDATE, Id: 0, Term: 1, N: 5, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	clientAppendEntry := []byte("Msg_ID:3@@Command3")
	ev := AppendEvent{Command: clientAppendEntry}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_CandidateAppendEvent_1")
	expect(t, int(sm.Term), 1, "Test_CandidateAppendEvent_2")
	expectActions(t, actions, []Action{StateStore{Term: 1, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 1, LastLogIndex: -1, LastLogTerm: 0}},
		Commit{Data: clientAppendEntry, Err: &AppendError{Prob: "ERR_NO_LEADER_YET"}}}, true, true, "Test_CandidateAppendEvent_3")
}

func Test_CandidateAppendEntriesReqEvent(t *testing.T) {
	var sm = &RaftServer{State: CANDIDATE, Id: 0, Term: 2, N: 5, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}

	// If Candidate receives AppendEntriesReqEvent and event term is less than its own, remains CANDIDATE
	ev := AppendEntriesReqEvent{Term: 1, LeaderId: 2}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_CandidateAppendEntriesReqEvent_1")
	expect(t, int(sm.Term), 2, "Test_CandidateAppendEntriesReqEvent_2")
	expect(t, int(sm.VotedFor), int(sm.Id), "Test_CandidateAppendEntriesReqEvent_3")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, AppendEntriesRespEvent{Term: sm.Term, FollowerId: sm.Id, Success: false, FollowerIndex: -1}}}, true, true, "Test_CandidateAppendEntriesReqEvent_4")

	// If candidate receives AppendEntriesReqEvent and event term is higher than its own, it updates the term and changes to FOLLOWER
	ev = AppendEntriesReqEvent{Term: 3, LeaderId: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_CandidateAppendEntriesReqEvent_5")
	expect(t, int(sm.Term), 3, "Test_CandidateAppendEntriesReqEvent_6")
	expect(t, int(sm.VotedFor), -1, "Test_CandidateAppendEntriesReqEvent_7")
	expect(t, int(sm.LeaderId), 2, "Test_CandidateAppendEntriesReqEvent_8")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		StateStore{Term: 3, VotedFor: -1}}, true, true, "Test_CandidateAppendEntriesReqEvent_9")
}

func Test_CandidateVoteReqEvent(t *testing.T) {
	var sm = &RaftServer{State: CANDIDATE, Id: 0, Term: 2, N: 5, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}

	// If Candidate receives VoteReqEvent and event term is less than its own, remains CANDIDATE
	ev := VoteReqEvent{Term: 1, CandidateId: 2}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_CandidateVoteReqEvent_1")
	expect(t, int(sm.Term), 2, "Test_CandidateVoteReqEvent_2")
	expect(t, int(sm.VotedFor), int(sm.Id), "Test_CandidateVoteReqEvent_3")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_CandidateVoteReqEvent_4")

	// If candidate receives VoteReqEvent and event term is higher than its own, it updates the term and changes to FOLLOWER
	ev = VoteReqEvent{Term: 3, CandidateId: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_CandidateVoteReqEvent_5")
	expect(t, int(sm.Term), 3, "Test_CandidateVoteReqEvent_6")
	expect(t, int(sm.VotedFor), 2, "Test_CandidateVoteReqEvent_7")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		StateStore{Term: 3, VotedFor: 2},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}}}, true, true, "Test_CandidateVoteReqEvent_8")
}

func Test_CandidateAppendEntriesRespEvent(t *testing.T) {
	var sm = &RaftServer{State: CANDIDATE, Id: 0, Term: 2, N: 5, VotesArray: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}

	// If Candidate receives AppendEntriesRespEvent and event term is less than its own, remains CANDIDATE
	ev := AppendEntriesRespEvent{Term: 1}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, CANDIDATE, "Test_CandidateAppendEntriesRespEvent_1")
	expect(t, int(sm.Term), 2, "Test_CandidateAppendEntriesRespEvent_2")
	expect(t, int(sm.VotedFor), int(sm.Id), "Test_CandidateAppendEntriesRespEvent_3")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}}}, true, true, "Test_CandidateAppendEntriesRespEvent_4")

	// If candidate receives AppendEntriesRespEvent and event term is higher than its own, it updates the term and changes to FOLLOWER
	ev = AppendEntriesRespEvent{Term: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_CandidateAppendEntriesRespEvent_5")
	expect(t, int(sm.Term), 3, "Test_CandidateAppendEntriesRespEvent_6")
	expect(t, int(sm.VotedFor), -1, "Test_CandidateAppendEntriesRespEvent_7")
	expectActions(t, actions, []Action{StateStore{Term: 2, VotedFor: int(sm.Id)}, Alarm{},
		Send{1, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{2, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{3, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		Send{4, VoteReqEvent{CandidateId: sm.Id, Term: 2, LastLogIndex: -1, LastLogTerm: 0}},
		StateStore{Term: 3, VotedFor: -1}}, true, true, "Test_CandidateAppendEntriesRespEvent_8")
}

/*************************************************************************************
*																					 *
*								LEADER TEST CASES									 *
*																					 *
**************************************************************************************/

func Test_LeaderTimeoutEvent_1(t *testing.T) {
	//Leader has started for the first time, It sends empty heartbeat messages for TIMEOUT event
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = TimeoutEvent{}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderTimeoutEvent_1_1")
	expect(t, int(sm.Term), 1, "Test_LeaderTimeoutEvent_1_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}}}, true, true, "Test_LeaderTimeoutEvent_1_3")
}

func Test_LeaderTimeoutEvent_2(t *testing.T) {
	//Leader sends AppendEntriesReqEvent event having logentry's to follower's as per their respective nextIndex
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.NextIndex[1] = 2
	sm.NextIndex[2] = 1
	sm.NextIndex[3] = 1
	var tmpLogEntry1 []LogEntry
	tmpLogEntry1 = append(tmpLogEntry1, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	tmpLogEntry1 = append(tmpLogEntry1, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	var tmpLogEntry2 []LogEntry
	tmpLogEntry2 = append(tmpLogEntry2, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})

	var ev = TimeoutEvent{}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderTimeoutEvent_2_1")
	expect(t, int(sm.Term), 1, "Test_LeaderTimeoutEvent_2_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 1, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 1, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 1, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 1, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 1, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 0, PrevLogTerm: 1, Entries: tmpLogEntry2, LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: 0, PrevLogTerm: 1, Entries: tmpLogEntry2, LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: tmpLogEntry1, LeaderCommit: -1}}}, true, true, "Test_LeaderTimeoutEvent_2_3")
}

func Test_LeaderVoteReqEvent(t *testing.T) {
	//Leader gets VoteReqEvent event with term less than or equal to its term, it rejects the Vote
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = VoteReqEvent{Term: 1, CandidateId: 2}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderVoteReqEvent_1")
	expect(t, int(sm.Term), 1, "Test_LeaderVoteReqEvent_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}}, true, true, "Test_LeaderVoteReqEvent_3")

	//Leader gets VoteReqEvent event with higher term but with less updated log (Same terms of last log but less log lentgth of candidate), it rejects the Vote
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}

	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})

	ev = VoteReqEvent{Term: 2, CandidateId: 2, LastLogTerm: 1, LastLogIndex: 0}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderVoteReqEvent_4")
	expect(t, int(sm.Term), 2, "Test_LeaderVoteReqEvent_5")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: -1},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderVoteReqEvent_6")

	//Leader gets VoteReqEvent event with higher term but with less updated log (different terms of last log), it rejects the Vote
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev = VoteReqEvent{Term: 2, CandidateId: 2, LastLogTerm: 0, LastLogIndex: 0}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderVoteReqEvent_7")
	expect(t, int(sm.Term), 2, "Test_LeaderVoteReqEvent_8")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: -1},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderVoteReqEvent_9")

	//Leader gets VoteReqEvent event with higher term and updated log, it should give the vote and change to follower state
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev = VoteReqEvent{Term: 2, CandidateId: 2, LastLogTerm: 1, LastLogIndex: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderVoteReqEvent_10")
	expect(t, int(sm.Term), 2, "Test_LeaderVoteReqEvent_11")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: 2},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderVoteReqEvent_12")

	//same candidate asks for vote again, already given vote for state machine with id 2, should give the vote again
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: 2, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev = VoteReqEvent{Term: 2, CandidateId: 2, LastLogTerm: 1, LastLogIndex: 2}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderVoteReqEvent_10")
	expect(t, int(sm.Term), 2, "Test_LeaderVoteReqEvent_11")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: 2},
		Send{2, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderVoteReqEvent_12")
}

func Test_LeaderAppendEvent(t *testing.T) {
	//if a client contacts leader for Appending new entries, leader accepts it
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	clientAppendEntry := []byte("Msg_ID:4@@Command4")
	ev := AppendEvent{Command: clientAppendEntry}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEvent_1")
	expect(t, int(sm.Term), 1, "Test_LeaderAppendEvent_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		LogStore{Index: 3, Term: 1, Data: clientAppendEntry}}, true, true, "Test_LeaderAppendEvent_3")
}

func Test_LeaderAppendEntriesReqEvent(t *testing.T) {
	//Leader gets AppendEntriesReqEvent event with term less than or equal to its term, it rejects the request
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 2, CommitIndex: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	var ev = AppendEntriesReqEvent{Term: 1}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEntriesReqEvent_1")
	expect(t, int(sm.Term), 2, "Test_LeaderAppendEntriesReqEvent_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{3, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{4, AppendEntriesReqEvent{Term: sm.Term, LeaderId: sm.Id, PrevLogIndex: -1, PrevLogTerm: 0, Entries: make([]LogEntry, 0), LeaderCommit: -1}},
		Send{2, AppendEntriesRespEvent{Term: 2, Success: false, FollowerId: 0, FollowerIndex: -1}}}, true, true, "Test_LeaderAppendEntriesReqEvent_3")

	//Leader gets AppendEntriesReqEvent event with term higher than its term, it changes to follower
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 2, CommitIndex: 0, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev = AppendEntriesReqEvent{Term: 3}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderAppendEntriesReqEvent_4")
	expect(t, int(sm.Term), 3, "Test_LeaderAppendEntriesReqEvent_5")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 2, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 2, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 2, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 2, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 3, VotedFor: -1},
		Send{2, AppendEntriesRespEvent{Term: 3, Success: false, FollowerId: 0, FollowerIndex: 2}},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderAppendEntriesReqEvent_6")
}

func Test_LeaderAppendEntriesRespEvent_Failed(t *testing.T) {
	//Leaders gets FAILED AppendEntriesRespEvent with term higher than its own, it changes to follower
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev := AppendEntriesRespEvent{Term: 2, FollowerId: 2, Success: false}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderAppendEntriesRespEvent_Failed_1")
	expect(t, int(sm.Term), 2, "Test_LeaderAppendEntriesRespEvent_Failed_2")
	expect(t, int(sm.LeaderId), 2, "Test_LeaderAppendEntriesRespEvent_Failed_3")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: -1},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{LeaderId: 2, Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{LeaderId: 2, Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderAppendEntriesRespEvent_Failed_4")

	//Leaders gets FAILED AppendEntriesRespEvent with term lower than its own, it updates the nextIndex of follower (decrements it by 1)
	// here Follower's nextIndex is > 0 i.e. it has some entries in its log
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 3, LeaderId: 0, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 2, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 3, Command: []byte("Msg_ID:3@@Command3")})
	sm.NextIndex[2] = 2
	ev = AppendEntriesRespEvent{Term: 2, FollowerId: 2, Success: false}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEntriesRespEvent_Failed_5")
	expect(t, int(sm.Term), 3, "Test_LeaderAppendEntriesRespEvent_Failed_6")
	expect(t, int(sm.LeaderId), 0, "Test_LeaderAppendEntriesRespEvent_Failed_7")
	expect(t, int(sm.NextIndex[2]), 1, "Test_LeaderAppendEntriesRespEvent_Failed_8")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}}}, true, true, "Test_LeaderAppendEntriesRespEvent_Failed_9")

	//Leaders gets FAILED AppendEntriesRespEvent with term lower than its own, it updates the nextIndex of follower (decrements it by 1)
	// here Follower's nextIndex is 0 i.e. it has no entry in its log  // This case wont occur in real scenario.. but for safety
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 3, LeaderId: 0, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 2, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 3, Command: []byte("Msg_ID:3@@Command3")})
	sm.NextIndex[2] = 0
	ev = AppendEntriesRespEvent{Term: 2, FollowerId: 2, Success: false}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEntriesRespEvent_Failed_10")
	expect(t, int(sm.Term), 3, "Test_LeaderAppendEntriesRespEvent_Failed_11")
	expect(t, int(sm.LeaderId), 0, "Test_LeaderAppendEntriesRespEvent_Failed_12")
	expect(t, int(sm.NextIndex[2]), 0, "Test_LeaderAppendEntriesRespEvent_Failed_13")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 0}}}, true, true, "Test_LeaderAppendEntriesRespEvent_Failed_14")
}

func Test_LeaderAppendEntriesRespEvent_Success(t *testing.T) {
	//Leaders gets Success AppendEntriesRespEvent
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 3, LeaderId: 0, CommitIndex: 3, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 2, Command: []byte("Msg_ID:3@@Command3")})
	sm.Log = append(sm.Log, LogEntry{Index: 3, Term: 3, Command: []byte("Msg_ID:4@@Command4")})
	sm.Log = append(sm.Log, LogEntry{Index: 4, Term: 3, Command: []byte("Msg_ID:5@@Command5")})
	sm.Log = append(sm.Log, LogEntry{Index: 5, Term: 3, Command: []byte("Msg_ID:6@@Command6")})

	//sm.MatchIndex[0] = 5
	sm.MatchIndex[1] = 4
	sm.MatchIndex[2] = 3
	sm.MatchIndex[3] = 1
	sm.MatchIndex[4] = 5

	ev := AppendEntriesRespEvent{Term: 2, Success: true, FollowerId: 2, FollowerIndex: 4}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEntriesRespEvent_Success_1")
	expect(t, int(sm.Term), 3, "Test_LeaderAppendEntriesRespEvent_Success_2")
	expect(t, int(sm.LeaderId), 0, "Test_LeaderAppendEntriesRespEvent_Success_3")
	expect(t, int(sm.MatchIndex[2]), 4, "Test_LeaderAppendEntriesRespEvent_Success_4") // match index for this follower is updated
	expect(t, int(sm.CommitIndex), 4, "Test_LeaderAppendEntriesRespEvent_Success_5")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 3}},
		Send{2, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 3}},
		Send{3, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 3}},
		Send{4, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 3}},
		Commit{Index: 4, Data: sm.Log[4].Command, Err: nil}}, true, true, "Test_LeaderAppendEntriesRespEvent_Success_6")

	// received success response from another follower
	ev = AppendEntriesRespEvent{Term: 2, Success: true, FollowerId: 3, FollowerIndex: 4}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderAppendEntriesRespEvent_Success_6")
	expect(t, int(sm.Term), 3, "Test_LeaderAppendEntriesRespEvent_Success_7")
	expect(t, int(sm.LeaderId), 0, "Test_LeaderAppendEntriesRespEvent_Success_8")
	expect(t, int(sm.MatchIndex[3]), 4, "Test_LeaderAppendEntriesRespEvent_Success_9") // match index for this follower is updated
	expect(t, int(sm.CommitIndex), 4, "Test_LeaderAppendEntriesRespEvent_Success_10")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 4}},
		Send{2, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 4}},
		Send{3, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 4}},
		Send{4, AppendEntriesReqEvent{Term: 3, LeaderId: sm.Id, PrevLogIndex: 5, PrevLogTerm: 3, Entries: make([]LogEntry, 0), LeaderCommit: 4}}}, true, true, "Test_LeaderAppendEntriesRespEvent_Success_11")
}

func Test_LeaderVoteRespEvent(t *testing.T) {
	//Leaders gets VoteRespEvent with term higher than its own, it changes to follower
	var sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev := VoteRespEvent{Term: 2}
	actions := sm.ProcessEvent(ev)
	expectState(t, sm.State, FOLLOWER, "Test_LeaderVoteRespEvent_1")
	expect(t, int(sm.Term), 2, "Test_LeaderVoteRespEvent_2")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		StateStore{Term: 2, VotedFor: -1},
		Commit{Data: sm.Log[1].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}},
		Commit{Data: sm.Log[2].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}}, true, true, "Test_LeaderVoteRespEvent_3")

	//Leaders gets VoteRespEvent with term not higher than its own
	sm = &RaftServer{State: LEADER, Id: 0, N: 5, Term: 1, CommitIndex: 0, VotedFor: -1, NextIndex: createUnsignedIntIndexArray(5, 0), MatchIndex: createIntArray(5, -1), ReceiveChannel: make(chan Event, NumChannels), SendChannel: make(chan Action, NumChannels)}
	sm.Log = append(sm.Log, LogEntry{Index: 0, Term: 1, Command: []byte("Msg_ID:1@@Command1")})
	sm.Log = append(sm.Log, LogEntry{Index: 1, Term: 1, Command: []byte("Msg_ID:2@@Command2")})
	sm.Log = append(sm.Log, LogEntry{Index: 2, Term: 1, Command: []byte("Msg_ID:3@@Command3")})
	ev = VoteRespEvent{Term: 1}
	actions = sm.ProcessEvent(ev)
	expectState(t, sm.State, LEADER, "Test_LeaderVoteRespEvent_4")
	expect(t, int(sm.Term), 1, "Test_LeaderVoteRespEvent_5")
	expectActions(t, actions, []Action{Alarm{},
		Send{1, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{2, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{3, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}},
		Send{4, AppendEntriesReqEvent{Term: 1, LeaderId: sm.Id, PrevLogIndex: 2, PrevLogTerm: 1, Entries: make([]LogEntry, 0), LeaderCommit: 0}}}, true, true, "Test_LeaderVoteRespEvent_6")
}

// Useful testing function
func expectEvents(t *testing.T, actual Event, expected Event, testId string) bool {
	errorFound := false
	switch expected.(type) {
	case AppendEntriesReqEvent:
		actEvent := actual.(AppendEntriesReqEvent)
		expEvent := expected.(AppendEntriesReqEvent)

		if actEvent.LeaderCommit != expEvent.LeaderCommit {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event LeaderCommit to be %v, found %v", testId, expEvent.LeaderCommit, actEvent.LeaderCommit))
			errorFound = true
			break
		}
		if actEvent.LeaderId != expEvent.LeaderId {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event LeaderId to be %v, found %v", testId, expEvent.LeaderId, actEvent.LeaderId))
			errorFound = true
			break
		}
		if actEvent.PrevLogIndex != expEvent.PrevLogIndex {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event PrevLogIndex to be %v, found %v", testId, expEvent.PrevLogIndex, actEvent.PrevLogIndex))
			errorFound = true
			break
		}
		if actEvent.PrevLogTerm != expEvent.PrevLogTerm {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event PrevLogTerm to be %v, found %v", testId, expEvent.PrevLogTerm, actEvent.PrevLogTerm))
			errorFound = true
			break
		}
		if actEvent.Term != expEvent.Term {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event Term to be %v, found %v", testId, expEvent.Term, actEvent.Term))
			errorFound = true
			break
		}
		if len(actEvent.Entries) != len(expEvent.Entries) {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event len of LogEntries struct to be %v, found %v", testId, len(expEvent.Entries), len(actEvent.Entries)))
			errorFound = true
			//break
		}
		for i := 0; i < len(actEvent.Entries); i++ {
			actLogEntry := actEvent.Entries[i]
			expLogEntry := expEvent.Entries[i]

			if actLogEntry.Index != expLogEntry.Index {
				t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event LogEntry.Index at Log Index %v to be %v, found %v", testId, i, expLogEntry.Index, actLogEntry.Index))
				errorFound = true
				break
			}

			if actLogEntry.Term != expLogEntry.Term {
				t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event LogEntry.Term at Log Index %v to be %v, found %v", testId, i, expLogEntry.Term, actLogEntry.Term))
				errorFound = true
				break
			}

			if string(actLogEntry.Command) != string(expLogEntry.Command) {
				t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesReqEvent Event LogEntry.Command at Log Index %v to be %v, found %v", testId, i, string(expLogEntry.Command), string(actLogEntry.Command)))
				errorFound = true
				break
			}
		}
		if errorFound {
			break
		}
	case AppendEntriesRespEvent:
		actEvent := actual.(AppendEntriesRespEvent)
		expEvent := expected.(AppendEntriesRespEvent)

		if actEvent.FollowerId != expEvent.FollowerId {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesRespEvent Event FollowerId to be %v, found %v", testId, expEvent.FollowerId, actEvent.FollowerId))
			errorFound = true
			break
		}

		if actEvent.FollowerIndex != expEvent.FollowerIndex {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesRespEvent Event FollowerIndex to be %v, found %v", testId, expEvent.FollowerIndex, actEvent.FollowerIndex))
			errorFound = true
			break
		}

		if actEvent.Success != expEvent.Success {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesRespEvent Event Success to be %v, found %v", testId, expEvent.Success, actEvent.Success))
			errorFound = true
			break
		}

		if actEvent.Term != expEvent.Term {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEntriesRespEvent Event Term to be %v, found %v", testId, expEvent.Term, actEvent.Term))
			errorFound = true
			break
		}

	case AppendEvent:
		actEvent := actual.(AppendEvent)
		expEvent := expected.(AppendEvent)

		if string(actEvent.Command) != string(expEvent.Command) {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action AppendEvent Event Command to be %v, found %v", testId, string(expEvent.Command), string(actEvent.Command)))
			errorFound = true
			break
		}

	case TimeoutEvent:
		actEvent := actual.(TimeoutEvent)
		expEvent := expected.(TimeoutEvent)

		if actEvent.Time != expEvent.Time {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action TimeoutEvent Event Time to be %v, found %v", testId, expEvent.Time, actEvent.Time))
			errorFound = true
			break
		}
	case VoteReqEvent:
		actEvent := actual.(VoteReqEvent)
		expEvent := expected.(VoteReqEvent)

		if actEvent.CandidateId != expEvent.CandidateId {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteReqEvent Event CandidateId to be %v, found %v", testId, expEvent.CandidateId, actEvent.CandidateId))
			errorFound = true
			break
		}

		if actEvent.LastLogIndex != expEvent.LastLogIndex {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteReqEvent Event LastLogIndex to be %v, found %v", testId, expEvent.LastLogIndex, actEvent.LastLogIndex))
			errorFound = true
			break
		}

		if actEvent.LastLogTerm != expEvent.LastLogTerm {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteReqEvent Event LastLogTerm to be %v, found %v", testId, expEvent.LastLogTerm, actEvent.LastLogTerm))
			errorFound = true
			break
		}

		if actEvent.Term != expEvent.Term {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteReqEvent Event Term to be %v, found %v", testId, expEvent.Term, actEvent.Term))
			errorFound = true
			break
		}

	case VoteRespEvent:
		actEvent := actual.(VoteRespEvent)
		expEvent := expected.(VoteRespEvent)

		if actEvent.Id != expEvent.Id {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteRespEvent Event Id to be %v, found %v", testId, expEvent.Id, actEvent.Id))
			errorFound = true
			break
		}

		if actEvent.Term != expEvent.Term {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteRespEvent Event Term to be %v, found %v", testId, expEvent.Term, actEvent.Term))
			errorFound = true
			break
		}

		if actEvent.VoteGranted != expEvent.VoteGranted {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action VoteRespEvent Event VoteGranted to be %v, found %v", testId, expEvent.VoteGranted, actEvent.VoteGranted))
			errorFound = true
			break
		}
	}

	return errorFound
}

func expectActions(t *testing.T, actual []Action, expected []Action, detailedAction bool, detailedEvent bool, testId string) {
	errorFound := false
	if len(expected) != len(actual) {
		t.Error(fmt.Sprintf("Test ID %v:: Received different number of actions expected to be %v, found %v", testId, len(expected), len(actual)))
		return
	}
	for i := 0; i < len(expected); i++ {
		if actual[i].getActionName() != expected[i].getActionName() {
			t.Error(fmt.Sprintf("Test ID %v:: Expected Action %v, found %v", testId, expected[i].getActionName(), actual[i].getActionName()))
			break
		} else {
			if detailedAction {
				switch expected[i].(type) {
				case Send:
					actAction := actual[i].(Send)
					expAction := expected[i].(Send)

					if actAction.Event.getEventName() != expAction.Event.getEventName() {
						t.Error(fmt.Sprintf("Test ID %v:: Expected Send Action Event to be %v, found %v", testId, expAction.Event.getEventName(), actAction.Event.getEventName()))
						errorFound = true
						break
					}

					if detailedEvent {
						errorFound = expectEvents(t, actAction.Event, expAction.Event, testId)
						if errorFound {
							break
						}
					}
				case Alarm:
				case Commit:
					actAction := actual[i].(Commit)
					expAction := expected[i].(Commit)

					if actAction.Err != nil && expAction.Err != nil {
						if actAction.Err.Error() != expAction.Err.Error() {
							t.Error(fmt.Sprintf("Test ID %v:: Expected Commit Action error to be %v, found %v", testId, expAction.Err.Error(), actAction.Err.Error()))
							errorFound = true
							break
						}
					} else {
						if actAction.Err != expAction.Err {
							t.Error(fmt.Sprintf("Test ID %v:: Expected Commit Action error (can be nil) to be %v, found %v", testId, expAction.Err, actAction.Err))
							errorFound = true
							break
						}
					}

					if string(actAction.Data) != string(expAction.Data) {
						t.Error(fmt.Sprintf("Test ID %v:: Expected Commit Action data to be %v, found %v", testId, string(expAction.Data), string(actAction.Data)))
						errorFound = true
						break
					}

					if actAction.Index != expAction.Index {
						t.Error(fmt.Sprintf("Test ID %v:: Expected Commit Action index to be %v, found %v", testId, expAction.Index, actAction.Index))
						errorFound = true
						break
					}
				case LogStore:
					actAction := actual[i].(LogStore)
					expAction := expected[i].(LogStore)

					if actAction.Index != expAction.Index {
						t.Error(fmt.Sprintf("Test ID %v:: Expected LogStore Action index to be %v, found %v", testId, expAction.Index, actAction.Index))
						errorFound = true
						break
					}

					if actAction.Term != expAction.Term {
						t.Error(fmt.Sprintf("Test ID %v:: Expected LogStore Action term to be %v, found %v ", testId, expAction.Term, actAction.Term))
						errorFound = true
						break
					}

					if string(actAction.Data) != string(expAction.Data) {
						t.Error(fmt.Sprintf("Test ID %v:: Expected LogStore Action data to be %v, found %v", testId, string(expAction.Data), string(actAction.Data)))
						errorFound = true
						break
					}

				case StateStore:
					actAction := actual[i].(StateStore)
					expAction := expected[i].(StateStore)

					if actAction.Term != expAction.Term {
						t.Error(fmt.Sprintf("Test ID %v:: Expected StateStore Action term to be %v, found %v", testId, expAction.Term, actAction.Term))
						errorFound = true
						break
					}

					if actAction.VotedFor != expAction.VotedFor {
						t.Error(fmt.Sprintf("Test ID %v:: Expected LogStore Action votedFor to be %v, found %v", testId, expAction.VotedFor, actAction.VotedFor))
						errorFound = true
						break
					}
				}
			}
		}
		if errorFound {
			break
		}
	}
}

func expectState(t *testing.T, a State, b State, testId string) {
	var expected, actual string
	if a == FOLLOWER {
		actual = "FOLLOWER"
	} else if a == LEADER {
		actual = "LEADER"
	} else {
		actual = "CANDIDATE"
	}
	if b == FOLLOWER {
		expected = "FOLLOWER"
	} else if b == LEADER {
		expected = "LEADER"
	} else {
		expected = "CANDIDATE"
	}
	if a != b {
		t.Error(fmt.Sprintf("Test ID %v:: Expected State %v, found %v", testId, expected, actual)) // t.Error is visible when running `go test -verbose`
	}
}

func expect(t *testing.T, a int, b int, testId string) {
	if a != b {
		t.Error(fmt.Sprintf("Test ID %v:: Expected %v, found %v", testId, b, a)) // t.Error is visible when running `go test -verbose`
	}
}


func createIntArray(N int, initialValue int) []int {
	var tmpArray []int
	for i := 0; i < N; i++ {
		tmpArray = append(tmpArray, initialValue)
	}
	//	for i := 0; i< len(actions); i++ {
	//		fmt.Printf("i = %v and event = %v \n", i+1, actions[i].getActionName())
	//	}
	return tmpArray
}
func createUnsignedIntIndexArray(N int, initialValue uint) []uint {
	var tmpArray []uint
	for i := 0; i < N; i++ {
		tmpArray = append(tmpArray, initialValue)
	}
	return tmpArray
}
