package gserver

import (
	"log"
	"fmt"
	"strings"
	"strconv"
	"sync"
	"regexp"
)

type Dictionary map[string]interface{}

type Lobby struct {
	State map[string]interface{}
	PrivateState map[*Client]Dictionary 
	id int
	leader *Client // Should always be non-null
	party []*Client
	lock sync.RWMutex
}

// Assigns "playerX" to *Client in lobby state, starting from 1 (the leader).
// Assigns X-1 to Client's private state, matching game's turn (0-indexed).
// sets numPlayers in state
func (l *Lobby) AssignPlayers() {
	l.UpdateState("numPlayers", 1 + len(l.party))
	l.UpdateState("player1", l.leader)
	l.UpdatePrivateState(l.leader, "player", 0)
	for i, player := range l.party {
		l.UpdateState(fmt.Sprintf("player%d", i+2), player)
		l.UpdatePrivateState(player, "player", i+1)
	}
}

func (l *Lobby) GetPlayer(number int) (*Client) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	player, ok := l.State[fmt.Sprintf("player%d", number)]
	if !ok {
		return nil
	}
	client, ok := player.(*Client)
	if !ok {
		return nil
	}
	return client
}

func (l *Lobby) DeletePrivateState(client *Client, key string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	clientState, ok := l.PrivateState[client]
	if !ok {
		return
	}
	delete(clientState, key)
}

func (l *Lobby) GetPrivateState(client *Client, key string) (interface{}, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	clientState, ok := l.PrivateState[client]
	if !ok {
		return nil, false
	}
	v, ok := clientState[key]
	return v, ok
}

// Locks
func (l *Lobby) UpdatePrivateState(client *Client, key string, value interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.PrivateState[client]; !ok {
		l.PrivateState[client] = make(Dictionary)
	}
	l.PrivateState[client][key] = value
}

func (l *Lobby) GetState(key string) (interface{}, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	v, ok := l.State[key]
	return v, ok
}

func (l *Lobby) UpdateState(key string, value interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.State[key] = value
}

// Rlocks server
func (server *Server) lobbyList() string {
	server.lock.RLock()
	defer server.lock.RUnlock()

	lobbyList := ""
	for e := server.lobbies.Front(); e != nil; e = e.Next() {
		lobby := e.Value.(*Lobby)

		lobby.lock.RLock()	
		lobbyList += fmt.Sprintf("Lobby %d\n", lobby.id)
		lobbyList += fmt.Sprintf("Members %d\n", len(lobby.party) + 1)
		lobbyList += fmt.Sprintf("(Leader) %s\n", lobby.leader.Name)
		for _, member := range lobby.party {
			lobbyList += fmt.Sprintf("(Member) %s\n", member.Name)
		}
		lobbyList += "\n"
		lobby.lock.RUnlock()	
	}
	return strings.TrimSpace(lobbyList)
}

func (server *Server) LogClients() {
	server.lock.RLock()
	defer server.lock.RUnlock()

	log.Printf(server.lobbyList())
	
	idle := "Clients not in lobbies:\n"
	for client := range server.idleClients {
		idle += "- Client: " + client.Name + "\n"
	}
	log.Println(strings.TrimSpace(idle))
}

type LobbyErr struct { msg string }
func (l LobbyErr) Error() string { return l.msg }

func (lobby *Lobby) IsLeader(client *Client) bool {
	lobby.lock.RLock()
	defer lobby.lock.RUnlock()

	return client == lobby.leader
}

// DOESNT lock lobby
func (lobby *Lobby) removeMember(i int) {
	lobby.party[i] = lobby.party[len(lobby.party)-1]
	lobby.party = lobby.party[:len(lobby.party)-1]
}

