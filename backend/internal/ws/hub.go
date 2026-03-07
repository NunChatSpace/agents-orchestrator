package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Event types pushed to the frontend.
const (
	EventJobUpdated     = "job_updated"
	EventMessageAdded   = "message_added"
	EventWorkerUpdated  = "worker_updated"
)

// Event is a typed payload sent over a WebSocket connection.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// client represents one WebSocket connection.
type client struct {
	userID uuid.UUID
	send   chan []byte
}

// Hub manages all active WebSocket connections and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID][]*client

	// Injected by the job service to fetch response DTOs for broadcast payloads.
	// Resolved lazily to avoid circular DI.
	jobFetcher     JobFetcher
	messageFetcher MessageFetcher
	workerFetcher  WorkerFetcher
}

// JobFetcher, MessageFetcher, WorkerFetcher are narrow interfaces so the hub
// can build broadcast payloads without importing the full service layer.
type JobFetcher interface {
	GetJobJSON(ctx context.Context, jobID uuid.UUID) (any, error)
}
type MessageFetcher interface {
	GetMessageJSON(ctx context.Context, msg *models.Message) any
}
type WorkerFetcher interface {
	GetWorkerJSON(ctx context.Context, workerID uuid.UUID) (any, error)
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID][]*client),
	}
}

// SetFetchers wires the broadcast payload resolvers after construction.
func (h *Hub) SetFetchers(jf JobFetcher, mf MessageFetcher, wf WorkerFetcher) {
	h.jobFetcher = jf
	h.messageFetcher = mf
	h.workerFetcher = wf
}

// Register adds a new WebSocket client for the given user.
func (h *Hub) Register(userID uuid.UUID, send chan []byte) *client {
	c := &client{userID: userID, send: send}
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], c)
	h.mu.Unlock()
	return c
}

// Unregister removes a client connection.
func (h *Hub) Unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[c.userID]
	for i, existing := range list {
		if existing == c {
			h.clients[c.userID] = append(list[:i], list[i+1:]...)
			close(c.send)
			break
		}
	}
	if len(h.clients[c.userID]) == 0 {
		delete(h.clients, c.userID)
	}
}

// broadcast encodes and sends an event to all connections for a user.
func (h *Hub) broadcast(userID uuid.UUID, evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Error().Err(err).Msg("ws: marshal error")
		return
	}
	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()
	for _, c := range clients {
		select {
		case c.send <- data:
		default:
			log.Warn().Str("user", userID.String()).Msg("ws: slow client, dropping event")
		}
	}
}

// broadcastAll sends to every connected user.
func (h *Hub) broadcastAll(evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Error().Err(err).Msg("ws: marshal error")
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for _, c := range clients {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// BroadcastJobUpdated implements services.WSBroadcaster.
func (h *Hub) BroadcastJobUpdated(ctx context.Context, jobID uuid.UUID) {
	var payload any = map[string]string{"job_id": jobID.String()}
	if h.jobFetcher != nil {
		if p, err := h.jobFetcher.GetJobJSON(ctx, jobID); err == nil {
			payload = p
		}
	}
	h.broadcastAll(Event{Type: EventJobUpdated, Data: payload})
}

// BroadcastMessageAdded implements services.WSBroadcaster.
func (h *Hub) BroadcastMessageAdded(ctx context.Context, jobID uuid.UUID, msg *models.Message) {
	var payload any = map[string]string{"job_id": jobID.String()}
	if h.messageFetcher != nil {
		payload = h.messageFetcher.GetMessageJSON(ctx, msg)
	}
	h.broadcastAll(Event{Type: EventMessageAdded, Data: payload})
}

// BroadcastWorkerUpdated implements services.WSBroadcaster.
func (h *Hub) BroadcastWorkerUpdated(ctx context.Context, workerID uuid.UUID) {
	var payload any = map[string]string{"worker_id": workerID.String()}
	if h.workerFetcher != nil {
		if p, err := h.workerFetcher.GetWorkerJSON(ctx, workerID); err == nil {
			payload = p
		}
	}
	h.broadcastAll(Event{Type: EventWorkerUpdated, Data: payload})
}
