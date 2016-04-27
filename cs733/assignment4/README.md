## Assignment 4- RAFT Integrated File System Node

Fourth assignment for course CS733, Spring-2015  
Submitted by: Aman Jain, Roll: 143050020

### Introduction
* In this assignment we have build a "raft Node", a wrapper over the state machine
* The instance is designed in Go language.    
* It can handle multiple types of events and generate multiple actions respectively.
* Automated testing is provided with Go's testing framework

### Installation Instructions
<code>go get </code> github.com/jain7aman/cs733/assignment4

Seven files are supposed to be there <br/>
1. node.go contains the code for all the structure variables state machine is using. <br/>
2. follower.go, candidate.go and leader.go contains the code for respective states of the state machine. <br/>
3. raft.go contains all the initialization and handler method of wrapper node for the state machine. <br/>
4. server.go contains all the code for the file system handler. <br/>
5. server_backend_test.go contains all the test cases for testing all the possible scenarios. <br/>

To run the program only below command is needed (assuming the current directory is set to the assignment2 which has the go files)
<br/><code>go test</code>


### Usage Instructions
First build the code by running ```go build``` and then run servers as per your configuration file. Below is the format
./assignment4 ```<server-id> <configuration-file-name>``` 
configuration file need to present in the same directory, if not include the full path to it.
Run "telnet localhost ````<client-port>```"  per server to connect to it. Commands like ```write, read``` and ```delete``` can only be execute don leader machine. If executed on follower machine a proper leader url is returned as an error message and the connection is closed for that client.



### Specification
* Write: create a file, or update the file’s contents if it already exists.
```
write <filename> <numbytes> [<exptime>]\r\n
 <content bytes>\r\n
```
exptime field is optional, it signifies the time in seconds after which the file on server 
is suppose to expire. If left unspecified or if its value is set to zero then file does not expires.

The server responds with the following:

```

OK <version>\r\n

``````
where version is a unique 64‐bit number (in decimal format) assosciated with the
filename.

* Read: Given a filename, retrieve the corresponding file:
```
read <filename>\r\n
```
The server responds with the following format if file is present at the server.
```
CONTENTS <version> <numbytes> <exptime> \r\n
 <content bytes>\r\n  
```
Here ```<exptime> ```is the remaining time in seconds left for the file after which it will expire. Zero v alue indicates that the file won't expire.

If the file is not present at the server or the file has expired then the server response is:
```
ERR_FILE_NOT_FOUND\r\n
```

* Compare and swap (cas): This replaces the old file contents with the new content
provided the version is still the same.
```
cas <filename> <version> <numbytes> [<exptime>]\r\n
 <content bytes>\r\n
```
exptime is optional and means the same thing as in "write" command.

The server responds with the new version if successful :-
```
OK <version>\r\n
```
If the file isn't found on the server or the file has expired then server response is:-
```
ERR_FILE_NOT_FOUND\r\n
```
If the version provided in the "cas" command does not match the version of the file, server
response is:-
```
ERR_VERSION <version>\r\n
```

* Delete file
```
delete <filename>\r\n
```
Server response (if successful)
```
OK\r\n
```

If the file isn't found on the server or the file has expired then server response is:-
```
ERR_FILE_NOT_FOUND\r\n
```

Apart from above mentioned errors if the above commands are not specified in proper format as mentioned, then server response is:-
```
ERR_CMD_ERR\r\n
```
After sending the above response to client, server terminates the client connection and thus forcing the client to connect again.


### RAFT Input Actions
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

### RAFT Output Actions
* ```Send(peerId, event):``` Send this event to a remote node. The event is one of AppendEntriesReq/Resp or VoteReq/Resp.Clearly the state machine needs to have some information about its peer node ids and its own id.

* ```Commit(index, data, err):``` Deliver this commit event (index + data) or report an error (data + err) to the layer above. This is the response to the Append event.

* ```Alarm(t):``` Send a Timeout after t milliseconds.

* ```LogStore(index, term, data []byte):``` This is an indication to the node to store the data at the given index. Note that data here can refer to the client’s

* ```StateStore(term, votedFor):``` This action is fired whenever one of term or votedFor for a server changes and it indicates that this change needs to stored persistently.


### Testing

For running the automated testing procedure, use the following command:
``` go test ```

System has been tested against various scenarios which include leader failures, Append request, elections, network partitions and reunions. These test cases thoroughly testing the system against various corner cases and also testing the normal functionality of the sytem to be able to provide fault resistent system. 

