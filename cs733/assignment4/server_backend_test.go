package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Simple serial check of getting and setting
func TestTCPSimple(t *testing.T) {
	deleteLogs(5)
	cfg := initializeSystem("config3.json")

	conn, _ := getLeaderConnection(t, cfg, "TestTCPSimple_1")
	defer conn.Close()

	name := "hi.txt"
	contents := "bye"
	exptime := 300000
	reader := bufio.NewReader(conn)

	//junk command
	fmt.Fprintf(conn, "junk\r\n")
	arr, _, errr := clientRead(t, reader, "junk", "junk")
	if errr != "" {
		expect(t, arr[0], "ERR_CMD_ERR")
	}

	//junk command
	fmt.Fprintf(conn, "junk junk\r\n")
	arr, _, errr = clientRead(t, reader, "junk", "junk")
	if errr != "" {
		expect(t, arr[0], "ERR_CMD_ERR")
	}

	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), "1", contents)
	arr, _, errr = clientRead(t, reader, "write", "WRITE")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}
	expect(t, arr[0], "OK")
	ver, err := strconv.Atoi(arr[1]) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	version := int64(ver)

	time.Sleep(1 * time.Second)

	// CAS on a expired file
	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version, len(contents), exptime, contents)
	_, _, errr = clientRead(t, reader, "cas", "CAS")
	if errr != "" {
		expect(t, errr, "ERR_FILE_NOT_FOUND")
	}

	//delete on expired file
	fmt.Fprintf(conn, "delete %v\r\n", name) // try a read now
	_, _, errr = clientRead(t, reader, "delete", "DELETE")
	if errr != "" {
		expect(t, errr, "ERR_FILE_NOT_FOUND")
	}

	// Write this command will be treated as new write command as file as expired already
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), "2", contents)
	arr, _, errr = clientRead(t, reader, "write", "WRITE")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}

	//write this command will update the version and contents of previous file itself
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	arr, _, errr = clientRead(t, reader, "write", "WRITE")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}
	expect(t, arr[0], "OK")
	ver, err = strconv.Atoi(arr[1]) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	version = int64(ver)

	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	arr, content, errr := clientRead(t, reader, "read", "READ")

	if errr != "" {
		t.Error("Error occur in reading the file, error = ", errr)
	}

	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version)) // expect only accepts strings, convert int version to string
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))

	expect(t, contents, content)

	//exptime = 300
	// CAS a file
	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version, len(contents), exptime, contents)
	arr, _, errr = clientRead(t, reader, "cas", "CAS")
	if errr != "" {
		t.Error("Error occur in cascading the file, error = ", errr)
	}

	expect(t, arr[0], "OK")
	ver, err = strconv.Atoi(arr[1]) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	version = int64(ver)

	//read
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	arr, content, errr = clientRead(t, reader, "read", "READ")

	if errr != "" {
		t.Error("Error occur in reading the file, error = ", errr)
	}

	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version)) // expect only accepts strings, convert int version to string
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	expect(t, contents, content)

	//delete the file
	fmt.Fprintf(conn, "delete %v\r\n", name) // try a read now
	time.Sleep(1 * time.Second)
	arr, _, errr = clientRead(t, reader, "delete", "DELETE")
	
	if errr != "" {
		t.Error("Error occur in deleting the file, error = ", errr)
	}
	expect(t, arr[0], "OK")

	//read
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	_, _, errr = clientRead(t, reader, "read", "READ")

	if errr != "" {
		expect(t, errr, "ERR_FILE_NOT_FOUND")
	}

}

func closeAllServers(t *testing.T, cfg *NewConfig) {
	numServers := len(cfg.Peers)

	for i := 0; i < numServers; i++ {
		address := strings.Split(cfg.Peers[i].Address, ":")[0] + cfg.Peers[i].ClientPort
		conn := getConnection(t, address)
		fmt.Fprintf(conn, "shutdown \r\n")
		time.Sleep(4 * time.Second)
	}

	deleteLogs(5)

}

/*
* This function runs multiple clients which execute "cas" command on single file
* only one of the client should be successful in updating the file,
* others should get ERR_VERSION error
 */

