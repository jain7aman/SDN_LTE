package main

import (
	"log"
	"time"
)

var DebugCandidate bool = false

func (sm *RaftServer) candidate() State {

	sm.VotedFor = int(sm.ID) //votes for self
	//	votesArray := make([]int, sm.N)
	// IN votesArray -1 => NOT voted, 0 => Voted NO and 1=> Voted Yes
	//	for i := uint(0); i < sm.N; i++ {
	//		if i == sm.ID {
	//			sm.VotedFor = int(sm.ID)
	//			votesArray[i] = 1 // voted for self
	//			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
	//		} else {
	//			votesArray[i] = -1 //NO received any vote
	//		}
	//	}
	sm.VotedFor = int(sm.ID)
	sm.VotesArray[sm.ID-1] = 1 // voted for self
	sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

	//	votesReceived := 1
	//	votesNotReceived := 0
	lastLogIndex := -1
	lastLogTerm := uint(0)
	logLength := len(sm.Log)

	if logLength > 0 {
		lastLogTerm = sm.Log[logLength-1].Term
		lastLogIndex = logLength - 1
	}

	// Sets a random ELECTION timer for the timeout event of candidate
	sm.SendChannel <- Alarm{Time: uint(sm.ElectionTimeout)} //sleep for RANDOM time

	// Iissues vote request RPCs in parallel to other servers in the cluster
	for i := 1; i <= int(sm.N); i++ {
		if i != sm.ID {
			sm.SendChannel <- Send{i, VoteReqEvent{CandidateId: sm.ID, LastLogIndex: lastLogIndex, LastLogTerm: lastLogTerm, Term: sm.Term}}
		}
	}

	//	event := <-sm.ReceiveChannel
	//	 Loops through the events being received from event channel
	for event := range sm.ReceiveChannel {
		switch event.(type) {
		case TimeoutEvent:
			if DebugCandidate {
				log.Printf("** CANDIDATE ** Server ID = %v  Term = %v  and event = %v \n", sm.Id(), sm.Term, event.(TimeoutEvent).getEventName())
			}
			sm.Term++ //increments the term number
			sm.VotedFor = -1
			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

			return CANDIDATE //returns to CANDIDATE state

		case VoteRespEvent:
			if DebugCandidate {
				log.Printf("%v:: ** CANDIDATE ** Server ID = %v  Term = %v  and event = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName())
			}
			msg := event.(VoteRespEvent)
			votesReceived := 0
			votesNotReceived := 0
			if msg.VoteGranted == true {

				if sm.VotesArray[msg.Id-1] != 1 { // If already received THIS vote ? If no, then update the acceptance count
					sm.VotesArray[msg.Id-1] = 1
					//					votesReceived++
				}

				for i := uint(0); i < sm.N; i++ {
					if sm.VotesArray[i] == 1 {
						votesReceived++
					} else if sm.VotesArray[i] == 0 {
						votesNotReceived++
					}
				}
				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** Server ID = %v  Term = %v  and event = %v RECEIVED vote from ID =  %v votesReceived = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName(), msg.Id, votesReceived)
				}
			} else {
				//vote rejected because my term is older => should change back to follower
				if sm.Term < msg.Term {
					if DebugCandidate {
						log.Printf("%v:: ** CANDIDATE ** Server ID = %v  Term = %v  and event = %v VOTE REJECTED due to higher term = %v of server = %v return to FOLLOWER \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName(), msg.Term, msg.Id)
					}
					sm.Term = msg.Term
					sm.VotedFor = -1
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

					return FOLLOWER
				}

				if sm.VotesArray[msg.Id-1] != 0 { // If already rejected by THIS vote ? If no, then update the reject count
					sm.VotesArray[msg.Id-1] = 0
					//votesNotReceived++
				}
				for i := uint(0); i < sm.N; i++ {
					if sm.VotesArray[i] == 0 {
						votesNotReceived++
					} else if sm.VotesArray[i] == 1 {
						votesReceived++
					}
				}
				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** Server ID = %v  Term = %v  and event = %v VOTE REJECTED by %v votesNotReceived = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName(), msg.Id, votesNotReceived)
				}
			}
			if DebugCandidate {
				log.Printf("%v:: ** CANDIDATE ** Server ID = %v  Term = %v  and event = %v votesReceived = %v  and votesNotReceived = %v \n", time.Now().Nanosecond(), sm.Id(), sm.Term, event.(VoteRespEvent).getEventName(), votesReceived, votesNotReceived)
			}

			if votesReceived > int(sm.N/2) { //received the majority
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
				sm.LeaderID = sm.ID
				return LEADER //becomes the leader
			}

			if votesNotReceived > int(sm.N/2) { // Did not received the majority
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
				return FOLLOWER                            // Rejected by majority of servers
			}

		case AppendEvent:
			if DebugCandidate {
				log.Printf("** CANDIDATE ** Server ID = %v  Term = %v  and event = %v \n", sm.Id(), sm.Term, event.(AppendEvent).getEventName())
			}
			msg := event.(AppendEvent)
			// can't do anything at this moment, as the State Machine does not know who the current leader is.
			// Send a response back to client saying that I am not the Leader
			sm.SendChannel <- Commit{Data: msg.Command, Err: &AppendError{Prob: "ERR_NO_LEADER_YET"}} //send a success response back to leader

		case AppendEntriesReqEvent:

			msg := event.(AppendEntriesReqEvent)
			// If the leader's term is at least as large as the candidates's current term, then
			// the candidate recognizes the leader as legitimate and returns to FOLLOWER state
			if msg.Term >= sm.Term {
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.LeaderID = msg.LeaderId

				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** AppendEntriesReqEvent ACCEPTED %v (Term = %v) -> %v (Term = %v) CANDIDATE -> FOLLOWER \n", time.Now().Nanosecond(), msg.LeaderId, msg.Term, sm.Id(), sm.Term)
				}
				if len(msg.Entries) == 0 { // heart beat message
					sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: true, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}} //Leader is old and not uptodate
				}

				// BUT WHAT ABOUT THE ENTRIES THAT LEADER SENT, DO WE HAVE TO UPDATE THE ENTRIES HERE ITSELF
				// OR LET THE LEADER SEND THE RPC AGAIN TO THIS SERVER (when it has become follower)
				// Choosing the second option thats is leader will again send AppendEntriesReqEvent to this server
				// after it returns to FOLLOWER state

				return FOLLOWER //return to follower state
			}

			if DebugCandidate {
				log.Printf("%v:: ** CANDIDATE ** AppendEntriesReqEvent Rejected %v (Term = %v) -> %v (Term = %v) \n", time.Now().Nanosecond(), msg.LeaderId, msg.Term, sm.Id(), sm.Term)
			}
			// If the term in the RPC is smaller than the candidate's term, then the
			// candidate reject the RPC and continues in candidate state
			sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.ID, FollowerIndex: (len(sm.Log) - 1)}}

		case VoteReqEvent:
			//			if DebugCandidate {
			//				log.Printf("** CANDIDATE ** Server ID = %v  Term = %v  and event = %v \n", sm.Id(), sm.Term, event.(VoteReqEvent).getEventName())}
			msg := event.(VoteReqEvent)

			// If the other candidate's term is less than or equal to my term, then Reject the VOTE
			// (For equal case we are rejecting because we have already voted for self)
			if msg.Term <= sm.Term {
				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** VoteReqEvent VOTE REJECTED  %v (Term = %v) ->  %v (Term = %v) msg term is lower or equal \n", time.Now().Nanosecond(), sm.Id(), sm.Term, msg.CandidateId, msg.Term)
				}
				sm.SendChannel <- Send{int(msg.CandidateId), VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: false}} // Reject the Vote

			} else {
				// In this case since the other candidate is more updated than myself
				// I will update my term and give the candidate my vote if I have not already voted
				// for any other candidate (other than myself) else reject this vote and
				// return to FOLLOWER state

				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** VoteReqEvent VOTE GRANTED  %v (Term = %v) ->  %v (Term = %v) CANDIDATE->FOLLOWER \n", time.Now().Nanosecond(), sm.Id(), msg.CandidateId, msg.Term, sm.Term)
				}
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer

				sm.Term = msg.Term
				sm.VotedFor = msg.CandidateId
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.ID, Term: sm.Term, VoteGranted: true}} // Give the vote

				//				// not voted OR already voted for same candidate OR already voted for SELF
				//				if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) {
				//					sm.VotedFor = int(msg.CandidateId)
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: true}} // Give the vote
				//				} else {
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Term: sm.Term, VoteGranted: false}} // Reject the Vote
				//				}

				return FOLLOWER
			}

		case AppendEntriesRespEvent:
			msg := event.(AppendEntriesRespEvent)
			if DebugCandidate {
				log.Printf("%v:: ** CANDIDATE ** AppendEntriesRespEvent %v (Term = %v) -> %v (Term = %v) \n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term)
			}
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
				if DebugCandidate {
					log.Printf("%v:: ** CANDIDATE ** AppendEntriesRespEvent %v (Term = %v) -> %v (Term = %v) CANDIDATE->FOLLOWER\n", time.Now().Nanosecond(), msg.FollowerId, msg.Term, sm.Id(), sm.Term)
				}
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}

				return FOLLOWER
			}

		case ShutDownEvent:
			msg := event.(ShutDownEvent)
			sm.SendChannel <- Alarm{Time: HighTimeout} // remove the timer
			sm.QuitChannel <- true
			if DebugCandidate {
				log.Printf("%v:: Server %v Shutting down...\n", time.Now().Nanosecond(), msg.Id)
			}
			return POWEROFF
		}
	}
	return CANDIDATE
}
