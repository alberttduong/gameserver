package gameserver

import (
	"net"
	"log"
	"bufio"
	"io"
	"strings"
	"github.com/gorilla/websocket"
)

type conn interface {
	close()
	read() ([]byte, error) // incoming
	write([]byte) error // outgoing 
	send() error
	init()
}

type tcpCon struct {
	reader *bufio.Reader
	conn net.Conn
}
func (t *tcpCon) init() {
	t.reader = bufio.NewReader(t.conn)
}
func (t tcpCon) close() {
	t.conn.Close()
}
func (t *tcpCon) read() ([]byte, error) {
	return t.reader.ReadBytes('\n')
}
func (t tcpCon) write(msg []byte) error {
	_, err := t.conn.Write(msg)
	return err
}
func (t tcpCon) send() error {
	_, err := t.conn.Write(newline)
	return err
}


type wsCon struct {
	conn *websocket.Conn
	nextWriter io.WriteCloser
}
func (c *wsCon) init() {
	c.nextWriter, _ = c.conn.NextWriter(websocket.TextMessage)
}
func (c *wsCon) close() {
	c.conn.Close()
}
func (c *wsCon) read() ([]byte, error) {
	_, msg , err := c.conn.ReadMessage()
	str := strings.ReplaceAll(
		strings.Trim(string(msg), "\""), 
		"\\\"",
		"\"")
	log.Printf("%s len:%d", str, len(str))
	return []byte(str), err
}

func (c *wsCon) write(msg []byte) error {
	_, err := c.nextWriter.Write(msg)
	return err
}

func (c *wsCon) send() (err error) {
	c.nextWriter, err = c.conn.NextWriter(websocket.TextMessage)
	return err
}
