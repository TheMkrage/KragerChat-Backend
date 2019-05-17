package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

var (
	repeat int
	db     *sql.DB
)

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
	ThreadID       int
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

// POST /threads
func createThread(c *gin.Context) {
	ctx := appengine.NewContext(c.Request)

	id, err := datastore.NewQuery("Thread").Count(ctx)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := datastore.NewIncompleteKey(ctx, "Thread", nil)
	name := c.Query("name")
	thread := Thread{ID: id, Name: name}
	if _, err := datastore.Put(ctx, key, thread); err != nil {
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
	key := datastore.NewIncompleteKey(ctx, "Subscription", nil)

	if _, err := datastore.Put(ctx, key, subscription); err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, subscription)
}

func main() {
	os.Setenv("PORT", "8000")
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	// checks origin before allowing upgrade to connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	router := gin.New()
	router.Use(gin.Logger())

	router.POST("/threads/:id/join", joinThread)
	router.POST("/threads", createThread)
	router.GET("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	router.Run(":" + port)
}
