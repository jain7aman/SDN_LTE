package main

func (sm *RaftServer) follower() State {
	//Step 1: sets a random timer for the timeout event of follower
	sm.SendChannel <- Alarm{Time: sm.Configuration.ElectionTimeout} //sleep for RANDOM time

	//Step 2: loops through the events being received from event channel
	//	event := <-sm.ReceiveChannel
	for event := range sm.ReceiveChannel {
		switch event.(type) {
		case TimeoutEvent:
			sm.Term++ //increments the term number
			sm.VotedFor = -1
			sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			sm.SendChannel <- NoAction{}
			return CANDIDATE //returns to CANDIDATE state

		case VoteReqEvent:
			//Step 1: reset the timer for Timeout event, as the network is alive
			sm.SendChannel <- Alarm{Time: sm.Configuration.ElectionTimeout} //sleep for RANDOM time

			msg := event.(VoteReqEvent)
			if sm.Term > msg.Term {
				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}}
				sm.SendChannel <- NoAction{}
				//continue
			} else {
				termChanged := false
				if sm.Term < msg.Term {
					sm.Term = msg.Term
					termChanged = true
				}

				lastTerm := uint(0)
				lastIndex := -1
				if len(sm.Log) > 0 {
					lastTerm = sm.Log[len(sm.Log)-1].Term
					lastIndex = len(sm.Log) - 1
				}

				// Voting server V denies vote if its log is “more complete”
				if (lastTerm > msg.LastLogTerm) || (lastTerm == msg.LastLogTerm && lastIndex > msg.LastLogIndex) {
					if termChanged == true {
						sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
					}
					sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}} // Reject the Vote
				} else {
					if sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId) {
						sm.VotedFor = int(msg.CandidateId)
						sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
						sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}} // Give the Vote
					} else {
						if termChanged == true {
							sm.VotedFor = -1
							sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
						}
						sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}} // Already voted for other, Reject the Vote
					}
				}

				//check if candidates log is as upto to date as follower's term
				//			if sm.Term == msg.Term && (sm.VotedFor == -1 || sm.VotedFor == int(msg.CandidateId)) && len(sm.Log)-1 <= msg.LastLogIndex {
				//				sm.VotedFor = int(msg.CandidateId)
				//				if termChanged == true {
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//				}
				//				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: true}} // Give the Vote
				//			} else {
				//				if termChanged == true {
				//					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				//				}
				//				sm.SendChannel <- Send{msg.CandidateId, VoteRespEvent{Id: sm.Id, Term: sm.Term, VoteGranted: false}} // Reject the Vote
				//			}
				sm.SendChannel <- NoAction{}
			}

		case AppendEntriesReqEvent:
			msg := event.(AppendEntriesReqEvent)

			if msg.Term < sm.Term { //Followers term is greater than leader's term
				sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}} //Leader is old and not uptodate
				sm.SendChannel <- NoAction{}
				// continue //continue the events loop
			} else {

				if msg.Term > sm.Term {
					sm.Term = msg.Term
					sm.VotedFor = -1
					sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
				}

				// Reset the timer for Timeout event, as the network is alive
				sm.SendChannel <- Alarm{Time: sm.Configuration.ElectionTimeout} //sleep for RANDOM time

				// If length of log is smaller at follower then its previous log index wont match with the prevLogIndex of Leader
				if len(sm.Log)-1 < msg.PrevLogIndex {
					sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}}
					sm.SendChannel <- NoAction{}
					//			continue //continue the events loop
				} else {

					termMatch := true
					if len(sm.Log) > 0 {
						//Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogIndex
						if sm.Log[msg.PrevLogIndex].Term != msg.PrevLogTerm {
							sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: false, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}}
							termMatch = false
							sm.SendChannel <- NoAction{}
							//continue //continue the events loop
						}
					}
					if termMatch == true {
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

						if msg.LeaderCommit > sm.CommitIndex {
							//set commit index = min(leaderCommit, index of last new entry)
							sm.CommitIndex = minimum(msg.LeaderCommit, len(sm.Log)-1)
						}
						sm.SendChannel <- Send{msg.LeaderId, AppendEntriesRespEvent{Term: sm.Term, Success: true, FollowerId: sm.Id, FollowerIndex: (len(sm.Log) - 1)}} //send a success response back to leader
						sm.SendChannel <- NoAction{}
					}
				}
			}
		case AppendEvent:
			//if a client contacts a follower, the follower redirects it to the leader
			msg := event.(AppendEvent)
			//Send a response back to client saying that I am not the Leader and giving client the leader ID to contact to
			sm.SendChannel <- Commit{Data: msg.Command, Err: &AppendError{LeaderId: sm.LeaderId, Prob: "ERR_REDIRECTION"}} //send a success response back to leader
			sm.SendChannel <- NoAction{}

		case AppendEntriesRespEvent:
			msg := event.(AppendEntriesRespEvent)
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			}
			sm.SendChannel <- NoAction{}

		case VoteRespEvent:
			msg := event.(VoteRespEvent)
			if msg.Term > sm.Term { //Followers term is greater than my term
				sm.Term = msg.Term
				sm.VotedFor = -1
				sm.SendChannel <- StateStore{Term: sm.Term, VotedFor: sm.VotedFor}
			}
			sm.SendChannel <- NoAction{}

		}
	}
	return FOLLOWER
}
