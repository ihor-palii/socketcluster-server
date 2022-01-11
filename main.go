package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pbnjay/memory"

	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/handlers"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/subscribers"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/utils"
)

var (
	startTime  = time.Now()
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 8 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
	}
	room = &subscribers.Room{
		Clients: map[string]*subscribers.Client{},
	}
)

type newMsgPayload struct {
	ID          string      `json:"id"`
	Text        string      `json:"text"`
	To          string      `json:"to"`
	ToNoPlus    string      `json:"to_no_plus"`
	From        string      `json:"from"`
	FromNoPlus  string      `json:"from_no_plus"`
	Channel     string      `json:"channel"`
	Metadata    interface{} `json:"metadata"`
	Attachments interface{} `json:"attachments"`
}

func receiveMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}

		payload := &newMsgPayload{}
		err := utils.DecodeAndValidateJSON(payload, r)
		if err != nil {
			return
		}

		publishEventPayload := map[string]interface{}{
			"event": "#publish",
			"data": map[string]interface{}{
				"channel": payload.To,
				"data":    payload,
			},
		}
		err = room.SendChannelMessage(payload.To, publishEventPayload)
		if err != nil {
			return
		}

		receivedMsgEventPayload := map[string]interface{}{
			"event": "receivedMessageFromChannel",
			"data":  payload,
		}
		err = room.SendChannelMessage(payload.To, receivedMsgEventPayload)
		if err != nil {
			return
		}
	} else {
		// http.Redirect(w, r, "https://help.communityconnectlabs.com/support/home", 301)
		tmpl, _ := template.ParseFiles("templates/index.html")
		tmpl.Execute(w, nil)
	}
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to Upgrade connection to WebSocket:", err)
		return
	}

	client := subscribers.NewClient(r, ws)
	client.Connection.SetCloseHandler(func(code int, text string) error {
		client.Active = false
		room.RemoveClient(client.UserUrn)
		message := websocket.FormatCloseMessage(code, "")
		return client.Connection.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
	})

	// ping pong goroutine to keep connection with socketcluster
	pingPongChannel := make(chan string)
	defer close(pingPongChannel)
	go func() {
		for {
			select {
			case pongMessage := <-pingPongChannel:
				if pongMessage == "#2" {
					time.Sleep(10 * time.Second)
					if client.Active {
						err = client.Connection.WriteMessage(websocket.TextMessage, []byte("#1"))
						if err != nil {
							log.Println("Failed to send ping message:", err)
							return
						}
					} else {
						return
					}
				}
			}
		}
	}()

	// main loop for handling WebSocket messages
	for {
		if client.Active {
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
					} else if msg.Event == "sendMessageToChannel" {
						err = handlers.HandleSendMessageToChannel(client, msg)
						if err != nil {
							log.Println("Failed to send message:", err)
							continue
						}
					} else if msg.Event == "#subscribe" {
						err = handlers.HandleSubscribe(client, msg, room)
						if err != nil {
							log.Println("Failed to subscribe:", err)
							return
						}
					}
				}
			}
		}
	}
}

type PingData struct {
	PID      int64
	HostName string
	UpTime   int64
	FreeMem  int64
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err)
		hostname = ""
		return
	}

	data := &PingData{
		PID:      int64(os.Getpid()),
		HostName: hostname,
		UpTime:   int64(time.Since(startTime).Seconds()),
		FreeMem:  int64(memory.FreeMemory()),
	}
	tmpl, _ := template.ParseFiles("templates/ping.html")
	tmpl.Execute(w, data)
}

func main() {
	log.Println("CCL Websocket Server Running...")

	http.HandleFunc("/", receiveMessage)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/socketcluster/", webSocketHandler)
	log.Fatal(http.ListenAndServe(":9090", nil))
}
