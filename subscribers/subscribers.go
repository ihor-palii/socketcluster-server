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
	UserUrn     string
	UserToken   string
	Connection  *websocket.Conn
	Active      bool
}

func NewClient(r *http.Request, ws *websocket.Conn) *Client {
	return &Client{
		Id:          sid.IdBase64(),
		ChannelUUID: r.URL.Query().Get("channelUUID"),
		HostApi:     r.URL.Query().Get("hostApi"),
		UserToken:   r.URL.Query().Get("userToken"),
		Connection:  ws,
		Active:      true,
		UserUrn:     "",
	}
}

func (r *Room) AddClient(client *Client) *Room {
	if client.UserUrn != "" {
		r.Clients[client.UserUrn] = client
	}
	return r
}

func (r *Room) RemoveClient(channel string) *Room {
	_, ok := r.Clients[channel]
	if ok {
		delete(r.Clients, channel)
	}
	return r
}

func (r *Room) SendChannelMessage(channel string, v interface{}) error {
	client, ok := r.Clients[channel]
	if ok {
		err := client.Connection.WriteJSON(v)
		if err != nil {
			return err
		}
	}
	return nil
}
