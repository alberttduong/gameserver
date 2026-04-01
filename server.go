package gameserver

import (
	"net"
	"log"
	"io"
	"encoding/json"
	"fmt"
	"sync"
	"container/list"
	"github.com/gorilla/websocket"
)

const (
	PORT = ":7280"
	maxLobbies = 10
)

var (
	newline = []byte("\n")
)

type Msg struct {
	StatusCode int
	Msg string
	Body map[string]interface{}
}

// Default status code 200
func MakeResponse(msg Msg) Msg {
	return Msg{
		Msg: msg.Msg,
		StatusCode: 200,
		Body: make(map[string]interface{}),
	}
}

func CheckNumber(msg Msg, key string, dest *int) error {
	v, ok := msg.Body[key]	
	if !ok {
		return LobbyErr{fmt.Sprintf("Key %s not in body", key)} 
	}
	value, ok := v.(int)
	if !ok {
		v, ok := v.(float64)
		if !ok {
			return LobbyErr{fmt.Sprintf("Key %s not a number", key)}
		}
		value = int(v)
	}
	*dest = value
	return nil
}

// not for numbers
func CheckBody[T any](msg Msg, key string, dest *T) error {
	v, ok := msg.Body[key]	
	if !ok {
		return LobbyErr{fmt.Sprintf("Key %s not in body", key)} 
	}
	value, ok := v.(T)
	if !ok {
		return LobbyErr{fmt.Sprintf("Nonnumber Key %s incorrect type", key)}
	}
	*dest = value
	return nil
}

func (m *Msg) Error(e string) {
	m.StatusCode = 400
	m.Body["error"] = e
}

func (m Msg) String() string {
	s := fmt.Sprintf("- StatusCode: %d\n- Msg: %s\n- Body:\n", 
		m.StatusCode, m.Msg)
	for k, v := range m.Body {
		s += fmt.Sprintf("\t- %s: %v\n", k, v)
	}
	return s
}

type Client struct {
	con conn
	Conn net.Conn
	Name string
	Lobby *Lobby
}

func SendToTCP(msg Msg, con net.Conn) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = con.Write(msgBytes)
	if err != nil {
		return err
	}
	con.Write([]byte("\n"))
	return nil
}

func SendToClient(msg Msg, client *Client) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err = client.con.write(msgBytes); err != nil {
		return err
	}
	
	if err = client.con.send(); err != nil {
		return err
	}
	return nil
}

type Server struct {
	lobbies *list.List
	clients []Client
	idleClients map[*Client]bool
	lock sync.RWMutex
}

func InitServer() *Server {
	return &Server{
		lobbies: list.New(), //make(map[int]*Lobby),
		idleClients: make(map[*Client]bool),
	}
}

func (c *Client) disconnect(server *Server) {
	c.con.close()
	if c.Conn != nil {
		c.Conn.Close()
	}	
	server.leaveLobby(c)
}

func HandleNetClient(conn net.Conn, server *Server, execute func(*Client, *Server, Msg) error) {
	c := Client{ con: &tcpCon{conn: conn} }
	HandleClient(&c, server, execute)
}

func HandleWSClient(conn *websocket.Conn, server *Server, execute func(*Client, *Server, Msg) error) {
	c := Client{ con: &wsCon{conn: conn} }
	HandleClient(&c, server, execute)
}

func HandleClient(client *Client, server *Server, execute func(*Client, *Server, Msg) error) {
	defer client.disconnect(server)

	log.Printf("Client joined") // todo ip?

	client.con.init()

	msg := Msg{
		Msg: "Welcome client",
		Body: map[string]interface{}{ },
	}

	SendToClient(msg, client)

	server.lock.Lock()
	server.idleClients[client] = true
	server.lock.Unlock()

	for {
		buf, err := client.con.read()
		if err == io.EOF {
			log.Printf("Client disconnected")
			break
		}
		if err != nil {
			log.Printf("Error in reading client msg: %s", err)
			break
		}
		var msg Msg
		err = json.Unmarshal(buf, &msg)
		if err != nil {
			log.Printf("Error in unmarshaling msg: %s", err)
			break
		}
		log.Printf("Client wrote \n%s", msg)
		execute(client, server, msg)
	}
}