// DOESNT lock server
func (server *Server) removeLobby(id int) error {
	for e := server.lobbies.Front(); e != nil; e = e.Next() {
		lobby := e.Value.(*Lobby)
		if lobby.id == id {
			if lobby.numMembers() > 0 {
				// note client lock ?
				lobby.leader.Lobby = nil	
				for _, member := range lobby.party {
					member.Lobby = nil	
				}
			}
			server.lobbies.Remove(e)
			return nil
		}
	}
	return LobbyErr{"Tried to remove nonexistent lobby"}
}

func (server *Server) findLobby(id int) *Lobby {
	server.lock.RLock()
	defer server.lock.RUnlock()
	
	for e := server.lobbies.Front(); e != nil; e = e.Next() {
		lobby := e.Value.(*Lobby)
		if lobby.id == id {
			return lobby
		}
	}
	return nil
}

func (l *Lobby) updateMsg() Msg {
	return Msg{
		Msg: "update lobby",
		Body: l.getBody(),
	}
}

type Member struct {
	Name string `json:"name"`
	IsLeader bool `json:"isLeader"`
}

func (lobby *Lobby) getBody() map[string]interface{} {
	type Member struct {
		Name string `json:"name"`
		IsLeader bool `json:"isLeader"`
	}

	members := []Member{{lobby.leader.Name, true}}
	for _, p := range lobby.party {
		members = append(members, Member{p.Name, false})
	}

	lobbyObj := map[string]interface{}{
		"id": lobby.id,
		"members": members,
	}
	return lobbyObj
}

func (server *Server) getLobbies() map[string]interface{} {
	body := map[string]interface{}{}
	for l := server.lobbies.Front(); l != nil; l = l.Next() {
		lobby := l.Value.(*Lobby)	


		members := []Member{{lobby.leader.Name, true}}
		for _, p := range lobby.party {
			members = append(members, Member{p.Name, false})
		}

		lobbyObj := map[string]interface{}{
			"id": lobby.id,
			"members": members,
		}
		body[strconv.Itoa(lobby.id)] = lobbyObj
	}
	return body
}

func validateNickname(n string) error {
	match, _ := regexp.MatchString("^[0-9A-Za-z]{1,16}$", n)
	if !match {
		return LobbyErr{"Nicknames must only contain numbers and letters"}
	}
	return nil
}

// Client commands executed by the server.
// change name, create/join lobbies
func ReceiveLobbyCommands(client *Client, server *Server, msg Msg) (executed bool) {
	executed = true
	response := MakeResponse(msg)
	
	lobby := client.Lobby
	switch msg.Msg {
	case "change name":
		name, ok := msg.Body["name"]
		if !ok {
			response.Error("No 'name' in request body")
			break
		}
		client.Name = name.(string)
	case "get name":
		response.Body["name"] = client.Name	
	case "create lobby":
		if client.Lobby != nil {
			response.Error("Already in a lobby")
			break
		}

		var name string
		err := CheckBody(msg, "nickname", &name)
		if err != nil {
			response.Error(err.Error())
			break
		}
		err = validateNickname(name)
		if err != nil {
			response.Error(err.Error())
			break
		}
		client.Name = name

		server.lock.Lock()
		newId := 1
		if server.lobbies.Len() > 0 {
			newId = server.lobbies.Back().Value.(*Lobby).id + 1
		}
		server.lobbies.PushBack(&Lobby{
			id: newId, 
			leader: client,
			State: make(map[string]interface{}),
			PrivateState: map[*Client]Dictionary{},
		})
		client.Lobby = server.lobbies.Back().Value.(*Lobby)
		delete(server.idleClients, client)

		response.Body["lobbyId"] = client.Lobby.id
		response.Body["leaderNickname"] = client.Name
		server.lock.Unlock()

		server.BroadcastIdle(client.Lobby.updateMsg())
	case "leave lobby":
		id, numMembers := -1, -1
		if lobby != nil {
			id = lobby.id
			numMembers = lobby.numMembers() - 1
		}

		err := server.leaveLobby(client)
		if err != nil {
			response.Error(err.Error())
		}

		log.Printf("Client left lobby %d, now has %d members",
			lobby.id, numMembers)
		if numMembers == 0 {
			server.BroadcastIdle(Msg{
				Msg: "update lobby",
				Body: map[string]interface{}{
					"id": id,
					"deleted": true,
				},
			})
		} else {
			lobby.Broadcast(lobby.updateMsg())
			server.BroadcastIdle(lobby.updateMsg())
		}
	case "join lobby":
		f64, ok := msg.Body["lobbyId"].(float64)
		if !ok {
			response.Error("Expected 'lobbyId' in body to be type f64")
			break
		}

		var name string
		err := CheckBody(msg, "nickname", &name)
		if err != nil {
			response.Error(err.Error())
			break
		}
		err = validateNickname(name)
		if err != nil {
			response.Error(err.Error())
			break
		}

		if client.Lobby != nil {
			response.Error("Must leave lobby before joining a new one")
			break
		}

		lobby := server.findLobby(int(f64))
		if lobby == nil {
			response.Error(fmt.Sprintf("Client tried to join nonexistent lobby %d", int(f64)))
			break
		}

		if !lobby.isUniqueNickname(name) {
			response.Error(fmt.Sprintf("Somebody already has that nickname. Please choose another one"))
			break
		}
		client.Name = name

		server.lock.Lock()
		lobby.party = append(lobby.party, client)
		client.Lobby = lobby
		delete(server.idleClients, client)
		log.Printf("Client joined lobby %d", client.Lobby.id)
		server.lock.Unlock()

		response.Body["lobbyId"] = client.Lobby.id

		lobby.Broadcast(lobby.updateMsg())
		server.BroadcastIdle(lobby.updateMsg())
	case "get lobby":
		response.Body["lobbies"] = server.getLobbies()
	default:
		return false
	}

	log.Println("Received and responded to lobby command")
	SendToClient(response, client)
	return
}

