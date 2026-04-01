package gameserver 

import (
	"net"
	"log"
	"bufio"
	"io"
	"encoding/json"
)

// For frontends in go
func StartClient(server net.Conn) {
	defer server.Close()
	log.Printf("Starting client")	

	reader := bufio.NewReader(server)
	for {
		buf, err := reader.ReadBytes('\n')
		if err == io.EOF {
			log.Printf("Server unexpectedly disconnected")
			break
		}
		if err != nil {
			log.Printf("Error in reading server msg: %s", err)
			break
		}
		
		var msg Msg
		err = json.Unmarshal(buf, &msg)
		if err != nil {
			log.Printf("Error in unmarshaling msg: %s", err)
			break
		}
		log.Printf("Server wrote \n%s", msg)
	}
}
