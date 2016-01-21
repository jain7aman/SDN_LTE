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

type file struct {
	version          int64
	fileContent      []byte
	fileCreationTime time.Time
	fileLife         int
	expirySpecified  bool
}

type ChannelCommand struct {
	cmdType    string
	filename   string
	fileStruct file
	result     chan string
}

const PORT = ":8080"

func handleClients(commands chan ChannelCommand, con net.Conn) {
	defer con.Close()

	reader := bufio.NewReader(con)
	for {
		cmdBytes, isPrefix, err := reader.ReadLine()
		if err != nil || isPrefix == true {
			io.WriteString(con, "ERR_CMD_ERR\r\n")
			continue
		}
		command := string(cmdBytes)
		fs := strings.Fields(command)

		if len(fs) >= 2 {
			switch fs[0] {
			case "write":
				if len(fs) < 3 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				numbytes, err := strconv.Atoi(fs[2])
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}
				expTime := 0
				expiryFound := false
				if len(fs) == 4 {
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
				curTime := time.Now()
				//				duration, err := time.ParseDuration(strconv.Itoa(expTime) + "s")
				//				if err != nil {
				//					io.WriteString(con, "ERR_CMD_ERR\r\n")
				//					continue
				//				}
				//				expiryTime := curTime.Add(duration)

				content := make([]byte, numbytes)

				readError := false
				for i := 1; i <= numbytes; i++ {
					content[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}

				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
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
					continue
				}

				if string(tmp) != "\r\n" {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
					//					io.WriteString(con, "ERR_CMD_ERR\r\n")
					//					continue
				}

				var version int64 = 1
				fileStructData := file{version: version, fileContent: content, fileCreationTime: curTime, fileLife: expTime, expirySpecified: expiryFound}
				result := make(chan string)
				commands <- ChannelCommand{
					cmdType:    "write",
					filename:   filename,
					fileStruct: fileStructData,
					result:     result,
				}
				io.WriteString(con, <-result)
			case "read":
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				result := make(chan string)
				commands <- ChannelCommand{
					cmdType:  "read",
					filename: filename,
					result:   result,
				}
				io.WriteString(con, <-result)
			case "cas":
				if len(fs) < 4 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
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
				//				duration, err := time.ParseDuration(strconv.Itoa(expTime) + "s")
				//				if err != nil {
				//					io.WriteString(con, "ERR_CMD_ERR\r\n")
				//					continue
				//				}
				//				expiryTime := curTime.Add(duration)

				content := make([]byte, numbytes)
				readError := false
				for i := 1; i <= numbytes; i++ {
					content[i-1], err = reader.ReadByte()
					if err != nil {
						readError = true
						break
					}
				}

				if readError {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
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
					continue
				}

				if string(tmp) != "\r\n" {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
					//					io.WriteString(con, "ERR_CMD_ERR\r\n")
					//					continue
				}

				fileStructData := file{version: fileVersion, fileContent: content, fileCreationTime: curTime, fileLife: expTime, expirySpecified: expiryFound}
				result := make(chan string)
				commands <- ChannelCommand{
					cmdType:    "cas",
					filename:   filename,
					fileStruct: fileStructData,
					result:     result,
				}
				io.WriteString(con, <-result)
			case "delete":
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					con.Close()
					break
				}

				result := make(chan string)
				commands <- ChannelCommand{
					cmdType:  "delete",
					filename: filename,
					result:   result,
				}
				io.WriteString(con, <-result)
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
		case "write":
			var version int64 = 1
			if fl, found := data[cmd.filename]; found {

				if cmd.fileStruct.expirySpecified {
					duration := time.Since(fl.fileCreationTime)
					//					fmt.Printf("seconds %v \r\n", duration.Seconds())
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
		case "read":
			if fl, found := data[cmd.filename]; found {
				if fl.expirySpecified {
					//curTime := time.Now()
					duration := time.Since(fl.fileCreationTime)
					//					fmt.Printf("seconds %v \r\n", duration.Seconds())
					if duration.Seconds() > float64(fl.fileLife) { //file has already expired
						delete(data, cmd.filename)
						cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
						continue
					}

					//remainDuration := fl.fileCreationTime.Sub(curTime)
					remainingSeconds := fl.fileLife - int(duration.Seconds())
					cmd.result <- "CONTENTS " + strconv.FormatInt(fl.version, 10) + " " + strconv.Itoa(len(fl.fileContent)) + " " + strconv.Itoa(remainingSeconds) + "\r\n" + string(fl.fileContent) + "\r\n"
				} else {
					cmd.result <- "CONTENTS " + strconv.FormatInt(fl.version, 10) + " " + strconv.Itoa(len(fl.fileContent)) + " " + strconv.Itoa(fl.fileLife) + "\r\n" + string(fl.fileContent) + "\r\n"
				}
			} else {
				cmd.result <- "ERR_FILE_NOT_FOUND\r\n"
			}
		case "cas":
			var fileVersion int64 = 1
			if fl, found := data[cmd.filename]; found {
				//				hh, min, ss := cmd.fileStruct.fileCreationTime.Clock()
				//				fmt.Println("CMD = %v : %v : %v", hh, min, ss)
				//
				//				hh, min, ss = fl.fileCreationTime.Clock()
				//				fmt.Println("old file  = %v : %v : %v", hh, min, ss)

				if cmd.fileStruct.expirySpecified { //file has already expired
					duration := time.Since(fl.fileCreationTime)
					//					fmt.Printf("seconds %v \r\n", duration.Seconds())
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
		case "delete":
			if fl, found := data[cmd.filename]; found {
				if fl.expirySpecified {
					duration := time.Since(fl.fileCreationTime)
					//					fmt.Printf("seconds %v \r\n", duration.Seconds())
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
		con, err := ls.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handleClients(commands, con)
	}
}

func main() {
	serverMain()
}
