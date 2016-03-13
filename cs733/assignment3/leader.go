package main

import (
	"log"
	"sort"
)

func (sm *RaftServer) leader() State {
	// declaring the Volatile state on leaders
	nextIndex := make([]uint, sm.N) // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex := make([]int, sm.N) // For each server, index of highest log entry known to be replicated on server (initialized to -1 (since log index starts from 0), increases monotonically)

	prevLogTerm := uint(0)
	if len(sm.Log) > 0 {
		prevLogTerm = sm.Log[len(sm.Log)-1].Term
	}

	matchIndex[sm.ID-1] = len(sm.Log) - 1
	sm.VotedFor = -1
	//Setup the hearbeat timer TIMEOUT event
	sm.SendChannel <- Alarm{Time: uint(sm.HeartbeatTimeout)} //sleep for RANDOM time

	// initialize the Volatile state on leaders and send initial empty AppendEntriesRPC (heartbeat) to each server
	for i := 1; i <= int(sm.N); i++ {
		//nextIndex[i] = uint(len(sm.Log))
		//matchIndex[i] = -1
		if i != sm.ID { //send to all except self
			sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommitIndex, LeaderId: sm.ID, PrevLogIndex: int(len(sm.Log)) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
		}
	}

	//	event := <-sm.ReceiveChannel
	// Loops through the events being received from event channel
	for event := range sm.ReceiveChannel {
		switch event.(type) {
		case TimeoutEvent: // heartbeat messages
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(TimeoutEvent).getEventName())
			for i := 1; i <= int(sm.N); i++ {
				if i != sm.ID {
					if int(nextIndex[i-1]) <= (len(sm.Log) - 1) { //receiver log is not uptodate
						prevLogTerm = uint(0)
						if len(sm.Log) > 0 && nextIndex[i-1] > 0 {
							prevLogTerm = sm.Log[nextIndex[i-1]-1].Term
						}
						// send the entries not present in follower's log to respective follower
						newEntries := sm.Log[nextIndex[i-1]:]
						sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: newEntries, LeaderCommit: sm.CommitIndex, LeaderId: sm.ID, PrevLogIndex: int(nextIndex[i-1]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
					} else {
						// sends the empty heartbeat
						sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommitIndex, LeaderId: sm.ID, PrevLogIndex: int(nextIndex[i-1]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
					}
				}
			}
			sm.SendChannel <- NoAction{}

		case VoteReqEvent: // This may be the old leader
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(VoteReqEvent).getEventName())
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
					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Reject the Vote
				} else {
					if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) {
						sm.VotedFor = int(msg.CandidateId)
						sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
						sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: true}} // Give the Vote
					}
					//				else { // This might not happen anytime
					//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					//					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Already voted for other, Reject the Vote
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
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEvent).getEventName())
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
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEntriesReqEvent).getEventName())
			msg := event.(AppendEntriesReqEvent)

			if sm.Term < msg.Term { // leader is outdated
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.LeaderID = msg.LeaderId
				// send a fail response to new leader, it will automatically bring this server in sync
				// using AppendEntries RPC's when this server changes to FOLLOWER state
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}

				//when leader changes to follower if sends NACKS to client for each uncomitted entry
				if sm.CommitIndex < len(sm.Log)-1 {
					for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
						sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderID, Prob: "ERR_REDIRECTION"}}
					}
				}

				sm.SendChannel <- NoAction{}
				return FOLLOWER
			} else {
				// This case should never occur => two leaders in the system
				// else send a fail AppendEntries response RPC's to let the other leader know it is there
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}
				sm.SendChannel <- NoAction{}
			}

		case AppendEntriesRespEvent:
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEntriesRespEvent).getEventName())
			msg := event.(AppendEntriesRespEvent)

			followerId := msg.FollowerId
			if msg.Success == false {

				if sm.Term < msg.Term { // it implies this leader is outdated to should immediately revert to follower
					sm.Term = msg.Term
					sm.VotedFor = -1
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					sm.LeaderID = followerId

					//when leader changes to follower if sends NACKS to client for each uncomitted entry
					if sm.CommitIndex < len(sm.Log)-1 {
						for i := sm.CommitIndex + 1; i < len(sm.Log); i++ {
							sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderID, Prob: "ERR_REDIRECTION"}}
						}
					}
					sm.SendChannel <- NoAction{}
					return FOLLOWER
				}

				if nextIndex[followerId-1] > 0 {
					nextIndex[followerId-1]--
				} else { // this should not happen but for safety
					nextIndex[followerId-1] = 0
				}

				sm.SendChannel <- NoAction{}
				//				continue // continue the event loop
			} else {

				//update the match index for this follower
				if msg.FollowerIndex >= 0 {
					matchIndex[followerId-1] = int(msg.FollowerIndex)

					// now check to see if an index entry has been replicated on majority of servers
					// if yes then commit the entry and send the response back to client
					tempMatchIndex := make([]int, sm.N)
					copy(tempMatchIndex, matchIndex)
					sort.Ints(tempMatchIndex)

					newCommitIndex := sm.CommitIndex
					for i := int(sm.N) - 1; i >= 0; i-- {
						count := 0
						for j := uint(0); j < sm.N; j++ {

							if matchIndex[j] >= tempMatchIndex[i] && tempMatchIndex[i] <= len(sm.Log)-1 {
								count++
							}
						}
						// Mark log entries committed if stored on a majority of
						// servers and at least one entry from current term is stored on
						// a majority of servers
						if len(sm.Log) > 0 {
							log.Printf("***** LEADER ***** server id = %v length of log = %v i = %v and tempMatchIndex[i] = %v \n", sm.ID, len(sm.Log), i, tempMatchIndex[i])
							if count > int(sm.N/2) && sm.Log[tempMatchIndex[i]].Term == sm.Term {
								newCommitIndex = tempMatchIndex[i]
								break
							}
						}
						//				else if  count == sm.N { // command even if previous leader is replicate on all servers
						//					sm.CommitIndex = matchIndexCopy[i]
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
				}
				sm.SendChannel <- NoAction{}
			}
		case VoteRespEvent:
			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(VoteRespEvent).getEventName())
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
	}
	return LEADER
}
