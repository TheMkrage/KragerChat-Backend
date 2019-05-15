package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ConnectionInfo struct {
	DeviceID int
}

var connected_clients = make(map[*websocket.Conn]ConnectionInfo)
var broadcast = make(chan Message)

// Configure the upgrader
var upgrader = websocket.Upgrader{}

// Chat methods
func handleMessages() {
	for {
		msg := <-broadcast
		// Send msg to all connected
		for client := range connected_clients {
			if connected_clients[client] == msg.ConnectionInfo {
				continue
			}
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(connected_clients, client)
			}
		}
	}
}

// GET /ws
func handleConnections(c *gin.Context) {
	// Upgrade initial GET request to a socket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	// Register our new client as the highest id
	connected_clients[ws] = ConnectionInfo{DeviceID: len(connected_clients)}
	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("error: %v", err)
			delete(connected_clients, ws)
			break
		}
		// set the corresponding connection Info and broadcast new message
		msg.ConnectionInfo = connected_clients[ws]
		broadcast <- msg
	}
}