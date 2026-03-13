package domains

import "time"

type TriggerWorkerBuildRequest struct {
	StackID string `json:"stack_id"`
	Role    string `json:"role"`
	Mode    string `json:"mode"`
}

type WorkerBuildReportRequest struct {
	WorkerBuildID  string            `json:"worker_build_id"`
	Status         string            `json:"status"`
	ImageReference string            `json:"image_reference,omitempty"`
	ImageDigest    string            `json:"image_digest,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	CallbackToken  string            `json:"-"` // set from X-Build-Token header, not JSON body
}

type WorkerBuildResponse struct {
	ID             string     `json:"id"`
	WorkerID       string     `json:"worker_id"`
	StackID        string     `json:"stack_id"`
	Role           string     `json:"role"`
	BuildMode      string     `json:"build_mode"`
	Status         string     `json:"status"`
	ImageReference *string    `json:"image_reference,omitempty"`
	ImageDigest    *string    `json:"image_digest,omitempty"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	BuildLog       *string    `json:"build_log,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
