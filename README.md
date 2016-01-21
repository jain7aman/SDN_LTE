## Assignment 1- File Server Clone

### Description
All the basic functionalities including write, read, cas & delete are implemented and in working condition. Channels are used as synchronization primitives for exclusive access to the hashmap

### Installation Instructions
<code>go get </code> github.com/jain7aman/cs733/assignment1

Two files are supposed to be there <br/>
1. server.go contains the code where all the commands are implemented and where server listens for the request <br/>
2. server_test.go contains all the test cases including commands which are fired concurrently evaluating all the necessary scenarios

To run the program only below command is needed (assuming the current directory is set to the assignment1 which has the go files)
<br/><code>go test -race</code>


### Todo
1. To make the file server storage persistent