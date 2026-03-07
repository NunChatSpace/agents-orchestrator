package domains

import "time"

type CreateJobRequest struct {
	Title                string  `json:"title"`
	Prompt               string  `json:"prompt"`
	TargetGroup          string  `json:"target_group"`
	ManualWorkerOverride *string `json:"manual_worker_override,omitempty"`
}

type PatchJobRequest struct {
	Title       *string `json:"title,omitempty"`
	Prompt      *string `json:"prompt,omitempty"`
	TargetGroup *string `json:"target_group,omitempty"`
}

type JobFilters struct {
	Status      *string `json:"status,omitempty"`
	TargetGroup *string `json:"target_group,omitempty"`
	Limit       int
	Offset      int
}

type JobResponse struct {
	JobID                string     `json:"job_id"`
	Title                string     `json:"title"`
	Prompt               string     `json:"prompt"`
	TargetGroup          string     `json:"target_group"`
	AssignedWorkerID     *string    `json:"assigned_worker_id,omitempty"`
	AssignedWorkerName   *string    `json:"assigned_worker_name,omitempty"`
	ManualWorkerOverride *string    `json:"manual_worker_override,omitempty"`
	Status               string     `json:"status"`
	ResumeID             *string    `json:"resume_id,omitempty"`
	FailureReason        *string    `json:"failure_reason,omitempty"`
	MessageCount         int        `json:"message_count"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
