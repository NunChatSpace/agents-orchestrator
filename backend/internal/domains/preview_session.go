package domains

import "time"

// CreatePreviewSessionRoleRequest specifies one agent's participation in a preview session.
type CreatePreviewSessionRoleRequest struct {
	WorkerID string `json:"worker_id"`
	Role     string `json:"role"` // "frontend" | "backend" | "app"
}

// CreatePreviewSessionRequest is the payload for POST /preview-sessions.
type CreatePreviewSessionRequest struct {
	Roles []CreatePreviewSessionRoleRequest `json:"roles"`
}

// PreviewSessionRoleResponse is the API representation of one role in a preview session.
type PreviewSessionRoleResponse struct {
	ID         string     `json:"id"`
	WorkerID   string     `json:"worker_id"`
	WorkerName string     `json:"worker_name"`
	Role       string     `json:"role"`
	Port       *int       `json:"port,omitempty"`
	Status     string     `json:"status"`
	PreviewURL *string    `json:"preview_url,omitempty"`
	Error      *string    `json:"error_message,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PreviewSessionResponse is the API representation of a preview session.
type PreviewSessionResponse struct {
	ID        string                       `json:"id"`
	Status    string                       `json:"status"`
	Error     *string                      `json:"error,omitempty"`
	Roles     []PreviewSessionRoleResponse `json:"roles"`
	CreatedAt time.Time                    `json:"created_at"`
	StoppedAt *time.Time                   `json:"stopped_at,omitempty"`
}