func TestCasConcurrently(t *testing.T) {
	deleteLogs(5)
	cfg := initializeSystem("config1.json")

	filename := "f1.txt"
	contents := "0"

	con, leaderAddress := getLeaderConnection(t, cfg, "TestCasConcurrently")
	con.Close()

	done := make(chan bool, 10)

	conn := getConnection(t, leaderAddress)
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", filename, len(contents), 0, contents)
	time.Sleep(2 * time.Second)
	arr, _, errr := clientRead(t, reader, "write", "OO")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}

	version := arr[1]
	_, err := strconv.Atoi(arr[1]) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}

	//testing for concurrent cascading of clients on same file
	for i := 1; i <= 2; i++ {
		con1, _ := getLeaderConnection(t, cfg, "TestCasConcurrently") //getConnection(t, leaderAddress)
		defer con1.Close()
		reader1 := bufio.NewReader(con1)
		//running concurrent cascading commands
		go concurrentCasInner(t, con1, reader1, done, filename, version, contents)
	}

	// Wait for tests to finish
	for i := 1; i <= 2; i++ {
		<-done
	}

	//read
	fmt.Fprintf(conn, "read %v\r\n", filename) // try a read now
	arr, content, errr := clientRead(t, reader, "read", "LL")
	if errr != "" {
		t.Error("Error occured in reading the file, error = " + errr)
	}
	newVersion := arr[1]
	_, err = strconv.Atoi(arr[1]) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}

	expect(t, content, "1")
	notExpect(t, newVersion, version)
}

/*
* This function runs for each concurrent client trying to use cas operation on the same file
* Only one client will be able to execute the command correctly, others will get VERSION MISMATCH ERROR
 */
func concurrentCasInner(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, filename string, version string, content string) {
	value, err := strconv.Atoi(content)
	if err != nil {
		t.Error(err.Error())
	}
	value = value + 1

	fileContents := strconv.Itoa(value)

	// CAS a file
	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", filename, version, len(fileContents), "0", fileContents)
	time.Sleep(1 * time.Second)
	arr, _, errr := clientRead(t, reader, "cas", "CAS")
	if errr != "" {
		expect(t, errr, "ERR_VERSION")
		_, err := strconv.Atoi(arr[1]) // parse version as number
		if err != nil {
			t.Error("Non-numeric version found", arr[1])
		}
	}

	done <- true
}

/*
* This will test server ability to be able to handle lots of writes
* and cascading commands simultanously on the single file.
 */
func TestCasWritesConcurrently(t *testing.T) {
	deleteLogs(5)
	cfg := initializeSystem("config2.json")

	filename := "f1.txt"
	contents := "aaa"
	//	go serverMain()
	//	time.Sleep(1 * time.Second) // one second is enough time for the server to start

	done := make(chan bool, 10)

	clientContents := []struct{ content string }{{"ONE"}} //, {"TWO"}}//, {"THREE"}, {"FOUR"}, {"FIVE"}, {"SIX"}, {"SEVEN"}, {"EIGHT"}, {"NINE"}, {"TEN"}}

	conn, _ := getLeaderConnection(t, cfg, "TestCasWritesConcurrently")
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", filename, len(contents), 0, contents)
	time.Sleep(4 * time.Second)
	arr, _, errr := clientRead(t, reader, "write", "OO")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}

	//testing for concurrent cascading of clients on same file
	for _, e := range clientContents {
		con1, _ := getLeaderConnection(t, cfg, "TestCasWritesConcurrently") //getConnection(t, leaderAddress)
		defer con1.Close()
		reader1 := bufio.NewReader(con1)
		//running concurrent cascading commands
		go concurrentCaswithDifferentContent(t, con1, reader1, done, filename, arr[1], e.content)
		con2, _ := getLeaderConnection(t, cfg, "TestCasWritesConcurrently") //getConnection(t, leaderAddress)
		defer con2.Close()
		reader2 := bufio.NewReader(con2)
		//running concurrent write commands
		go concurrentWritewithDifferentContent(t, con2, reader2, done, filename, arr[1], e.content)
	}

	// Wait for tests to finish
	for i := 1; i <= 2*len(clientContents); i++ {
		<-done
	}
	//read
	fmt.Fprintf(conn, "read %v\r\n", filename) // try a read now
	arr, content, errr := clientRead(t, reader, "read", "LL")
	if errr != "" {
		t.Error("Error occured in reading the file, error = " + errr)
	}

	found := false
	for _, e := range clientContents {
		val := e.content + "5"
		if val == content {
			found = true
			break
		}
	}
	//It tests if the final content of the file is one of last writes or compare and swap
	if !found {
		t.Error("Problem in compare and swap routines", content)
	}
}

/*
* This function will run many write commands in a loop on the same file.
* This is done concurrently with cascading commands
 */
func concurrentWritewithDifferentContent(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, filename string, version string, contents string) {
	for i := 1; i <= 5; i++ {
		fileContents := contents + strconv.Itoa(i)

		// write a file
		fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", filename, len(fileContents), "0", fileContents)
		time.Sleep(4 * time.Second)
		arr, _, errr := clientRead(t, reader, "cas", contents)
		if errr != "" {
			t.Error("Error occured in writing the file, error = " + errr)
			break
		}
		notExpect(t, arr[1], version)
		version = arr[1]
	}
	done <- true
}

/*
* This function will run many cas commands in a loop on the same file.
* This is done concurrently with write commands
 */
