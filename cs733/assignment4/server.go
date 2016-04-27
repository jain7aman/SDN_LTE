package main

import (
	"bufio"
	"encoding/json"
	"github.com/cs733-iitb/cluster"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//structure to store file details
type file struct {
	Version          int64
	FileContent      []byte
	FileCreationTime time.Time
	FileLife         int
	ExpirySpecified  bool
}

//structure for sending data onto the channel
type ChannelCommand struct {
	ClientId   int
	MessageId  int
	CmdType    string
	Filename   string
	FileStruct file
}

type NewConfig struct {
	Peers      []PeerConfig
	InboxSize  int
	OutboxSize int
}

type PeerConfig struct {
	Id         int
	Address    string
	ClientPort string
}

func ToConfig(configFile string) (config *NewConfig, err error) {
	var cfg NewConfig
	var f *os.File

	if f, err = os.Open(configFile); err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

/*
* This function parses the clients command and pass it to clientHandler function to serialize
* concurrent access to common data structure
 */
func parseCommand(commands chan ChannelCommand, clientId int, con net.Conn, connectionMap map[int]net.Conn, serverAddress string,
	channelMap map[int]chan string, shutDownChannel chan string, serverShutDown chan string) {

	defer con.Close()
	result := make(chan string)
	channelMap[clientId] = result
	reader := bufio.NewReader(con)
	for {
		cmdBytes, isPrefix, err := reader.ReadLine() // read the first line of the input command
		if err != nil || isPrefix == true {
			io.WriteString(con, "ERR_CMD_ERR\r\n")
			continue
		}

		command := string(cmdBytes)
		if len(command) > 500 { //for invalid command having more number of bytes than expected close the client connection
			io.WriteString(con, "ERR_CMD_ERR\r\n")
			con.Close()
			break
		}
		fs := strings.Fields(command)

		if len(fs) >= 2 {
			switch fs[0] {
			case "write": //hanldes the write command
				if len(fs) < 3 { //for invalid command close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //for invalid filename close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				numbytes, err := strconv.Atoi(fs[2])
				if err != nil { //for invalid filename close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				expTime := 0
				expiryFound := false
				if len(fs) == 4 { //check if the expiry time is specified or not
					expiryFound = true
					expTime, err = strconv.Atoi(fs[3])
					if err != nil || expTime < 0 {
						io.WriteString(con, "ERR_CMD_ERR\r\n")
						con.Close()
						delete(connectionMap, clientId)
						break
					}
					if expTime == 0 {
						expiryFound = false
					}
				}

				curTime := time.Now() //get the current time
				content := make([]byte, numbytes)

				readError := false
				//read the specified number of content bytes
				for i := 1; i <= numbytes; i++ {
					content[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}

				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				tmp := make([]byte, 2)
				readError = false
				for i := 1; i <= 2; i++ {
					tmp[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}
				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				//next two bytes after the content should be \r\n
				if string(tmp) != "\r\n" { //for invalid command close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				var version int64 = 1
				fileStructData := file{Version: version, FileContent: content, FileCreationTime: curTime, FileLife: expTime, ExpirySpecified: expiryFound}
				//sending structure onto the channel
				commands <- ChannelCommand{
					ClientId:   clientId,
					CmdType:    "write",
					Filename:   filename,
					FileStruct: fileStructData,
				}
				io.WriteString(con, <-channelMap[clientId])

			case "read": //hanldes the read command
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				commands <- ChannelCommand{ //sends the data onto the channel
					ClientId: clientId,
					CmdType:  "read",
					Filename: filename,
				}
				io.WriteString(con, <-channelMap[clientId]) //receive the data from the channel

			case "cas": //handles compare and swap command
				if len(fs) < 4 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				filename := fs[1]
				if len(filename) > 250 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				var fileVersion int64
				fileVersion, err := strconv.ParseInt(fs[2], 10, 64) //10 for base decimal
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				numbytes, err := strconv.Atoi(fs[3])
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				expTime := 0
				expiryFound := false
				if len(fs) == 5 {
					expiryFound = true
					expTime, err = strconv.Atoi(fs[4])
					if err != nil || expTime < 0 {
						io.WriteString(con, "ERR_CMD_ERR\r\n")
						con.Close()
						delete(connectionMap, clientId)
						break
					}
					if expTime == 0 {
						expiryFound = false
					}
				}
				curTime := time.Now()
				content := make([]byte, numbytes)
				readError := false
				for i := 1; i <= numbytes; i++ { //read the specified number of bytes
					content[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}

				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				tmp := make([]byte, 2)
				readError = false
				for i := 1; i <= 2; i++ {
					tmp[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}
				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				//after content is finished, next two bytes must be \r\n else command is not valid
				if string(tmp) != "\r\n" {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				fileStructData := file{Version: fileVersion, FileContent: content, FileCreationTime: curTime, FileLife: expTime, ExpirySpecified: expiryFound}
				commands <- ChannelCommand{ //sends the data onto the channel
					ClientId:   clientId,
					CmdType:    "cas",
					Filename:   filename,
					FileStruct: fileStructData,
				}
				io.WriteString(con, <-channelMap[clientId]) //receives the data from the channel

			case "delete": //handles delete command
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					delete(connectionMap, clientId)
					break
				}

				commands <- ChannelCommand{ //sends the data onto the channel
					ClientId: clientId,
					CmdType:  "delete",
					Filename: filename,
				}
				io.WriteString(con, <-channelMap[clientId]) //reads the data from the channel
			default:
				io.WriteString(con, "ERR_CMD_ERR\r\n")
			}
		} else {
			if len(fs) == 1 && fs[0] == "shutdown" {
				commands <- ChannelCommand{ //sends the data onto the channel
					ClientId: clientId,
					CmdType:  "shutdown",
					Filename: "",
				}
//				log.Printf("Shutdown command received\n")
				shutDownChannel <- "shutdown" // for the backend channel to shutdown
				time.Sleep(100 * time.Millisecond)
				msg := "shutdown"
				select {
				case serverShutDown <- msg: // for the main server to shutdown
				default:
				}
				// create a dummmy client to connect to this server, to unbloack the Accept call
				conn, err := net.Dial("tcp", serverAddress)
				if err != nil {
					panic(err)
				}
				defer conn.Close()
				con.Close()
				delete(connectionMap, clientId)
				break
			} else {
				io.WriteString(con, "ERR_CMD_ERR\r\n")
			}
		}
	}
}

func handleMap(cmd ChannelCommand, sendReply bool, serverId int, data map[string]file, channelMap map[int]chan string) {
	//mutexLock.Lock()
//	log.Printf("Client ID = %v command type=%v filename = %v\n", cmd.ClientId, cmd.CmdType, cmd.Filename)
	switch cmd.CmdType {
	case "write": //handles the write command
//		log.Printf("Sever ID %v Client ID = %v----------INSIDE WRITE------------\n", serverId, cmd.ClientId)
		var version int64 = 1
		if fl, found := data[cmd.Filename]; found {
			if cmd.FileStruct.ExpirySpecified { //if the expiry time was specified or if its value was not zero
				duration := time.Since(fl.FileCreationTime)
				if duration.Seconds() > float64(fl.FileLife) { //file has already expired
					delete(data, cmd.Filename)
					data[cmd.Filename] = cmd.FileStruct
				} else {
					fl.Version = fl.Version + 1
					version = fl.Version
					fl.FileContent = cmd.FileStruct.FileContent
					fl.FileCreationTime = cmd.FileStruct.FileCreationTime
					fl.FileLife = cmd.FileStruct.FileLife //duration in seconds after which file will expire
					fl.ExpirySpecified = cmd.FileStruct.ExpirySpecified
					data[cmd.Filename] = fl
				}

			} else {
				fl.Version = fl.Version + 1
				version = fl.Version
				fl.FileContent = cmd.FileStruct.FileContent
				fl.FileCreationTime = cmd.FileStruct.FileCreationTime
				fl.FileLife = cmd.FileStruct.FileLife //duration in seconds after which file will expire
				fl.ExpirySpecified = cmd.FileStruct.ExpirySpecified
				data[cmd.Filename] = fl
			}

		} else {
			data[cmd.Filename] = cmd.FileStruct
		}
		if sendReply {
//			log.Printf("yoooooooooo\n")
			channelMap[cmd.ClientId] <- "OK " + strconv.FormatInt(version, 10) + "\r\n"
		}

	case "read": //handles the read command
//		log.Printf("Server ID = %v Client ID = %v----------INSIDE READ------------\n", serverId, cmd.ClientId)
		if fl, found := data[cmd.Filename]; found {
			if fl.ExpirySpecified {
				duration := time.Since(fl.FileCreationTime)
				if duration.Seconds() > float64(fl.FileLife) { //file has already expired
					delete(data, cmd.Filename)
					channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
					//					continue
					break
				}

				remainingSeconds := fl.FileLife - int(duration.Seconds())
				if sendReply {
					channelMap[cmd.ClientId] <- "CONTENTS " + strconv.FormatInt(fl.Version, 10) + " " + strconv.Itoa(len(fl.FileContent)) + " " + strconv.Itoa(remainingSeconds) + "\r\n" + string(fl.FileContent) + "\r\n"
				}
			} else {
				if sendReply {
					channelMap[cmd.ClientId] <- "CONTENTS " + strconv.FormatInt(fl.Version, 10) + " " + strconv.Itoa(len(fl.FileContent)) + " " + strconv.Itoa(fl.FileLife) + "\r\n" + string(fl.FileContent) + "\r\n"
				}
			}
		} else {
			if sendReply {
				channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
			}
		}

	case "cas": //handles the compare and swap command
//		log.Printf("Server ID = %v Client ID = %v----------INSIDE CAS------------\n", serverId, cmd.ClientId)
		var fileVersion int64 = 1
		if fl, found := data[cmd.Filename]; found {
			if cmd.FileStruct.ExpirySpecified { //file has already expired
				duration := time.Since(fl.FileCreationTime)
				if duration.Seconds() > float64(fl.FileLife) {
					delete(data, cmd.Filename)
					if sendReply {
						channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
					}
					//					continue
					break
				}
			}
			if fl.Version != cmd.FileStruct.Version {
				if sendReply {
					channelMap[cmd.ClientId] <- "ERR_VERSION " + strconv.FormatInt(fl.Version, 10) + "\r\n"
				}
				//				continue
				break
			}
			fl.Version += 1
			fileVersion = fl.Version
			fl.FileContent = cmd.FileStruct.FileContent
			fl.FileCreationTime = cmd.FileStruct.FileCreationTime
			fl.FileLife = cmd.FileStruct.FileLife
			fl.ExpirySpecified = cmd.FileStruct.ExpirySpecified
			data[cmd.Filename] = fl
		} else {
			if sendReply {
				channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
			}
			//			continue
			break
		}
		channelMap[cmd.ClientId] <- "OK " + strconv.FormatInt(fileVersion, 10) + "\r\n"

	case "delete": //handles the delete command
//		log.Printf("Server ID = %v Client ID = %v ----------INSIDE DELETE------------\n", serverId, cmd.ClientId)
		if fl, found := data[cmd.Filename]; found {
			if fl.ExpirySpecified {
				duration := time.Since(fl.FileCreationTime)
				if duration.Seconds() > float64(fl.FileLife) { //file has already expired
					delete(data, cmd.Filename)
					if sendReply {
						channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
					}
					//					continue
					break
				}
			}
			delete(data, cmd.Filename)
			if sendReply {
				channelMap[cmd.ClientId] <- "OK\r\n"
			}
		} else {
			if sendReply {
				channelMap[cmd.ClientId] <- "ERR_FILE_NOT_FOUND\r\n"
			}
			//			continue
			break
		}
	}
	//mutexLock.Unlock()
}

func backendHandler(sm *RaftServer, data map[string]file, addressMap map[int]string,
	connectionMap map[int]net.Conn, channelMap map[int]chan string, shutDownChannel chan string) {
	var lastMessageId int
	firstTime := true
//	log.Printf("---------backendHandler------------\n")
	done := false
	cmd := ChannelCommand{}
	for {
		select {
		case event := <-sm.ClientCommitChannel:
			commitEvent := event.(Commit)

			err := json.Unmarshal(commitEvent.Data, &cmd)

			if err != nil {
				panic(err)
			}
//			log.Printf("Client ID = %v Server ID = %v*********CMD=%v filename=%v\n", cmd.ClientId, sm.Id(), cmd.CmdType, cmd.Filename)

			if commitEvent.Err != nil {
				leaderId := commitEvent.Err.(*AppendError).getLeaderId()
				if leaderId > 0 {
//					log.Printf("ERR REDIRECTION TO client Id = %v SERVER = %v and address = %v \n", cmd.ClientId, leaderId, addressMap[leaderId])
					channelMap[cmd.ClientId] <- "ERR_REDIRECT " + addressMap[leaderId] + "\r\n"
				} else {
//					log.Printf("ERR REDIRECTION No_Known_Leader \n")
					channelMap[cmd.ClientId] <- "ERR_REDIRECT No_Known_Leader \r\n"
				}
				if con, found := connectionMap[cmd.ClientId]; found {
					time.Sleep(100 * time.Millisecond)
					con.Close()
					delete(connectionMap, cmd.ClientId)
				}
			} else {
				ignore := false
				// taking care of duplicate and out of order delivery
				if firstTime {
					firstTime = false
					lastMessageId = cmd.MessageId
				} else {
					if cmd.MessageId != lastMessageId+1 {
						//duplicate message
						if cmd.MessageId <= lastMessageId {
//							log.Printf("Server ID: %v got duplicate message with id = %v expected to %v \n", sm.Id(), cmd.MessageId, lastMessageId+1)
//							ignore = true
						}else { // out of order delivery, initial packet got dropped
//							log.Printf("Server ID: %v got out of order message with id = %v expected to %v \n", sm.Id(), cmd.MessageId, lastMessageId+1)
//							ignore = true
							// dont know what else to do ??? 
						}
					} else {
						lastMessageId = cmd.MessageId
					}
				}
				if !ignore {
					if sm.State == LEADER {
						handleMap(cmd, true, sm.Id(), data, channelMap)
					} else {
						handleMap(cmd, false, sm.Id(), data, channelMap)
					}
				}
			}

		case <-shutDownChannel:
			sm.Shutdown()
			done = true
			break
		}

		if done {
			break
		}
	}
}

func clientHandler(commands chan ChannelCommand, sm *RaftServer, data map[string]file, channelMap map[int]chan string) {
	done := false
	var messageCounter int = 0
	for cmd := range commands {
		switch cmd.CmdType {
		case "write", "cas", "delete": //handles the write command
			messageCounter++
			cmd.MessageId = messageCounter
//			log.Printf("write, cas, delete Right^^^^^^^^^^^^^^^^^^^^^^^SERVER ID = %v STATE = %v cmd type = %v and filename=%v\n", sm.Id(), sm.getState(), cmd.CmdType, cmd.Filename)
			buf, err := json.Marshal(cmd)
			if err != nil {
				panic(err)
			}
			sm.Append(buf)

		case "read":
//			log.Printf("READ Right^^^^^^^^^^^^^^^^^^^^^^^SERVER ID = %v STATE = %v cmd type = %v and filename=%v\n", sm.Id(), sm.getState(), cmd.CmdType, cmd.Filename)
			handleMap(cmd, true, sm.Id(), data, channelMap)

		case "shutdown":
			done = true
			break
		}

		if done {
			break
		}
	}
}

func restoreMap(serverId int, logArray []LogEntry, data map[string]file, channelMap map[int]chan string) {
	cmd := ChannelCommand{}
	for i := 0; i < len(logArray); i++ {
		err := json.Unmarshal(logArray[i].Command, &cmd)
		if err != nil {
			panic(err)
		}
//		log.Printf("Server id= %v, Index = %v  and command = %v file name = %v data = %v\n", serverId, i, cmd.CmdType, cmd.Filename, string(cmd.FileStruct.FileContent))
		handleMap(cmd, false, serverId, data, channelMap)
	}
}

func serverMain(serverId int, addressMap map[int]string, cfg *cluster.Config) {
	ls, err := net.Listen("tcp", ":"+strings.Split(addressMap[serverId], ":")[1])
	if err != nil {
		log.Fatalln(err)
	}
	defer ls.Close()

	shutDownChannel := make(chan string)
	serverShutDown := make(chan string, 10)

	var channelMap = make(map[int]chan string)
	var data = make(map[string]file)
	var connectionMap = make(map[int]net.Conn)

	// initialize the corresponding raft node
	sm := makeRaft(serverId, cfg)

	// initialize the backend handler
	go backendHandler(sm, data, addressMap, connectionMap, channelMap, shutDownChannel)

	// waits for the leader to get elected
	time.Sleep(1 * time.Second)

	// restore the fileserver data from leader's log
	restoreMap(serverId, sm.Log, data, channelMap)

	commands := make(chan ChannelCommand)
	go clientHandler(commands, sm, data, channelMap)

	var clientCounter int = 0
	done := false

	for {
		clientCounter++
		con, err := ls.Accept() //accepting the TCP connection
		if err != nil {
			log.Fatalln(err)
		}
		select {
		case <-serverShutDown:
			con.Close()
			done = true
			break
		default:
		}
		if done {
			break
		}
		connectionMap[clientCounter] = con
		go parseCommand(commands, clientCounter, con, connectionMap, addressMap[serverId], channelMap, shutDownChannel, serverShutDown) //func to parse the client commands and pass its data onto the channel
	}

}
