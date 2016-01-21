package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

//structure to store file details
type file struct {
	version          int64
	fileContent      []byte
	fileCreationTime time.Time
	fileLife         int
	expirySpecified  bool
}


//structure for sending data onto the channel
type ChannelCommand struct {
	cmdType    string
	filename   string
	fileStruct file
	result     chan string
}


const PORT = ":8080"

/*
* This function parses the clients command and pass it to channelHandler function to serialize 
* concurrent access to common data structure
*/
func handleClients(commands chan ChannelCommand, con net.Conn) {
	defer con.Close()

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
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //for invalid filename close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				numbytes, err := strconv.Atoi(fs[2])
				if err != nil { //for invalid filename close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
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
					break
				}
				//next two bytes after the content should be \r\n
				if string(tmp) != "\r\n" { //for invalid command close the connection
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				var version int64 = 1
				fileStructData := file{version: version, fileContent: content, fileCreationTime: curTime, fileLife: expTime, expirySpecified: expiryFound}
				result := make(chan string)
				//sending structure onto the channel
				commands <- ChannelCommand{
					cmdType:    "write",
					filename:   filename,
					fileStruct: fileStructData,
					result:     result,
				}
				io.WriteString(con, <-result)
			case "read": //hanldes the read command
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				result := make(chan string)
				commands <- ChannelCommand{ //sends the data onto the channel
					cmdType:  "read",
					filename: filename,
					result:   result,
				}
				io.WriteString(con, <-result) //receive the data from the channel
			case "cas": //handles compare and swap command
				if len(fs) < 4 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				filename := fs[1]
				if len(filename) > 250 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				var fileVersion int64
				fileVersion, err := strconv.ParseInt(fs[2], 10, 64) //10 for base decimal
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				numbytes, err := strconv.Atoi(fs[3])
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
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
						break
					}
					if expTime == 0 {
						expiryFound = false
					}
				}
				curTime := time.Now()
				content := make([]byte, numbytes)
				readError := false
				for i := 1; i <= numbytes; i++ {//read the specified number of bytes
					content[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}

				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
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
					break
				}
				//after content is finished, next two bytes must be \r\n else command is not valid
				if string(tmp) != "\r\n" {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				fileStructData := file{version: fileVersion, fileContent: content, fileCreationTime: curTime, fileLife: expTime, expirySpecified: expiryFound}
				result := make(chan string)
				commands <- ChannelCommand{//sends the data onto the channel
					cmdType:    "cas",
					filename:   filename,
					fileStruct: fileStructData,
					result:     result,
				}
				io.WriteString(con, <-result)//receives the data from the channel
			case "delete": //handles delete command
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				result := make(chan string)
				commands <- ChannelCommand{//sends the data onto the channel
					cmdType:  "delete",
					filename: filename,
					result:   result,
				}
				io.WriteString(con, <-result)//reads the data from the channel
			default:
				io.WriteString(con, "ERR_CMD_ERR\r\n")
			}
		} else {
			io.WriteString(con, "ERR_CMD_ERR\r\n")
		}
	}
}

func channelHandler(commands chan ChannelCommand) {
	var data = make(map[string]file)
	for cmd := range commands {
		switch cmd.cmdType {
		case "write": //handles the write command
			var version int64 = 1
			if fl, found := data[cmd.filename]; found {

				if cmd.fileStruct.expirySpecified { //if the expiry time was specified or if its value was not zero
					duration := time.Since(fl.fileCreationTime)
					if duration.Seconds() > float64(fl.fileLife) { //file has already expired
						delete(data, cmd.filename)
						data[cmd.filename] = cmd.fileStruct
					} else {
						fl.version = fl.version + 1
						version = fl.version
						fl.fileContent = cmd.fileStruct.fileContent
						fl.fileCreationTime = cmd.fileStruct.fileCreationTime
						fl.fileLife = cmd.fileStruct.fileLife //duration in seconds after which file will expire
						fl.expirySpecified = cmd.fileStruct.expirySpecified
						data[cmd.filename] = fl
					}

				} else {
					fl.version = fl.version + 1
					version = fl.version
					fl.fileContent = cmd.fileStruct.fileContent
					fl.fileCreationTime = cmd.fileStruct.fileCreationTime
					fl.fileLife = cmd.fileStruct.fileLife //duration in seconds after which file will expire
					fl.expirySpecified = cmd.fileStruct.expirySpecified
					data[cmd.filename] = fl
				}

			} else {
				data[cmd.filename] = cmd.fileStruct
			}
			cmd.result <- "OK " + strconv.FormatInt(version, 10) + "\r\n"
		case "read": //handles the read command
			if fl, found := data[cmd.filename]; found {
				if fl.expirySpecified {
					duration := time.Since(fl.fileCreationTime)
					if duration.Seconds() > float64(fl.fileLife) { //file has already expired
						delete(data, cmd.filename)
						cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
						continue
					}

					remainingSeconds := fl.fileLife - int(duration.Seconds())
					cmd.result <- "CONTENTS " + strconv.FormatInt(fl.version, 10) + " " + strconv.Itoa(len(fl.fileContent)) + " " + strconv.Itoa(remainingSeconds) + "\r\n" + string(fl.fileContent) + "\r\n"
				} else {
					cmd.result <- "CONTENTS " + strconv.FormatInt(fl.version, 10) + " " + strconv.Itoa(len(fl.fileContent)) + " " + strconv.Itoa(fl.fileLife) + "\r\n" + string(fl.fileContent) + "\r\n"
				}
			} else {
				cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
			}
		case "cas": //handles the compare and swap command
			var fileVersion int64 = 1
			if fl, found := data[cmd.filename]; found {
				if cmd.fileStruct.expirySpecified { //file has already expired
					duration := time.Since(fl.fileCreationTime)
					if duration.Seconds() > float64(fl.fileLife) {
						delete(data, cmd.filename)
						cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
						continue
					}
				}
				if fl.version != cmd.fileStruct.version {
					cmd.result <- "ERR_VERSION " + strconv.FormatInt(fl.version, 10) + "\r\n"
					continue
				}
				fl.version += 1
				fileVersion = fl.version
				fl.fileContent = cmd.fileStruct.fileContent
				fl.fileCreationTime = cmd.fileStruct.fileCreationTime
				fl.fileLife = cmd.fileStruct.fileLife
				fl.expirySpecified = cmd.fileStruct.expirySpecified
				data[cmd.filename] = fl
			} else {
				cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
				continue
			}
			cmd.result <- "OK " + strconv.FormatInt(fileVersion, 10) + "\r\n"
		case "delete": //handles the delete command
			if fl, found := data[cmd.filename]; found {
				if fl.expirySpecified {
					duration := time.Since(fl.fileCreationTime)
					if duration.Seconds() > float64(fl.fileLife) { //file has already expired
						delete(data, cmd.filename)
						cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
						continue
					}
				}
				delete(data, cmd.filename)
				cmd.result <- "OK\r\n"
			} else {
				cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
				continue
			}
		}
	}
}

func serverMain() {
	ls, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatalln(err)
	}
	defer ls.Close()

	commands := make(chan ChannelCommand)
	go channelHandler(commands)

	for {
		con, err := ls.Accept() //accepting the TCP connection
		if err != nil {
			log.Fatalln(err)
		}
		go handleClients(commands, con) //func to parse the client commands and pass its data onto the channel
	}
}

func main() {
	serverMain()
}