func concurrentCaswithDifferentContent(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, filename string, version string, contents string) {

	for i := 1; i <= 5; i++ {
		for {

			fileContents := contents + strconv.Itoa(i)
			// CAS a file
			fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", filename, version, len(fileContents), "0", fileContents)
			time.Sleep(1 * time.Second)
			arr, _, errr := clientRead(t, reader, "cas", contents)

			if errr == "" {
				version = arr[1]
				break
			}

			expect(t, errr, "ERR_VERSION")

			_, err := strconv.Atoi(arr[1]) // parse version as number
			if err != nil {
				t.Error("Non-numeric version found ")
			}
			version = arr[1]
		}
	}
	done <- true
}



/*
* This function is test server capabilties to handle load and concurrency
 */

func TestServerConcurrently(t *testing.T) {
	deleteLogs(5)
	done := make(chan bool, 10)
	cfg := initializeSystem("config4.json")

	clientContents := []struct{ content string }{{"ONE"}} 

	filename := "f"
	i := 1

	//do concurrency test
	for _, e := range clientContents {
		con, _ := getLeaderConnection(t, cfg, "TestServerConcurrently") //getConnection(t, leaderAddress)
		defer con.Close()
		reader := bufio.NewReader(con)
		go concurrent(t, con, reader, done, filename+strconv.Itoa(i), e.content, cfg)
		i++
	}

	// Wait for tests to finish
	for i := 1; i <= len(clientContents); i++ {
		<-done
	}
}

//writes a file and then creates other go routines to race on it
func concurrent(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, filename string, contents string, cfg *NewConfig) {
	done2 := make(chan bool, 10)
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", filename, len(contents), "0", contents)
	time.Sleep(1 * time.Second)
	arr, _, errr := clientRead(t, reader, "write", "WRITE")
	if errr != "" {
		t.Error("Error occur in writing the file, error = ", errr)
	}

	//below go routines will be called to perform concurrent read, write, cas and delete command
	// on a single file
	version := arr[1]
	con1, _ := getLeaderConnection(t, cfg, "TestServerConcurrently") // getConnection(t, leaderAddress)
	defer con1.Close()
	reader1 := bufio.NewReader(con1)
	go concurrentWrite(t, con1, reader1, done2, version, filename, contents)

	con2, _ := getLeaderConnection(t, cfg, "TestServerConcurrently") //getConnection(t, leaderAddress)
	defer con2.Close()
	reader2 := bufio.NewReader(con2)
	go concurrentRead(t, con2, reader2, done2, version, filename, contents)

	con3, _ := getLeaderConnection(t, cfg, "TestServerConcurrently") //getConnection(t, leaderAddress)
	defer con3.Close()
	reader3 := bufio.NewReader(con3)
	go concurrentDelete(t, con3, reader3, done2, version, filename, contents)

	for i := 1; i <= 3; i++ {
		<-done2
	}
	done <- true
}

/*
* This function will perform write on the file, which is being concurrently accessed by
* read, cas and delete commands.
 */

func concurrentWrite(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, oldVersion string, filename string, contents string) {
	for i := 1; i <= 2; i++ {
		fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", filename, len(contents), "0", contents)
		time.Sleep(1 * time.Second)
		_, _, errr := clientRead(t, reader, "write", "WRITE")
		if errr != "" {
			t.Error("Error occur in writing the file, error = ", errr)
		}
		//notExpect(t, oldVersion, arr[1])
	}
	done <- true
}

/*
* This function will perform read on the file, which is being concurrently accessed by
* write, cas and delete commands.
 */

func concurrentRead(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, oldVersion string, filename string, contents string) {
	for i := 1; i <= 2; i++ {
		fmt.Fprintf(conn, "read %v\r\n", filename) // try a read now
		time.Sleep(1 * time.Second)
		arr, content, errr := clientRead(t, reader, "read", "READ")

		if errr != "" {
			expect(t, errr, "ERR_FILE_NOT_FOUND")
			break
		}
		_, err := strconv.Atoi(arr[1]) // parse version as number
		if err != nil {
			t.Error("Non-numeric version found")
		}
		expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
		expect(t, content, fmt.Sprintf("%v", contents)) // expect only accepts strings, convert int version to string

	}
	done <- true
}

/*
* This function will perform delete on the file, which is being concurrently accessed by
* read, cas and write commands.
 */

func concurrentDelete(t *testing.T, conn net.Conn, reader *bufio.Reader, done chan bool, oldVersion string, filename string, contents string) {
	for i := 1; i <= 2; i++ {
		//delete the file
		fmt.Fprintf(conn, "delete %v\r\n", filename) // try a read now
		time.Sleep(2 * time.Second)
		arr, _, errr := clientRead(t, reader, "delete", "DELETE")
		if errr != "" {
			expect(t, errr, "ERR_FILE_NOT_FOUND")
		} else {
			if len(arr) > 0 {
				expect(t, arr[0], "OK")
			}
		}
	}
	done <- true
}


