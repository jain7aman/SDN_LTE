## Assignment 3- RAFT Node

Third assignment for course CS733, Spring-2015  
Submitted by: Aman Jain, Roll: 143050020

### Introduction
* In this assignment we have build a "raft Node", a wrapper over the state machine
* The instance is designed in Go language.    
* It can handle multiple types of events and generate multiple actions respectively.
* Automated testing is provided with Go's testing framework

### Installation Instructions
<code>go get </code> github.com/jain7aman/cs733/assignment3

Five files are supposed to be there <br/>
1. node.go contains the code for all the structure variables state machine is using. <br/>
2. follower.go, candidate.go and leader.go contains the code for respective states of the state machine. <br/>
3. raft_test.go contains all the test cases for testing all the possible scenarios. <br/>
4. raft.go contains all the initialization and handler method of wrapper node for the state machine. <br/>

To run the program only below command is needed (assuming the current directory is set to the assignment2 which has the go files)
<br/><code>go test</code>



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

* ```ShutDownEvent:``` An event which shuts down the running state machine.

### Output Actions
* ```Send(peerId, event):``` Send this event to a remote node. The event is one of AppendEntriesReq/Resp or VoteReq/Resp.Clearly the state machine needs to have some information about its peer node ids and its own id.

* ```Commit(index, data, err):``` Deliver this commit event (index + data) or report an error (data + err) to the layer above. This is the response to the Append event.

* ```Alarm(t):``` Send a Timeout after t milliseconds.

* ```LogStore(index, term, data []byte):``` This is an indication to the node to store the data at the given index. Note that data here can refer to the client’s

* ```StateStore(term, votedFor):``` This action is fired whenever one of term or votedFor for a server changes and it indicates that this change needs to stored persistently.


### Testing

For running the automated testing procedure, use the following command:
``` go test ```

System has been tested against various scenarios which include leader failures, Append request, elections, network partitions and reunions. These test cases thoroughly testing the system against various corner cases and also testing the normal functionality of the sytem to be able to provide fault resistent system. 

