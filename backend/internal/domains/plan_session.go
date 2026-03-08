package domains

import (
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
)

type CreatePlanSessionRequest struct {
	WorkerID string `json:"worker_id"`
}

type UpdatePlanSessionTitleRequest struct {
	Title string `json:"title"`
}

type SendPlanMessageRequest struct {
	Content string `json:"content"`
}

type CompletePlanSessionRequest struct {
	GeneratedPrompt string `json:"generated_prompt"`
}

type PlanSessionResponse struct {
	ID              uuid.UUID                `json:"id"`
	WorkerID        uuid.UUID                `json:"worker_id"`
	Title           string                   `json:"title"`
	Status          models.PlanSessionStatus `json:"status"`
	GeneratedPrompt *string                  `json:"generated_prompt,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	Messages        []PlanMessageResponse    `json:"messages,omitempty"`
}

type PlanMessageResponse struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
