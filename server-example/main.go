package main 

import (
	"gserver"
	"net"
	"log"
	"bufio"
	"os"
	"strings"
)

const (
	PORT = ":33445"
)

func main() {
	listener, err := net.Listen("tcp", PORT) 
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	log.Printf("Starting server at port %s", PORT)

	server := gserver.InitServer()

	go func() {
		for {
			in := bufio.NewReader(os.Stdin)

			msg, _ := in.ReadString('\n')
			msg = strings.TrimSpace(msg)

			log.Println(msg)
			switch msg {
			case "ls":
				server.LogClients()
			}
		}
	}()

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Printf("Error: %s", err)
			break
		}

		// CUSTOM command handler
		executeClientCommand := func(client *gserver.Client, server *gserver.Server, msg gserver.Msg) error {
			// PREDEFINED command handlers
			if gserver.ReceiveLobbyCommands(client, server, msg) || 
			   gserver.ReceiveChatCommands(client, server, msg) {
				return nil
			}

			response := gserver.MakeResponse(msg)

			switch msg.Msg {
			case "start game":
				if client.Lobby == nil {
					response.Error("Need to be in lobby to start game")
					break
				}
				started, ok := client.Lobby.GetState("started")
				if ok && started.(bool) {
					response.Error("Game already started")
					break
				}
				// check if there are 2 players

				client.Lobby.AssignPlayers()
				
				startBroadcast := msg
				startBroadcast.StatusCode = 0
				client.Lobby.UpdateState("started", true)
				client.Lobby.Broadcast(startBroadcast)
			}

			inGameCommands := map[string]bool{"rock": false, "paper": false, "scissors": false}
			switch _, ok := inGameCommands[msg.Msg]; ok {
			case true:
				started, ok := client.Lobby.GetState("started")
				if !ok || !started.(bool) {
					response.Error("Game not started")
					break
				}

				client.Lobby.UpdatePrivateState(client, "move", msg.Msg)

				player1 := client.Lobby.GetPlayer(1)
				player1move, ok := client.Lobby.GetPrivateState(player1, "move")
				if !ok {
					break
				}
				player2 := client.Lobby.GetPlayer(2)
				player2move, ok := client.Lobby.GetPrivateState(player2, "move")
				if !ok {
					break
				}

				p1move := player1move.(string)
				p2move := player2move.(string)
				winner := "player1 wins"
				if p1move == p2move {
					winner = "it's a draw"
				} else if p1move == "scissors" && p2move == "rock" || 
						p1move == "paper" && p2move == "scissors" || 
						p1move == "rock" && p2move == "paper" {
					winner = "player2 wins"
				}

				client.Lobby.DeletePrivateState(player1, "move")
				client.Lobby.DeletePrivateState(player2, "move")

				client.Lobby.Broadcast(gserver.Msg{Msg: winner})
			}

			gserver.SendToClient(response, client)
			return nil
		}

		go gserver.HandleNetClient(client, server, executeClientCommand)
	}
}
