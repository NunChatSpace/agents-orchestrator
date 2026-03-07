package ws

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS handled by middleware
	},
}

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second
	maxMsgSize = 512
)

// ServeWS upgrades the HTTP connection to WebSocket for the given user.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws: upgrade failed")
		return
	}

	send := make(chan []byte, 64)
	c := h.Register(userID, send)
	defer h.Unregister(c)

	// Write pump
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		defer conn.Close()
		for {
			select {
			case msg, ok := <-send:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read pump — just drain pong/close frames
	conn.SetReadLimit(maxMsgSize)
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pingPeriod + writeWait))
		return nil
	})
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
