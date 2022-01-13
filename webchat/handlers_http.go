package webchat

import (
	"fmt"
	"github.com/chilts/sid"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/utils"
	"github.com/pbnjay/memory"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
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

func MessageReceived(hub *Hub, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	payload := &newMsgPayload{}
	err := utils.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return
	}

	hub.receive <- &HubMessage{
		client: payload.To,
		msg: map[string]interface{}{
			"event": "#publish",
			"data": map[string]interface{}{
				"channel": payload.To,
				"data":    payload,
			},
		},
	}
	hub.receive <- &HubMessage{
		client: payload.To,
		msg: map[string]interface{}{
			"event": "receivedMessageFromChannel",
			"data":  payload,
		},
	}
}

type pingPayload struct {
	PID      int64
	HostName string
	UpTime   int64
	FreeMem  int64
}

func Ping(startTime time.Time, w http.ResponseWriter, r *http.Request) {
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

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorln(err)
	}

	client := &Client{
		Id:          sid.IdBase64(),
		ChannelUUID: r.URL.Query().Get("channelUUID"),
		HostApi:     r.URL.Query().Get("hostApi"),
		UserToken:   r.URL.Query().Get("userToken"),
		Connection:  conn,
		hub:         hub,
		send:        make(chan interface{}),
	}

	go client.writePump()
	go client.readPump()
}