func (lobby *Lobby) Broadcast(msg Msg) {
	lobby.lock.RLock()
	defer lobby.lock.RUnlock()

	if lobby.leader != nil {
		SendToClient(msg, lobby.leader)
	}
	for _, client := range lobby.party {
		SendToClient(msg, client)
	}
}

// Rlocks server
func (server *Server) BroadcastIdle(msg Msg) {
	server.lock.RLock()
	defer server.lock.RUnlock()

	for client := range server.idleClients {
		if client == nil {
			delete(server.idleClients, client)
			log.Fatalf("Unexpected: Server idle client is nil")
		}
		err := SendToClient(msg, client)
		if err != nil {
			delete(server.idleClients, client)
			log.Printf("Couldn't send msg '%s' to client", msg)
		}
	}
}

// Locks lobby and maybe server.
// Removes lobby if empty after leaving
// and adds client to idleclients
func (server *Server) leaveLobby(client *Client) error {
	lobby := client.Lobby
	if lobby == nil {
		return LobbyErr{"Not in a lobby"}
	}

	lobby.lock.Lock()
	if lobby.leader == client {
		client.Lobby = nil
		if len(lobby.party) > 0 {
			log.Printf("Promoting party member")
			lobby.leader = lobby.party[0]
			lobby.removeMember(0)
		} else {
			lobby.leader = nil
		}
	} else {
		for i, member := range lobby.party {
			if member == client {
				member.Lobby = nil
				lobby.removeMember(i)
				break
			}
		}
	}
	lobby.lock.Unlock()

	server.lock.Lock()
	if lobby.numMembers() == 0 {
		server.removeLobby(lobby.id)
	}
	server.idleClients[client] = true
	server.lock.Unlock()

	return nil
}

// Rlocks lobby
func (l *Lobby) numMembers() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	sum := len(l.party)
	if l.leader != nil {
		return sum + 1
	}
	return sum
}

// Rlocks lobby
func (l *Lobby) isUniqueNickname(n string) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if n == l.leader.Name {
		return false
	}
	for _, p := range l.party {
		if n == p.Name {
			return false
		}
	}
	return true
}
