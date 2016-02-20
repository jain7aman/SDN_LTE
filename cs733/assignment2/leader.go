package main

import (
	"sort"
)

func (sm *RaftServer) leader() State {
	// declaring the Volatile state on leaders
	//	nextIndex := make([]uint, sm.N) // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	//	matchIndex := make([]int, sm.N) // For each server, index of highest log entry known to be replicated on server (initialized to -1 (since log index starts from 0), increases monotonically)

	prevLogTerm := uint(0)
	if len(sm.Log) > 0 {
		prevLogTerm = sm.Log[len(sm.Log)-1].Term
	}

	sm.MatchIndex[sm.Id] = len(sm.Log) - 1
	sm.VotedFor = -1
	//Setup the hearbeat timer TIMEOUT event
	sm.SendChannel <- Alarm{Time: RandomWaitTime} //sleep for RANDOM time

	// initialize the Volatile state on leaders and send initial empty AppendEntriesRPC (heartbeat) to each server
	for i := uint(0); i < sm.N; i++ {
		//sm.NextIndex[i] = uint(len(sm.Log))
		//sm.MatchIndex[i] = -1
		if i != sm.Id { //send to all except self
			sm.SendChannel <- Send{sm.Id, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommitIndex, LeaderId: sm.Id, PrevLogIndex: int(len(sm.Log)) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
		}
	}

	event := <-sm.ReceiveChannel
	// Loops through the events being received from event channel
	//	for event := range sm.ReceiveChannel {
	switch event.(type) {
	case TimeoutEvent: // heartbeat messages
		for i := uint(0); i < sm.N; i++ {
			if i != sm.Id {
				if int(sm.NextIndex[i]) <= (len(sm.Log) - 1) { //receiver log is not uptodate
					prevLogTerm = uint(0)
					if len(sm.Log) > 0 && sm.NextIndex[i] > 0 {
						prevLogTerm = sm.Log[sm.NextIndex[i]-1].Term
					}
					// send the entries not present in follower's log to respective follower
					newEntries := sm.Log[sm.NextIndex[i]:]
					sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: newEntries, LeaderCommit: sm.CommitIndex, LeaderId: sm.Id, PrevLogIndex: int(sm.NextIndex[i]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
				} else {
					// sends the empty heartbeat
					sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommitIndex, LeaderId: sm.Id, PrevLogIndex: int(sm.NextIndex[i]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
				}
			}
		}
		sm.SendChannel <- NoAction{}

	case VoteReqEvent: // This may be the old leader
		msg := event.(VoteReqEvent)
		if sm.Term >= msg.Term {
			sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}} // Reject the Vote
			sm.SendChannel <- NoAction{}
		} else {
			lastTerm := uint(0)
			lastIndex := -1
			if len(sm.Log) > 0 {
				lastTerm = sm.Log[len(sm.Log)-1].Term
				lastIndex = len(sm.Log) - 1
			}
			sm.Term = msg.Term //updating my term
			sm.VotedFor = -1
			// Voting server V denies vote if its log is “more complete”
			if (lastTerm > msg.LastLogTerm) || (lastTerm == msg.LastLogTerm && lastIndex > msg.LastLogIndex) {
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}} // Reject the Vote
			} else {
				if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) {
					sm.VotedFor = int(msg.CandidateId)
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}} // Give the Vote
				}
				//				else { // This might not happen anytime
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}} // Already voted for other, Reject the Vote
				//				}
			}

			//when leader changes to follower if sends NACKS to client for each uncomitted entry
			if sm.CommitIndex < len(sm.Log)-1 {
				for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
					sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}
				}
			}

			sm.SendChannel <- NoAction{}
			return FOLLOWER
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
		sm.SendChannel <- LogStore{Index: uint(index), Term: sm.Term, Data: msg.Command}
		sm.SendChannel <- NoAction{}

	case AppendEntriesReqEvent: // may be this leader is old leader and new leader has alreday been elected
		msg := event.(AppendEntriesReqEvent)

		if sm.Term < msg.Term { // leader is outdated
			sm.Term = msg.Term
			sm.VotedFor = -1
			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			sm.LeaderId = msg.LeaderId
			// send a fail response to new leader, it will automatically bring this server in sync
			// using AppendEntries RPC's when this server changes to FOLLOWER state
			sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}}

			//when leader changes to follower if sends NACKS to client for each uncomitted entry
			if sm.CommitIndex < len(sm.Log)-1 {
				for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
					sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderId, Prob: "ERR_REDIRECTION"}}
				}
			}

			sm.SendChannel <- NoAction{}
			return FOLLOWER
		} else {
			// This case should never occur => two leaders in the system
			// else send a fail AppendEntries response RPC's to let the other leader know it is there
			sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}}
			sm.SendChannel <- NoAction{}
		}

	case AppendEntriesRespEvent:
		msg := event.(AppendEntriesRespEvent)

		followerId := msg.FollowerId
		if msg.Success == false {

			if sm.Term < msg.Term { // it implies this leader is outdated to should immediately revert to follower
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.LeaderId = followerId

				//when leader changes to follower if sends NACKS to client for each uncomitted entry
				if sm.CommitIndex < len(sm.Log)-1 {
					for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
						sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderId, Prob: "ERR_REDIRECTION"}}
					}
				}
				sm.SendChannel <- NoAction{}
				return FOLLOWER
			}

			if sm.NextIndex[followerId] > 0 {
				sm.NextIndex[followerId]--
			} else { // this should not happen but for safety
				sm.NextIndex[followerId] = 0
			}

			sm.SendChannel <- NoAction{}
			//				continue // continue the event loop
		} else {

			//update the match index for this follower
			if msg.FollowerIndex >= 0 {
				sm.MatchIndex[followerId] = int(msg.FollowerIndex)
			}

			// now check to see if an index entry has been replicated on majority of servers
			// if yes then commit the entry and send the response back to client
			tempMatchIndex := make([]int, len(sm.MatchIndex))
			copy(tempMatchIndex, sm.MatchIndex)
			sort.Ints(tempMatchIndex)

			newCommitIndex := sm.CommitIndex
			for i := int(sm.N) - 1; i >= 0; i-- {
				count := 0
				for j := uint(0); j < sm.N; j++ {

					if sm.MatchIndex[j] >= tempMatchIndex[i] && tempMatchIndex[i] <= len(sm.Log)-1 {
						count++
					}
				}
				// Mark log entries committed if stored on a majority of
				// servers and at least one entry from current term is stored on
				// a majority of servers
				if count > int(sm.N/2) && sm.Log[tempMatchIndex[i]].Term == sm.Term {
					newCommitIndex = tempMatchIndex[i]
					break
				}
				//				else if  count == sm.N { // command even if previous leader is replicate on all servers
				//					sm.CommitIndex = sm.MatchIndexCopy[i]
				//					break
				//				}
			}

			if newCommitIndex > sm.CommitIndex {
				//send the commit success response to client for each new committed entry
				for i := sm.CommitIndex + 1; i <= newCommitIndex; i++ {
					sm.SendChannel <- Commit{Index: uint(i), Data: sm.Log[i].Command, Err: nil}
				}
				sm.CommitIndex = newCommitIndex
			}
			sm.SendChannel <- NoAction{}
		}
	case VoteRespEvent:
		msg := event.(VoteRespEvent)
		if msg.Term > sm.Term { //Followers term is greater than my term
			sm.Term = msg.Term
			sm.VotedFor = -1
			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

			//when leader changes to follower if sends NACKS to client for each uncomitted entry
			if sm.CommitIndex < len(sm.Log)-1 {
				for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
					sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}
				}
			}
			sm.SendChannel <- NoAction{}
			return FOLLOWER
		}
		sm.SendChannel <- NoAction{}
	}
	//	}
	return LEADER
}
