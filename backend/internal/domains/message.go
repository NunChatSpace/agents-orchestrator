package domains

import "time"

type SendMessageRequest struct {
	Content string `json:"content"`
}

type MessageResponse struct {
	MessageID string    `json:"message_id"`
	JobID     string    `json:"job_id"`
	Role      string    `json:"role"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
