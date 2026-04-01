package gserver

import "log"

func ReceiveChatCommands(client *Client, server *Server, msg Msg) (executed bool) {
	executed = true
	response := MakeResponse(msg)

	switch msg.Msg {
	case "chat send":
		if client.Lobby == nil {
			response.Error("Client can't use chat, not in lobby")
			break
		}

		msg, ok := msg.Body["msg"]
		if !ok {
			response.Error("No 'msg' in request body")
			break
		}
		msgToSend := Msg{
			Msg: "chat newmsg",
			Body: map[string]interface{}{"msg": msg},
		}
		client.Lobby.Broadcast(msgToSend)
	default:
		return false
	}

	log.Println("Received and responded to chat command")
	SendToClient(response, client)
	return executed
}
