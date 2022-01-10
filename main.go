package main

import (
	"encoding/json"
	"fmt"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/handlers"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pbnjay/memory"

	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/subscribers"
)

func receiveMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// todo: send message to channel to specific room.
	} else {
		// http.Redirect(w, r, "https://help.communityconnectlabs.com/support/home", 301)
		tmpl, _ := template.ParseFiles("templates/index.html")
		tmpl.Execute(w, nil)
	}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 8 * time.Second,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

var room = &subscribers.Room{
	Clients: map[string]*subscribers.Client{},
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to Upgrade connection to WebSocket:", err)
		return
	}

	client := subscribers.NewClient(r, ws)
	room.AddClient(client)

	// ping pong goroutine to keep connection with socketcluster
	pingPongChannel := make(chan string)
	defer close(pingPongChannel)
	go func() {
		for {
			select {
			case pongMessage := <-pingPongChannel:
				if pongMessage == "#2" {
					time.Sleep(10 * time.Second)
					err = client.Connection.WriteMessage(websocket.TextMessage, []byte("#1"))
					if err != nil {
						log.Println("Something went wrong", err)
						return
					}
				}
			}
		}
	}()

	// main loop for handling WebSocket messages
	for {
		msgType, rawData, err := client.Connection.ReadMessage()
		if err != nil {
			log.Println("Failed to read message:", err)
			return
		}

		if msgType == websocket.TextMessage {
			// handle pong message
			if string(rawData) == "#2" {
				// put pong message to channel to start sending of the new ping message
				pingPongChannel <- "#2"
				continue
			}

			// handle json messages
			msg := &handlers.WSMessage{}
			jsonError := json.Unmarshal(rawData, msg)
			if jsonError == nil {
				if msg.Event == "#handshake" {
					err = handlers.HandleHandshakeMsg(client, msg, pingPongChannel)
					if err != nil {
						log.Println("Failed to send handshake response message:", err)
						return
					}
				} else if msg.Event == "registerUser" {
					err = handlers.HandleRegisterUser(client, msg)
					if err != nil {
						log.Println("Failed to process register user:", err)
						return
					}
				} else if msg.Event == "getHistory" {
					err = handlers.HandleGetHistory(client, msg)
					if err != nil {
						log.Println("Failed to get history:", err)
						return
					}
				} else if msg.Event == "#subscribe" {
					log.Println(string(msg.Data))
				}
			}
		}
	}
}

var startTime = time.Time{}

type PingData struct {
	PID      int64
	HostName string
	UpTime   string
	FreeMem  int64
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		hostname = ""
	}

	data := &PingData{
		PID:      int64(os.Getpid()),
		HostName: hostname,
		UpTime:   fmt.Sprint(time.Since(startTime)),
		FreeMem:  int64(memory.FreeMemory()),
	}
	tmpl, _ := template.ParseFiles("templates/ping.html")
	tmpl.Execute(w, data)
}

func setupRoutes() {
	http.HandleFunc("/", receiveMessage)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/socketcluster/", webSocketHandler)
}

func main() {
	log.Println("CCL Websocket Server Running...")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":9090", nil))
}