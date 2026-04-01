package main

import (
	"net"
	"gserver"
	"fmt"
	"bufio"
	"os"
	"log"
	"strings"
	"strconv"
)

var (
	SERVER_IP = net.IPv4(127, 0, 0, 1)
	SERVER_PORT = ":33445"
)


func main() {
	server, serverErr := net.Dial("tcp", fmt.Sprintf("%v%s", SERVER_IP, SERVER_PORT))
	if serverErr == nil {
		go gserver.StartClient(server)
	} else {
		log.Println("Failed to connect")
	}

	for {
		in := bufio.NewReader(os.Stdin)

		msg, _ := in.ReadString('\n')
		msg = strings.TrimSpace(msg)

		body := map[string]interface{}{}

		if serverErr != nil {
			if msg == "connect" {
				log.Println("reconnecting...")
				server, serverErr = net.Dial("tcp", fmt.Sprintf("%v%s", SERVER_IP, SERVER_PORT))
				if serverErr == nil {
					log.Println("connected")
					go gserver.StartClient(server)
				} else {
					log.Println("Failed to connect")
				}
			}
			continue
		}

		if strings.HasPrefix(msg, "join lobby") {
			number, _ := strconv.Atoi(strings.Split(msg, " ")[2])
			msg = "join lobby"
			body["lobbyId"] = number
		} else if strings.HasPrefix(msg, "change name") {
			params := strings.Split(msg, " ")
			if len(params) > 2 {
				body["name"] = params[2]
			}
			msg = "change name"
		} else if strings.HasPrefix(msg, "chat send") {
			params := strings.Split(msg, " ")
			if len(params) > 2 {
				body["msg"] = params[2]
			}
			msg = "chat send"
		}
		
		gserver.SendToTCP(gserver.Msg{Msg: msg, Body: body}, server)
	}
}
