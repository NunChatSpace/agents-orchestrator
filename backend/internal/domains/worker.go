package domains

import "time"

// WorkerReplyRequest is the payload a WAgent sends to POST /api/v1/workers/reply.
type WorkerReplyRequest struct {
	JobID    string    `json:"job_id"`
	WorkerID string    `json:"worker_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	ResumeID *string   `json:"resume_id,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkerResponse is the API representation of a worker.
type WorkerResponse struct {
	WorkerID     string     `json:"worker_id"`
	GroupName    string     `json:"group_name"`
	Name         string     `json:"name"`
	CLICommand   string     `json:"cli_command"`
	Workspace    string     `json:"workspace"`
	GitRepoURL   string     `json:"git_repo_url"`
	Status       string     `json:"status"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

// CreateWorkerRequest is the payload to POST /api/v1/workers.
type CreateWorkerRequest struct {
	Name       string `json:"name"`
	GitRepoURL string `json:"git_repo_url"`
	GroupName  string `json:"group_name"`
	CLICommand string `json:"cli_command"` // "claude" or "codex"
}

// UpdateWorkerRequest is the payload to PATCH /api/v1/workers/{worker_id}.
type UpdateWorkerRequest struct {
	CLICommand *string `json:"cli_command,omitempty"`
}

// PingWorkerResponse is returned by POST /api/v1/workers/{worker_id}/ping.
type PingWorkerResponse struct {
	OK      bool   `json:"ok"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// DispatchRequest is what OAgent sends to a WAgent's callback_url.
type DispatchRequest struct {
	JobID             string  `json:"job_id"`
	WorkerID          string  `json:"worker_id"`
	Action            string  `json:"action"` // "new" | "continue" | "cancel"
	ResumeID          *string `json:"resume_id,omitempty"`
	Instruction       string  `json:"instruction"`
	OAgentCallbackURL string  `json:"oagent_callback_url"`
	AuthKey           string  `json:"auth_key"`
}
