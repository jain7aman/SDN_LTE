package main

import (
	"log"
	"sort"
	"time"
)

var DebugLeader bool = false

func (sm *RaftServer) leader() State {
	// declaring the Volatile state on leaders
	nextIndex := make([]int, sm.N)  // For each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
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
		nextIndex[i-1] = int(len(sm.Log))
		matchIndex[i-1] = -1
		if i != sm.ID { //send to all except self
			sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommittedIndex(), LeaderId: sm.ID, PrevLogIndex: int(len(sm.Log)) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
		}
	}

	//	event := <-sm.ReceiveChannel
	// Loops through the events being received from event channel
	for event := range sm.ReceiveChannel {
		matchIndex[sm.ID-1] = len(sm.Log) - 1 // updating my match index
		switch event.(type) {
		case TimeoutEvent: // heartbeat messages
			//			log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(TimeoutEvent).getEventName())
			for i := 1; i <= int(sm.N); i++ {
				if i != sm.ID {
					//					if DebugLeader {
					//						log.Printf(" $$$$$$$$$$$$$ LEADER $$$$$$$$$$$ leader id = %v server id = %v  nextindex = %v and leader log length = %v\n", sm.Id(), i, int(nextIndex[i-1]), len(sm.Log))
					//					}
					if len(sm.Log) > 0 && int(nextIndex[i-1]) < len(sm.Log) { //receiver log is not uptodate
						prevLogTerm = sm.Term
						if len(sm.Log) > 0 && nextIndex[i-1] > 0 {
							prevLogTerm = sm.Log[nextIndex[i-1]-1].Term
						}
						// send the entries not present in follower's log to respective follower
						newEntries := sm.Log[nextIndex[i-1]:]

						if DebugLeader && i == 1 {
							log.Printf("%v:: #############################################Data = %v Size of new entries = %v to server = %v by server = %v nextIndex[i-1] =%v \n", time.Now().Nanosecond(), string(newEntries[0].Command), len(newEntries), i, sm.ID, nextIndex[i-1])
						}
						sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: newEntries, LeaderCommit: sm.CommittedIndex(), LeaderId: sm.ID, PrevLogIndex: int(nextIndex[i-1]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
						nextIndex[i-1] = int(len(sm.Log))

					} else {
						if DebugLeader && i == 1 {
							log.Printf("%v::  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^HEART BEAT to server = %v by server = %v \n", time.Now().Nanosecond(), i, sm.ID)
						}
						// sends the empty heartbeat
						sm.SendChannel <- Send{i, AppendEntriesReqEvent{Entries: make([]LogEntry, 0), LeaderCommit: sm.CommittedIndex(), LeaderId: sm.ID, PrevLogIndex: int(nextIndex[i-1]) - 1, PrevLogTerm: prevLogTerm, Term: sm.Term}}
					}
				}
			}
			sm.SendChannel <- Alarm{Time: uint(sm.HeartbeatTimeout)} //sleep for RANDOM time

		case VoteReqEvent: // This may be the old leader
			if DebugLeader {
				log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(VoteReqEvent).getEventName())
			}
			msg := event.(VoteReqEvent)
			if sm.Term >= msg.Term {
				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}} // Reject the Vote

			} else {

				lastTerm := uint(0)
				lastIndex := -1
				if len(sm.Log) > 0 {
					lastTerm = sm.Log[len(sm.Log)-1].Term
					lastIndex = len(sm.Log) - 1
				}
				if DebugLeader {
					log.Printf("%v:: ***** OUTDATED LEADER *****VoteReqEvent   %v (Term = %v) -> %v (Term = %v) \n", time.Now().Nanosecond(), msg.CandidateId, msg.Term, sm.Id(), sm.Term)
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
				if sm.CommittedIndex() < len(sm.Log)-1 {
					for i := sm.CommittedIndex() + 1; i < len(sm.Log); i++ {
						sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}
					}
				}
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer

				return FOLLOWER
			}

		case AppendEvent:
			if DebugLeader {
				log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEvent).getEventName())
			}
			msg := event.(AppendEvent)

			index := 0
			if len(sm.Log) > 0 {
				index = len(sm.Log)
			}
			// if received a new command from client, append it to log, it will sent to followers in heartbeat
			// messages or in AppendEntriesRespEvent
			sm.Log = append(sm.Log, LogEntry{Command: msg.Command, Index: uint(index), Term: sm.Term})
			sm.SendChannel <- LogStore{Index: uint(index), Term: sm.Term, Data: msg.Command}

		case AppendEntriesReqEvent: // may be this leader is old leader and new leader has alreday been elected
			if DebugLeader {
				log.Printf("***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEntriesReqEvent).getEventName())
			}
			msg := event.(AppendEntriesReqEvent)

			if sm.Term < msg.Term { // leader is outdated
				if DebugLeader {
					log.Printf("%v:: ***** OUTDATED LEADER *****  AppendEntriesReqEvent %v (Term = %v) -> %v (Term = %v )  \n", time.Now().Nanosecond(), msg.LeaderId, msg.Term, sm.Id(), sm.Term)
				}
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.LeaderID = msg.LeaderId
				// send a fail response to new leader, it will automatically bring this server in sync
				// using AppendEntries RPC's when this server changes to FOLLOWER state
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}

				//when leader changes to follower if sends NACKS to client for each uncomitted entry
				if sm.CommittedIndex() < len(sm.Log)-1 {
					for i := sm.CommittedIndex() + 1; i < len(sm.Log); i++ {
						sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderID, Prob: "ERR_REDIRECTION"}}
					}
				}
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer

				return FOLLOWER
			} else {
				// This case should never occur => two leaders in the system
				// else send a fail AppendEntries response RPC's to let the other leader know it is there
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}

			}

		case AppendEntriesRespEvent:

			msg := event.(AppendEntriesRespEvent)
			followerId := msg.FollowerId
			if msg.Success == false {
				if DebugLeader {
					log.Printf("%v:: ***** LEADER ***** AppendEntriesRespEvent REJECTED %v (Term = %v) -> %v (Term = %v) \n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term)
				}
				if sm.Term < msg.Term { // it implies this leader is outdated to should immediately revert to follower
					if DebugLeader {
						log.Printf("%v:: ***** OUTDATED LEADER *****  AppendEntriesRespEvent %v (Term = %v) -> %v (Term = %v )  \n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term)
					}
					sm.Term = msg.Term
					sm.VotedFor = -1
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					sm.LeaderID = followerId
					if DebugLeader {
						log.Printf("%v:: ***** LEADER ***** AppendEntriesRespEvent REJECTED %v (Term = %v) -> %v (Term = %v)  LEADER -> FOLLOWER \n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term)
					}
					//when leader changes to follower if sends NACKS to client for each uncomitted entry
					if sm.CommittedIndex() < len(sm.Log)-1 {
						for i := sm.CommittedIndex() + 1; i < len(sm.Log); i++ {
							sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{LeaderId: sm.LeaderID, Prob: "ERR_REDIRECTION"}}
						}
					}
					sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer

					return FOLLOWER
				}

				if nextIndex[followerId-1] > 0 {

					nextIndex[followerId-1]--
					if DebugLeader || Debug2 {
						log.Printf(" Decreasing nextIndex for follower = %v by leader = %v now next index = %v \n", followerId, sm.Id(), nextIndex[followerId-1])
					}
				} else { // this should not happen but for safety
					nextIndex[followerId-1] = 0
				}

				//				continue // continue the event loop
			} else {
				if DebugLeader {
					log.Printf("%v:: ***** LEADER ***** AppendEntriesRespEvent ACCEPTED %v (Term = %v) -> %v (Term = %v) ______________ folllower id = %v  follower index = %v next index = %v test =%v\n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term, msg.FollowerId, msg.FollowerIndex, nextIndex[followerId-1], (msg.FollowerIndex+1 < nextIndex[followerId-1] && nextIndex[followerId-1] > 0))
				}
				//update the nextIndex of follower if required
				if msg.FollowerIndex+1 < nextIndex[followerId-1] && nextIndex[followerId-1] > 0 {
					//					if DebugLeader || Debug2 {
					//						log.Printf(" *^^^^^** LEADER **** Leader id = %v  folllower id = %v  follower index = %v next index = %v \n", sm.Id(), msg.FollowerId, msg.FollowerIndex, nextIndex[followerId-1])
					//					}
					nextIndex[followerId-1] = msg.FollowerIndex + 1
					//					if DebugLeader || Debug2 {
					//						log.Printf(" NEW******* LEADER **** Leader id = %v  folllower id = %v  follower index = %v next index = %v \n", sm.Id(), msg.FollowerId, msg.FollowerIndex, nextIndex[followerId-1])
					//					}
				}
				//update the match index for this follower
				if msg.FollowerIndex >= 0 {
					matchIndex[followerId-1] = int(msg.FollowerIndex)

					// now check to see if an index entry has been replicated on majority of servers
					// if yes then commit the entry and send the response back to client
					tempMatchIndex := make([]int, sm.N)
					copy(tempMatchIndex, matchIndex)
					sort.Ints(tempMatchIndex)

					newCommitIndex := sm.CommittedIndex()
					for i := int(sm.N) - 1; i >= 0; i-- {
						count := 0
						for j := uint(0); j < sm.N; j++ {
							//							if DebugLeader {
							//								log.Printf(" &&&&&&&&&& LEADER &&&&&&&&&& LEADER ID = %v  server_id = %v   j = %v matchIndex[j] = %v tempMatchIndex[i] = %v len(sm.Log) = %v test = %v", sm.ID,j+1, j,matchIndex[j], tempMatchIndex[j],len(sm.Log),(matchIndex[j] >= tempMatchIndex[i] && tempMatchIndex[i] <= len(sm.Log)-1))
							//							}
							if matchIndex[j] >= tempMatchIndex[i] && tempMatchIndex[i] <= len(sm.Log)-1 {
								count++
							}
						}
						// Mark log entries committed if stored on a majority of
						// servers and at least one entry from current term is stored on
						// a majority of servers
						if len(sm.Log) > 0 {
							//							if DebugLeader {
							//								log.Printf("***** LEADER ***** server id = %v length of log = %v i = %v and tempMatchIndex[i] = %v count = %v required = %v \n", sm.ID, len(sm.Log), i, tempMatchIndex[i], count, int(sm.N/2))
							//								if tempMatchIndex[i] > -1 {
							//									log.Printf(" #### LEADER #### sm.Log[tempMatchIndex[i]].Term = %v sm.term = %v test = %v \n", sm.Log[tempMatchIndex[i]].Term, sm.Term, (tempMatchIndex[i] > -1 && count > int(sm.N/2) && sm.Log[tempMatchIndex[i]].Term == sm.Term))
							//								}
							//							}
							if tempMatchIndex[i] > -1 && count > int(sm.N/2) && sm.Log[tempMatchIndex[i]].Term == sm.Term {
								newCommitIndex = tempMatchIndex[i]
								//								if DebugLeader {
								//									log.Printf("newCommitIndex %v  sm.CommittedIndex() = %v test = %v\n", newCommitIndex, sm.CommittedIndex(), (newCommitIndex > sm.CommittedIndex()))
								//								}
								break
							}
						}
						//				else if  count == sm.N { // command even if previous leader is replicate on all servers
						//					sm.CommitIndex = matchIndexCopy[i]
						//					break
						//				}
					}

					if newCommitIndex > sm.CommittedIndex() {
						//send the commit success response to client for each new committed entry
						for i := sm.CommittedIndex() + 1; i <= newCommitIndex; i++ {
							sm.SendChannel <- Commit{Index: uint(i), Data: sm.Log[i].Command, Err: nil}
						}
						//						sm.CommitIndex = newCommitIndex
						sm.setCommittedIndex(newCommitIndex)
					}
				}

			}
		case VoteRespEvent:
			if DebugLeader {
				log.Printf("%v:: ***** LEADER ***** Server ID = %v  Term = %v and event = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName())
			}
			msg := event.(VoteRespEvent)
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

				//when leader changes to follower if sends NACKS to client for each uncomitted entry
				if sm.CommittedIndex() < len(sm.Log)-1 {
					for i := sm.CommittedIndex() + 1; i < len(sm.Log); i++ {
						sm.SendChannel <- Commit{Data: sm.Log[i].Command, Err: &AppendError{Prob: "ERR_REDIRECTION"}}
					}
				}
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer

				return FOLLOWER
			}

		case ShutDownEvent:
			msg := event.(ShutDownEvent)
			sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
			sm.QuitChannel <- true
			if DebugLeader {
				log.Printf("%v:: Server %v Shutting down...\n", time.Now().Nanosecond(), msg.Id)
			}
			return POWEROFF
		}
	}
	return LEADER
}
