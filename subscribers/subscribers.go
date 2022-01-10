package subscribers

import (
	"github.com/chilts/sid"
	"github.com/gorilla/websocket"
	"net/http"
)

type Room struct {
	Clients map[string]*Client
}

type Client struct {
	Id          string
	ChannelUUID string
	HostApi     string
	UserToken   string
	Connection  *websocket.Conn
}

func NewClient(r *http.Request, ws *websocket.Conn) *Client {
	return &Client{
		Id:          sid.IdBase64(),
		ChannelUUID: r.URL.Query().Get("channelUUID"),
		HostApi:     r.URL.Query().Get("hostApi"),
		UserToken:   r.URL.Query().Get("userToken"),
		Connection:  ws,
	}
}

func (r *Room) AddClient(client *Client) *Room {
	r.Clients[client.Id] = client
	return r
}

func (r *Room) SendMessageToRoom(msgType int, message []byte) {
	for _, client := range r.Clients {
		if err := client.Connection.WriteMessage(msgType, message); err != nil {
			// todo: add logging
			continue
		}
	}
}