func getLeaderConnection(t *testing.T, cfg *NewConfig, identifier string) (conn net.Conn, address string) {
	address = strings.Split(cfg.Peers[0].Address, ":")[0] + cfg.Peers[0].ClientPort

	for {
		conn = getConnection(t, address)
		fmt.Fprintf(conn, "delete %v\r\n", "non-existing-file") // try a read now
		reader := bufio.NewReader(conn)
		_, _, errr := clientRead(t, reader, "delete", identifier)
		if errr != "" {
			if errr == "ERR_FILE_NOT_FOUND" {
//				fmt.Printf("Found leader\n")
				break
			}
			reason := strings.Split(errr, " ")[1]
			if reason == "No_Known_Leader" {
				time.Sleep(1 * time.Second)
				continue
			} else {
				address = reason
			}
		} else {
			t.Fatal("No file should have been found\n")
		}
	}
	return conn, address
}

func getFollowerConnection(t *testing.T, cfg *NewConfig, identifier string) (conn net.Conn) {
	address := strings.Split(cfg.Peers[0].Address, ":")[0] + cfg.Peers[0].ClientPort

	for {
		//		fmt.Printf("dddddddddddd %v*********", address)
		conn = getConnection(t, address)
		fmt.Fprintf(conn, "delete %v\r\n", "non-existing-file") // try a read now
		reader := bufio.NewReader(conn)
		_, _, errr := clientRead(t, reader, "delete", identifier)
		if errr != "" {
			if errr == "ERR_FILE_NOT_FOUND" {
				conn.Close()
				address = strings.Split(cfg.Peers[1].Address, ":")[1] + cfg.Peers[1].ClientPort
				continue
			}
			reason := strings.Split(errr, " ")[1]
			if reason == "No_Known_Leader" {
				time.Sleep(1 * time.Second)
				continue
			} else {
				conn = getConnection(t, address)
				break
			}
		} else {
			t.Fatal("No file should have been found\n")
		}
	}
	return conn
}

//utility function to get the connection
func getConnection(t *testing.T, address string) (conn net.Conn) {
	for {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			//		t.Fatal(err.Error()) // report error through testing framework
			time.Sleep(1 * time.Second)
			continue
		}
		return conn
	}
}

// Useful testing function
func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}

func notExpect(t *testing.T, a string, b string) {
	if a == b {
		t.Error(fmt.Sprintf("Not Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}

//command output reader
func clientRead(t *testing.T, reader *bufio.Reader, cmd string, identifier string) (arr []string, content string, errr string) {
	cmdBytes, isPrefix, err := reader.ReadLine()
	readError := false
	if err != nil || isPrefix == true {
		t.Error(err.Error())
		readError = true
	}
	if readError == false {
		command := string(cmdBytes)
		arr = strings.Fields(command)
		//		fmt.Println("cmd=" + cmd + " " + command + "ppppppppppp= " + identifier)
		switch cmd {
		case "write", "delete":

			if arr[0] != "OK" {
				errr = command
				break
			}
		case "cas":
			if arr[0] != "OK" {
				errr = arr[0]
				break
			}
		case "read":
			if arr[0] != "CONTENTS" {
				errr = command
				break
			}
			numb, err := strconv.Atoi(arr[2]) // parse version as number
			if err != nil {
				t.Error("Non-numeric numbytes found")
			}
			numbytes := int(numb)
			tmp := make([]byte, numbytes)
			readError = false
			for i := 1; i <= numbytes; i++ {
				tmp[i-1], err = reader.ReadByte()
				if err != nil {
					t.Error("Error in reading content of read command")
				}
			}

			if readError == false {
				content = string(tmp)
				tmp = make([]byte, 2)
				for i := 1; i <= 2; i++ {
					tmp[i-1], err = reader.ReadByte()
					if err != nil {
						t.Error("Error in reading content of read command")
					}
				}

				if string(tmp) != "\r\n" {
					t.Error("Error in reading content of read command")
				}
			}

		}
	}
	if errr != "" {
		return arr, "", errr
	} else {
		return arr, content, ""
	}
}

func deleteLogs(numServers int) {
	for i := 1; i <= numServers; i++ {
		if _, err := os.Stat("server_" + strconv.Itoa(i) + "_log"); err == nil {
			err = os.RemoveAll("server_" + strconv.Itoa(i) + "_log")
			if err != nil {
				panic(err)
			}
		}
		if _, err := os.Stat(strconv.Itoa(i) + "_state"); err == nil {
			err = os.RemoveAll(strconv.Itoa(i) + "_state")
			if err != nil {
				panic(err)
			}
		}
	}
}
