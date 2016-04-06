package main

import (
	"log"
	"time"
)

//			log.Printf(.*\(\).*)
// log.Printf$1}
var DebugFollower bool = false

func (sm *RaftServer) follower() State {
	//Step 1: sets a random timer for the timeout event of follower
	sm.SendChannel <- Alarm{Time: uint(sm.ElectionTimeout)} //sleep for RANDOM time

	//Step 2: loops through the events being received from event channel
	//	event := <-sm.ReceiveChannel
	for event := range sm.ReceiveChannel {
		switch event.(type) {
		case TimeoutEvent:
			if DebugFollower {
				log.Printf("%v:: ** FOLLOWER ** Server ID = %v  Term = %v and event = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(TimeoutEvent).getEventName())
			}
			sm.Term++ //increments the term number
			sm.VotedFor = -1
			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

			return CANDIDATE //returns to CANDIDATE state

		case VoteReqEvent:

			msg := event.(VoteReqEvent)
			if sm.Term > msg.Term {
				if DebugFollower {
					log.Printf("%v:: ** FOLLOWER ** VOTE REJECTED by Server ID = %v (Term = %v)  to server %v (Term = %v) \n", time.Now().Nanosecond(), sm.Id(), sm.Term, msg.CandidateId, msg.Term)
				}
				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}}

				//continue
			} else {
				termChanged := false
				if sm.Term < msg.Term {
					sm.Term = msg.Term
					termChanged = true
				}

				// reset the timer for Timeout event, as the network is alive
				sm.SendChannel <- Alarm{Time: uint(sm.ElectionTimeout)} //sleep for RANDOM time

				lastTerm := uint(0)
				lastIndex := -1
				if len(sm.Log) > 0 {
					lastTerm = sm.Log[len(sm.Log)-1].Term
					lastIndex = len(sm.Log) - 1
				}

				// Voting server V denies vote if its log is “more complete”
				if (lastTerm > msg.LastLogTerm) || (lastTerm == msg.LastLogTerm && lastIndex > msg.LastLogIndex) {
					if DebugFollower {
						log.Printf("%v:: ** FOLLOWER ** VOTE REJECTED by Server ID = %v (Term = %v)  to server %v (Term = %v) my log more complete TIMER UPDATED \n", time.Now().Nanosecond(), sm.Id(), sm.Term, msg.CandidateId, msg.Term)
					}
					if termChanged == true {
						sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					}
					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Reject the Vote
				} else {
					if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) {
						if DebugFollower {
							log.Printf("%v:: ** FOLLOWER ** VOTE GIVEN by Server ID = %v (Term = %v)  to server %v (Term = %v) TIMER UPDATED \n", time.Now().Nanosecond(), sm.Id(), sm.Term, msg.CandidateId, msg.Term)
						}
						sm.VotedFor = int(msg.CandidateId)
						sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
						sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: true}} // Give the Vote
					} else {
						if DebugFollower {
							log.Printf("%v:: ** FOLLOWER ** VOTE REJECTED by Server ID = %v (Term = %v)  to server %v (Term = %v) already voted for different candidate TIMER UPDATED \n", time.Now().Nanosecond(), sm.Id(), sm.Term, msg.CandidateId, msg.Term)
						}
						if termChanged == true {
							sm.VotedFor = -1
							sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
						}
						sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Already voted for other, Reject the Vote
					}
				}

				//check if candidates log is as upto to date as follower's term
				//			if sm.Term == msg.Term && (sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId)) && len(sm.Log)-1 <= msg.LastLogIndex {
				//				sm.VotedFor = int(msg.CandidateId)
				//				if termChanged == true {
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//				}
				//				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: true}} // Give the Vote
				//			} else {
				//				if termChanged == true {
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//				}
				//				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Reject the Vote
				//			}

			}

		case AppendEntriesReqEvent:

			msg := event.(AppendEntriesReqEvent)

			if msg.Term < sm.Term { //Followers term is greater than leader's term
				if DebugFollower {
					log.Printf("%v:: ** FOLLOWER ** AppendEntriesReqEvent Rejected %v (Term = %v) - > Server ID = %v (Term = %v) TIMER NOT UPDATED \n", time.Now().Nanosecond(), msg.LeaderId, msg.Term, sm.Id(), sm.Term)
				}
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}} //Leader is old and not uptodate

				// continue //continue the events loop
			} else {
				sm.LeaderID = msg.LeaderId // update the leader ID
				if DebugFollower {
					log.Printf(" ########################## UPdating the leaders id to %v in follower = %v \n", sm.LeaderId(), sm.Id())
				}
				if msg.Term > sm.Term {
					sm.Term = msg.Term
					sm.VotedFor = -1
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				}

				if DebugFollower {
					log.Printf("%v:: %%%%%%%%%%%%%%** FOLLOWER ** AppendEntriesReqEvent log size = %v  msg.PrevLogIndex = %v ****************** %v (Term = %v) - > Server ID = %v (Term = %v) TIMER UPDATED \n", time.Now().Nanosecond(), len(sm.Log), msg.PrevLogIndex, msg.LeaderId, msg.Term, sm.Id(), sm.Term)
				}
				// Reset the timer for Timeout event, as the network is alive
				sm.SendChannel <- Alarm{Time: uint(sm.ElectionTimeout)} //sleep for RANDOM time

				if msg.LeaderCommit > sm.CommittedIndex() {
					//set commit index = min(leaderCommit, index of last new entry)
					newCommitIndex := minimum(msg.LeaderCommit, len(sm.Log)-1)

					if newCommitIndex > sm.CommittedIndex() {
						//send the commit success response to client for each new committed entry
						for i := sm.CommittedIndex() + 1; i <= newCommitIndex; i++ {
							sm.SendChannel <- Commit{Index: uint(i), Data: sm.Log[i].Command, Err: nil}
						}
						//						sm.CommitIndex = newCommitIndex
						sm.setCommittedIndex(newCommitIndex)
					}
				}

				if len(msg.Entries) == 0 { // heart beat message
					if DebugFollower {
						log.Printf("%v:: ** FOLLOWER ** AppendEntriesReqEvent HEARTBEAT %v (Term = %v) - > Server ID = %v (Term = %v) TIMER UPDATED \n", time.Now().Nanosecond(), msg.LeaderId, msg.Term, sm.Id(), sm.Term)
					}
					sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: true, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}} //Leader is old and not uptodate
					continue
				}

				// If length of log is smaller at follower then its previous log index wont match with the prevLogIndex of Leader
				if len(sm.Log)-1 < msg.PrevLogIndex {
					if DebugFollower {
						log.Printf(" &*& FOLLOWER &*& FOllower id = %v (term =%v) leader id = %v (term =%v) len(sm.Log)-1 = %v msg.PrevLogIndex = %v \n", sm.Id(), sm.Term, msg.LeaderId, msg.Term, len(sm.Log)-1, msg.PrevLogIndex)
					}
					sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}

					//			continue //continue the events loop
				} else {

					termMatch := true
					if DebugFollower {
						log.Printf("%v:: (((((((((((((((((((((((( server id = %v len(sm.Log) = %v msg.PrevLogIndex %v \n", time.Now().Nanosecond(), sm.ID, len(sm.Log), msg.PrevLogIndex)
					}
					if len(sm.Log) > 0 {
						if DebugFollower {
							log.Printf("%v:: kkkkkkkkkkkkkkkkkkkkk server id = %v len(sm.Log) = %v msg.PrevLogIndex %v  msg.PrevLogTerm = %v \n", time.Now().Nanosecond(), sm.ID, len(sm.Log), msg.PrevLogIndex, msg.PrevLogTerm)
						}
						//Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogIndex
						if msg.PrevLogIndex > -1 && sm.Log[msg.PrevLogIndex].Term != msg.PrevLogTerm {
							sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}
							termMatch = false

							continue //continue the events loop
						}
					}
					if termMatch == true {
						if DebugFollower {
							log.Printf("%v:: YOOOOOOOOOOOOOOOOOOOOOOO server id = %v len(msg.Entries) = %v msg.PrevLogIndex %v \n", time.Now().Nanosecond(), sm.ID, len(msg.Entries), msg.PrevLogIndex)
						}
						for i := 0; i < len(msg.Entries); i++ {
							// check if the entries to be appended is already present in the log at PrevLogIndex+1+i
							// if present then ignore the entry else delete that entry and all that follows it
							// and starts appending the new entries send by leader
							if len(sm.Log)-1 >= (msg.PrevLogIndex + 1 + i) {
								if sm.Log[msg.PrevLogIndex+1+i].Term == msg.Entries[i].Term {
									continue //Entry already exists.. do nothing
								} else {
									sm.Log = sm.Log[0 : msg.PrevLogIndex+i+1] //delete the entry and all the entries after it
								}
							}
							sm.Log = append(sm.Log, msg.Entries[i])
							sm.SendChannel <- LogStore{Index: uint(len(sm.Log) - 1), Term: msg.Entries[i].Term, Data: msg.Entries[i].Command}
						}

						if msg.LeaderCommit > sm.CommittedIndex() {
							//set commit index = min(leaderCommit, index of last new entry)
							newCommitIndex := minimum(msg.LeaderCommit, len(sm.Log)-1)

							if newCommitIndex > sm.CommittedIndex() {
								//send the commit success response to client for each new committed entry
								for i := sm.CommittedIndex() + 1; i <= newCommitIndex; i++ {
									sm.SendChannel <- Commit{Index: uint(i), Data: sm.Log[i].Command, Err: nil}
								}
								//								sm.CommitIndex = newCommitIndex
								sm.setCommittedIndex(newCommitIndex)
							}
						}
						sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: true, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}} //send a success response back to leader

					}
				}
			}

		case AppendEvent:
			if DebugFollower {
				log.Printf("*$$$* FOLLOWER *$$$* Server ID = %v  sm.LeaderID = %v Term = %v and event = %v \n", sm.Id(), sm.LeaderID, sm.Term, event.(AppendEvent).getEventName())
			}
			//if a client contacts a follower, the follower redirects it to the leader
			msg := event.(AppendEvent)
			//Send a response back to client saying that I am not the Leader and giving client the leader ID to contact to
			sm.SendChannel <- Commit{Data: msg.Command, Err: &AppendError{LeaderId: sm.LeaderID, Prob: "ERR_REDIRECTION"}} //send a success response back to leader

		case AppendEntriesRespEvent:
			if DebugFollower {
				log.Printf("** FOLLOWER ** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(AppendEntriesRespEvent).getEventName())
			}
			msg := event.(AppendEntriesRespEvent)
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			}

		case VoteRespEvent:
			if DebugFollower {
				log.Printf("** FOLLOWER ** Server ID = %v  Term = %v and event = %v \n", sm.Id(), sm.Term, event.(VoteRespEvent).getEventName())
			}
			msg := event.(VoteRespEvent)
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			}

		case ShutDownEvent:
			msg := event.(ShutDownEvent)
			sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
			sm.QuitChannel <- true
			if DebugFollower {
				log.Printf("%v:: Server %v Shutting down...\n", time.Now().Nanosecond(), msg.Id)
			}
			return POWEROFF
		}
	}
	return FOLLOWER
}
