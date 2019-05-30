package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/appengine"
)

type ConnectionInfo struct {
	DeviceID int
	ThreadID int
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
			// if this client sent the message, skip them
			if connected_clients[client].DeviceID == msg.ConnectionInfo.DeviceID {
				continue
			}
			// check if this client is in the corresponding thread, if not continue
			if connected_clients[client].ThreadID != msg.ConnectionInfo.ThreadID {
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
	// Register our new client as the highest id and corresponding thread
	threadID, _ := strconv.Atoi(c.Query("threadID"))
	connected_clients[ws] = ConnectionInfo{DeviceID: len(connected_clients), ThreadID: threadID}
	for {
		var msg Message
		// read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("error: %v", err)
			delete(connected_clients, ws)
			break
		}
		msg.Posted = time.Now()
		msg.ThreadID = threadID
		ctx := appengine.NewContext(c.Request)
		key := datastore.IncompleteKey("Message", nil)
		if _, err := client.Put(ctx, key, &msg); err != nil {
			log.Printf("error: %v", err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		// set the corresponding connection Info and broadcast new message
		msg.ConnectionInfo = connected_clients[ws]
		broadcast <- msg
	}
}
