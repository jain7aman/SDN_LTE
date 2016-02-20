## Assignment 2- RAFT state machine

First assignment for course CS733, Spring-2015  
Submitted by: Aman Jain, Roll: 143050020

### Introduction
* This is a single instance of raft state machine
* The instance is designed in Go language.    
* It can handle multiple types of events and generate multiple actions respectively.
* Automated testing is provided with Go's testing framework

### Installation Instructions
<code>go get </code> github.com/jain7aman/cs733/assignment2

Five files are supposed to be there <br/>
1. raft.go contains the code for all the structure variables state machine is using. <br/>
1. follower.go, candidate.go and leader.go contains the code for respective states of the state machine. <br/>
2. raft_test.go contains all the test cases for testing all the possible scenarios

To run the program only below command is needed (assuming the current directory is set to the assignment2 which has the go files)
<br/><code>go test -race</code>



### Input Actions
The state machine will have to deal with the following events. In general, these events can come at any time, so regardless of which
state the machine is in, it reacts suitably to all events.

* ```AppendEvent(data:[]byte):``` This is a request from the layer above to append the data to the replicated log. The response is in the form of an eventual Commit action. ```[]byte``` is the command send by the client for execution.

* ```TimeoutEvent:``` A timeout event is interpreted according to the state. If the state machine is a leader, it is interpreted as a heartbeat
timeout, and if it is a follower or candidate, it is interpreted as an election timeout.

* ```AppendEntriesReqEvent:``` Message from another Raft state machine. For this and the next three event types, the parameters to the messages can be taken from the Raft paper.

* ```AppendEntriesRespEvent:``` Response from another Raft state machine in response to a previous AppendEntriesReq.

* ```VoteReqEvent:``` Message from another Raft state machine to request votes for its candidature.

* ```VoteRespEvent:``` Response to a Vote request.

### Output Actions
* ```Send(peerId, event):``` Send this event to a remote node. The event is one of AppendEntriesReq/Resp or VoteReq/Resp.Clearly the state machine needs to have some information about its peer node ids and its own id.

* ```Commit(index, data, err):``` Deliver this commit event (index + data) or report an error (data + err) to the layer above. This is the response to the Append event.

* ```Alarm(t):``` Send a Timeout after t milliseconds.

* ```LogStore(index, term, data []byte):``` This is an indication to the node to store the data at the given index. Note that data here can refer to the client’s

* ```StateStore(term, votedFor):``` This action is fired whenever one of term or votedFor for a server changes and it indicates that this change needs to stored persistently.


### Testing

For running the automated testing procedure, use the following command:
``` go test -race ```


The testing is divided into 3 parts:
* Follower's Testing
* Candidates's Testing
* Leader's Testing

Each of the above state is tested for every possible type of input events that can occur in the system. The testing is very thorough covering most of the corner cases. 
