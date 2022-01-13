package webchat

type HubMessage struct {
	client string
	msg    interface{}
}

type Hub struct {
	clients    map[string]*Client // clients available by ID
	register   chan *Client
	unregister chan *Client
	receive    chan *HubMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		receive:    make(chan *HubMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.UserUrn] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.UserUrn]; ok {
				delete(h.clients, client.UserUrn)
				close(client.send)
			}
		case hubMsg := <-h.receive:
			if client, ok := h.clients[hubMsg.client]; ok {
				client.send <- hubMsg.msg
			}
		}
	}
}
