package controller

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// Manager is the realtime hub: it tracks the open websocket connections
// of every user and fans events out to them.
//
// NOTE: the hub lives in memory, so events only reach clients connected
// to this server instance. Running several instances needs a shared
// pub/sub (e.g. Redis) behind SendToUsers.
type Manager struct {
	mu      sync.RWMutex
	clients map[int]map[*Client]struct{}
}

var M *Manager

func NewManager() *Manager {
	return &Manager{clients: map[int]map[*Client]struct{}{}}
}

const (
	wsWriteWait       = 10 * time.Second
	wsPongWait        = 60 * time.Second
	wsPingPeriod      = 50 * time.Second
	wsSessionRecheck  = 5 * time.Minute
	wsMaxMessageBytes = 8 * 1024
	wsSendBuffer      = 64
	wsMaxPerSecond    = 5
)

// Client is one websocket connection of a user. A user can have several
// (tabs, devices).
type Client struct {
	userID int
	token  string
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once

	// simple rate limit for inbound messages
	windowStart time.Time
	windowCount int
}

// Serve registers the connection and starts its read / write pumps.
func (m *Manager) Serve(conn *websocket.Conn, session model.Session) {
	c := &Client{userID: session.UserId, token: session.Token, conn: conn, send: make(chan []byte, wsSendBuffer)}
	m.register(c)
	go c.writePump(m)
	go c.readPump(m)
}

func (m *Manager) register(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.clients[c.userID]
	if !ok {
		set = map[*Client]struct{}{}
		m.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (m *Manager) unregister(c *Client) {
	m.mu.Lock()
	if set, ok := m.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(m.clients, c.userID)
		}
	}
	m.mu.Unlock()
	c.once.Do(func() { close(c.send) })
}

// IsOnline reports whether the user has at least one open connection.
func (m *Manager) IsOnline(userID int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients[userID]) > 0
}

// SendToUsers delivers the event to every connection of the users.
// Slow clients whose buffer is full are dropped.
func (m *Manager) SendToUsers(userIDs []int, event model.WSEvent) {
	if m == nil || len(userIDs) == 0 {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return
	}

	var slow []*Client
	m.mu.RLock()
	for _, id := range userIDs {
		for c := range m.clients[id] {
			select {
			case c.send <- data:
			default:
				slow = append(slow, c)
			}
		}
	}
	m.mu.RUnlock()

	for _, c := range slow {
		m.unregister(c)
	}
}

// notify pushes a notification to online users. It is a no-op when the
// hub is not running (e.g. in tests).
func notify(userIDs []int, n model.Notification) {
	if M == nil {
		return
	}
	M.SendToUsers(userIDs, model.WSEvent{Type: "notification", Notification: &n})
}

func (c *Client) sendEvent(event model.WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	defer func() { recover() }() // send on a closed channel after unregister
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) allow() bool {
	now := time.Now()
	if now.Sub(c.windowStart) >= time.Second {
		c.windowStart = now
		c.windowCount = 0
	}
	c.windowCount++
	return c.windowCount <= wsMaxPerSecond
}

func (c *Client) readPump(m *Manager) {
	defer func() {
		m.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(wsMaxMessageBytes)
	c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		var in model.WSInbound
		if err := c.conn.ReadJSON(&in); err != nil {
			if _, ok := err.(*json.SyntaxError); ok {
				c.sendEvent(model.WSEvent{Type: "error", Error: "invalid json"})
				continue
			}
			return
		}
		if !c.allow() {
			c.sendEvent(model.WSEvent{Type: "error", ClientId: in.ClientId, Error: rateLimitedError.Error()})
			continue
		}
		handleInbound(c, in)
	}
}

func (c *Client) writePump(m *Manager) {
	ping := time.NewTicker(wsPingPeriod)
	recheck := time.NewTicker(wsSessionRecheck)
	defer func() {
		ping.Stop()
		recheck.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ping.C:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-recheck.C:
			// close the socket once the session is logged out / invalid;
			// the client reconnects with a fresh token
			if _, err := IsAuthorized(c.token, 0); err != nil {
				c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session expired"))
				return
			}
		}
	}
}

// handleInbound dispatches a client event.
func handleInbound(c *Client, in model.WSInbound) {
	switch in.Type {
	case "send":
		if _, err := SendMessage(c.userID, in.ConversationId, in.Body, in.ClientId); err != nil {
			c.sendEvent(model.WSEvent{Type: "error", ConversationId: in.ConversationId, ClientId: in.ClientId, Error: err.Error()})
		}
	case "typing":
		BroadcastTyping(c.userID, in.ConversationId)
	case "read":
		if err := MarkConversationRead(c.userID, in.ConversationId, in.MessageId); err != nil {
			c.sendEvent(model.WSEvent{Type: "error", ConversationId: in.ConversationId, Error: err.Error()})
		}
	case "ping":
		c.sendEvent(model.WSEvent{Type: "pong"})
	default:
		c.sendEvent(model.WSEvent{Type: "error", Error: "unknown event type"})
	}
}
