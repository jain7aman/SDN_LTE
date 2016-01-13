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
	version         int64
	fileContent     []byte
	fileExpTime     time.Time
	fileLife        int
	expirySpecified bool
}

var data = make(map[string]file)

func handleClients(con net.Conn) {
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
					continue
				}
				numbytes, err := strconv.Atoi(fs[2])
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				expTime := 0
				expiryFound := false
				if len(fs) == 4 {
					expiryFound = true
					expTime, err = strconv.Atoi(fs[3])
					if err != nil || expTime < 0 {
						io.WriteString(con, "ERR_CMD_ERR\r\n")
						continue
					}
				}
				curTime := time.Now()
				duration, err := time.ParseDuration(strconv.Itoa(expTime) + "s")
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				expiryTime := curTime.Add(duration)

				content := make([]byte, numbytes)
				n, err := reader.Read(content)
				if err != nil || n != numbytes {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}

				_, err = reader.Discard(2) //for \r\n at the end of content
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}

				var version int64 = 1
				if fl, found := data[filename]; found {
					if expiryFound && expiryTime.After(fl.fileExpTime) { //file has already expired
						io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
						delete(data, filename)
						continue
					}
					version = fl.version
					fl.fileContent = content
					fl.fileExpTime = expiryTime
					fl.fileLife = expTime //duration in seconds after which file will expire
					fl.expirySpecified = expiryFound
					data[filename] = fl
				} else {
					fl := file{version: 1, fileContent: content, fileExpTime: expiryTime, fileLife: expTime, expirySpecified: expiryFound}
					data[filename] = fl
				}
				io.WriteString(con, "OK "+strconv.FormatInt(version, 10)+"\r\n")

			case "read":
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				if fl, found := data[filename]; found {
					if fl.expirySpecified {
						curTime := time.Now()
						if curTime.After(fl.fileExpTime) { //file has already expired
							io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
							delete(data, filename)
							continue
						}
						remainDuration := fl.fileExpTime.Sub(curTime)
						remainingSeconds := int(remainDuration.Seconds())
						io.WriteString(con, "CONTENTS "+strconv.FormatInt(fl.version, 10)+" "+strconv.Itoa(len(fl.fileContent))+" "+strconv.Itoa(remainingSeconds)+"\r\n")
					}else {
						io.WriteString(con, "CONTENTS "+strconv.FormatInt(fl.version, 10)+" "+strconv.Itoa(len(fl.fileContent))+" "+strconv.Itoa(fl.fileLife)+"\r\n")
					}
					io.WriteString(con, string(fl.fileContent)+"\r\n")
				} else {
					io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
				}
			case "cas":
				if len(fs) < 4 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				var fileVersion int64
				fileVersion, err := strconv.ParseInt(fs[2], 10, 64) //10 for base decimal
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}

				numbytes, err := strconv.Atoi(fs[3])
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				expTime := 0
				expiryFound := false
				if len(fs) == 5 {
					expiryFound = true
					expTime, err = strconv.Atoi(fs[4])
					if err != nil || expTime < 0 {
						io.WriteString(con, "ERR_CMD_ERR\r\n")
						continue
					}
				}
				curTime := time.Now()
				duration, err := time.ParseDuration(strconv.Itoa(expTime) + "s")
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				expiryTime := curTime.Add(duration)

				content := make([]byte, numbytes)
				n, err := reader.Read(content)
				if err != nil || n != numbytes {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}

				_, err = reader.Discard(2) //for \r\n at the end of content
				if err != nil {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}

				if fl, found := data[filename]; found {
					if fl.version != fileVersion {
						io.WriteString(con, "ERR_VERSION\r\n")
						continue
					}
					if expiryFound && expiryTime.After(fl.fileExpTime) { //file has already expired
						io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
						delete(data, filename)
						continue
					}
					fl.version += 1
					fileVersion = fl.version
					fl.fileContent = content
					fl.fileExpTime = expiryTime
					fl.fileLife = expTime
					fl.expirySpecified = expiryFound
					data[filename] = fl
				} else {
					io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
					continue
				}
				io.WriteString(con, "OK "+strconv.FormatInt(fileVersion, 10)+"\r\n")
			case "delete":
				if len(fs) != 2 {
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				filename := fs[1]
				if len(filename) > 250 { //len(filename) retuens number of bytes in the string
					io.WriteString(con, "ERR_CMD_ERR\r\n")
					continue
				}
				if fl, found := data[filename]; found {
					if fl.expirySpecified {
						curTime := time.Now()
						if curTime.After(fl.fileExpTime) { //file has already expired
							io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
							delete(data, filename)
							continue
						}
					}
					delete(data, filename)
					io.WriteString(con, "OK\r\n")
				} else {
					io.WriteString(con, "ERR_FILE_NOT_FOUND\r\n")
					continue
				}
			default:
				io.WriteString(con, "ERR_CMD_ERR\r\n")
			}
		} else {
			io.WriteString(con, "ERR_CMD_ERR\r\n")
		}
	}

}

func serverMain() {
	ls, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer ls.Close()

	for {
		con, err := ls.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handleClients(con)
	}
}

func main() {
	serverMain()
}
