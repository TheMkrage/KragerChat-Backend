package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var client *datastore.Client

// Define our message object
type Photo struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type Message struct {
	Contents       string `json:"contents"`
	Sender         string `json:"sender"`
	Photo          Photo  `json:"photo"`
	ConnectionInfo ConnectionInfo
	ThreadID       int `json:"threadID"`
	Posted         time.Time
}

type Thread struct {
	ID   int
	Name string
}

type Subscription struct {
	APN_Token string
	ThreadID  int
}

// Get /threads/:id/message
func getMessages(c *gin.Context) {

	ctx := appengine.NewContext(c.Request)
	threadID, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize := 50

	q := datastore.NewQuery("Message").Order("Posted").Filter("ThreadID =", threadID).Offset(page * pageSize).Limit(pageSize)
	var messages []Message
	_, err := client.GetAll(ctx, q, &messages)
	if err != nil {
		log.Printf("error: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, messages)
}

// POST /threads
func createThread(c *gin.Context) {
	ctx := appengine.NewContext(c.Request)

	q := datastore.NewQuery("Thread")
	id, err := client.Count(ctx, q)
	if err != nil {
		log.Printf("error: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := datastore.IncompleteKey("Thread", nil)
	name := c.PostForm("name")
	log.Printf("name: %v", name)
	thread := Thread{ID: id, Name: name}
	if _, err := client.Put(ctx, key, &thread); err != nil {
		log.Printf("error: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, thread)
}

// POST /threads/:id/join
func joinThread(c *gin.Context) {
	ctx := appengine.NewContext(c.Request)
	id, _ := strconv.Atoi(c.Param("id"))
	apn_token := c.PostForm("apn_token")

	subscription := Subscription{ThreadID: id, APN_Token: apn_token}
	key := datastore.IncompleteKey("Subscription", nil)

	if _, err := client.Put(ctx, key, &subscription); err != nil {
		log.Printf("error: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, subscription)
}

func main() {
	//os.Setenv("PORT", "8000")
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	// init google datastore connection
	c, err := datastore.NewClient(context.Background(),
		"kragerchat", option.WithCredentialsFile("./KragerChat-fa2b8563afa1.json"))
	client = c
	if err != nil {
		log.Printf("error: %v", err)
	}
	// checks origin before allowing upgrade to connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	router := gin.New()
	router.Use(gin.Logger())

	router.POST("/threads/:id/join", joinThread)
	router.POST("/threads", createThread)
	router.GET("/threads/:id/messages", getMessages)
	router.GET("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	router.Run(":" + port)
}
