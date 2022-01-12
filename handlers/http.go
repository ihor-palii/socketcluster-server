package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/subscribers"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/utils"
	"github.com/pbnjay/memory"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"time"
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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://help.communityconnectlabs.com/support/home", 301)
}

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

func ReceiveMessageHandler(w http.ResponseWriter, r *http.Request) {
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
}

type pingPayload struct {
	PID      int64
	HostName string
	UpTime   int64
	FreeMem  int64
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Println(err)
		hostname = ""
		return
	}

	data := &pingPayload{
		PID:      int64(os.Getpid()),
		HostName: hostname,
		UpTime:   int64(time.Since(startTime).Seconds()),
		FreeMem:  int64(memory.FreeMemory()),
	}
	tmpl, _ := template.ParseFiles("templates/ping.html")
	err = tmpl.Execute(w, data)
	if err != nil {
		logrus.Errorln(err)
	}
}

func WebSocketConnectionHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorln("Failed to Upgrade connection to WebSocket:", err)
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
							logrus.Errorln("Failed to send ping message:", err)
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
				logrus.Errorln("Failed to read message:", err)
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
				msg := &WSMessage{}
				jsonError := json.Unmarshal(rawData, msg)
				if jsonError == nil {
					if msg.Event == "#handshake" {
						err = HandleHandshakeMsg(client, msg, pingPongChannel)
						if err != nil {
							logrus.Errorln("Failed to send handshake response message:", err)
							return
						}
					} else if msg.Event == "registerUser" {
						err = HandleRegisterUser(client, msg)
						if err != nil {
							logrus.Errorln("Failed to process register user:", err)
							return
						}
					} else if msg.Event == "getHistory" {
						err = HandleGetHistory(client, msg)
						if err != nil {
							logrus.Errorln("Failed to get history:", err)
							return
						}
					} else if msg.Event == "sendMessageToChannel" {
						err = HandleSendMessageToChannel(client, msg)
						if err != nil {
							logrus.Errorln("Failed to send message:", err)
							continue
						}
					} else if msg.Event == "#subscribe" {
						err = HandleSubscribe(client, msg, room)
						if err != nil {
							logrus.Errorln("Failed to subscribe:", err)
							return
						}
					}
				}
			}
		}
	}
}
