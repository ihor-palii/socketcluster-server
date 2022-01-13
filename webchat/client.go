package webchat

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var (
	pingPeriod = 10 * time.Second
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 8 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
	}
)

type Client struct {
	Id          string
	ChannelUUID string
	HostApi     string
	UserUrn     string
	UserToken   string
	Connection  *websocket.Conn

	hub  *Hub
	send chan interface{}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.Connection.Close()
	}()

	for {
		_, rawData, err := c.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorln("Failed to read message:", err)
			}
			break
		}

		// skip message if it pong message
		if string(rawData) == "#2" {
			continue
		}

		// handle new WebSocket message
		msg := &WSMessage{}
		jsonError := json.Unmarshal(rawData, msg)
		if jsonError == nil {
			err, errMsg := HandleWSMessage(c, msg)
			if err != nil {
				logrus.Errorln(errMsg, err)
			}
		}
	}
}

func (c *Client) writePump() {
	ticket := time.NewTicker(pingPeriod)
	defer func() {
		ticket.Stop()
		_ = c.Connection.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}

			if msg != nil {
				err := c.Connection.WriteJSON(msg)
				if err != nil {
					logrus.Errorln("Failed to send json message:", err)
					return
				}
			}
		case <-ticket.C:
			// ping message sending
			err := c.Connection.WriteMessage(websocket.TextMessage, []byte("#1"))
			if err != nil {
				logrus.Errorln("Failed to send ping message:", err)
				return
			}
		}
	}
}
