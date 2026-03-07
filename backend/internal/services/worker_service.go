package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
)

// WorkerService manages worker agent lifecycle.
type WorkerService interface {
	CreateWorker(ctx context.Context, req domains.CreateWorkerRequest) (*domains.WorkerResponse, error)
	UpdateWorker(ctx context.Context, workerID uuid.UUID, req domains.UpdateWorkerRequest) (*domains.WorkerResponse, error)
	ListWorkers(ctx context.Context) ([]*domains.WorkerResponse, error)
	GetWorker(ctx context.Context, workerID uuid.UUID) (*domains.WorkerResponse, error)
	PingWorker(ctx context.Context, workerID uuid.UUID) (*domains.PingWorkerResponse, error)
	DeleteWorker(ctx context.Context, workerID uuid.UUID) error
	// ws.WorkerFetcher implementation for hub broadcast payloads.
	GetWorkerJSON(ctx context.Context, workerID uuid.UUID) (any, error)
}

type workerService struct {
	workerRepo repository.WorkerRepository
}

func NewWorkerService(workerRepo repository.WorkerRepository) WorkerService {
	return &workerService{workerRepo: workerRepo}
}

// workspacesRoot returns the mount root for agent workspaces.
// Defaults to /workspaces if WORKSPACES_ROOT is not set.
func workspacesRoot() string {
	if root := os.Getenv("WORKSPACES_ROOT"); root != "" {
		return strings.TrimRight(root, "/")
	}
	return "/workspaces"
}

func (s *workerService) CreateWorker(ctx context.Context, req domains.CreateWorkerRequest) (*domains.WorkerResponse, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.GitRepoURL == "" {
		return nil, errors.New("git_repo_url is required")
	}
	if req.GroupName == "" {
		return nil, errors.New("group_name is required")
	}

	cliCommand := req.CLICommand
	if cliCommand != "claude" && cliCommand != "codex" {
		return nil, errors.New("cli_command must be 'claude' or 'codex'")
	}

	// Resolve workspace path.
	workspace := fmt.Sprintf("%s/%s", workspacesRoot(), req.Name)

	// Create workspace directory.
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace directory: %w", err)
	}

	// Clone the git repo into the workspace directory.
	// If GIT_TOKEN is set, inject it into HTTPS URLs so no credentials are needed in the URL.
	cloneURL := req.GitRepoURL
	if token := os.Getenv("GIT_TOKEN"); token != "" && strings.HasPrefix(cloneURL, "https://") {
		cloneURL = strings.Replace(cloneURL, "https://", fmt.Sprintf("https://x-access-token:%s@", token), 1)
	}
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, workspace)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=echo",
		// Skip the host SSH config — it may contain macOS-only options (e.g. UseKeychain)
		// that are unsupported on Linux/Alpine.
		"GIT_SSH_COMMAND=ssh -F /dev/null -o StrictHostKeyChecking=accept-new",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Clean up the directory on failure.
		_ = os.RemoveAll(workspace)
		return nil, fmt.Errorf("git clone failed: %w — %s", err, strings.TrimSpace(string(out)))
	}

	// Get or create the worker group.
	groupID, err := s.workerRepo.GetOrCreateGroup(ctx, req.GroupName)
	if err != nil {
		_ = os.RemoveAll(workspace)
		return nil, fmt.Errorf("resolve group: %w", err)
	}

	// Generate a random API key hash (not used for subprocess workers, but schema requires it).
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		_ = os.RemoveAll(workspace)
		return nil, fmt.Errorf("generate api key: %w", err)
	}
	hash := sha256.Sum256(rawKey)
	apiKeyHash := hex.EncodeToString(hash[:])

	worker := &models.Worker{
		WorkerID:   uuid.New(),
		GroupID:    groupID,
		Name:       req.Name,
		CallbackURL: "",
		APIKeyHash:  apiKeyHash,
		Workspace:   workspace,
		CLICommand:  cliCommand,
		GitRepoURL:  req.GitRepoURL,
		Status:      models.WorkerStatusIdle,
	}

	if err := s.workerRepo.CreateWorker(ctx, worker); err != nil {
		_ = os.RemoveAll(workspace)
		return nil, fmt.Errorf("save worker: %w", err)
	}

	return &domains.WorkerResponse{
		WorkerID:   worker.WorkerID.String(),
		GroupName:  req.GroupName,
		Name:       worker.Name,
		CLICommand: worker.CLICommand,
		Workspace:  worker.Workspace,
		GitRepoURL: worker.GitRepoURL,
		Status:     string(worker.Status),
	}, nil
}

func (s *workerService) UpdateWorker(ctx context.Context, workerID uuid.UUID, req domains.UpdateWorkerRequest) (*domains.WorkerResponse, error) {
	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, errors.New("worker not found")
	}

	if req.CLICommand != nil {
		if *req.CLICommand != "claude" && *req.CLICommand != "codex" {
			return nil, errors.New("cli_command must be 'claude' or 'codex'")
		}
		if err := s.workerRepo.UpdateCLICommand(ctx, workerID, *req.CLICommand); err != nil {
			return nil, err
		}
		worker.CLICommand = *req.CLICommand
	}

	return &domains.WorkerResponse{
		WorkerID:   worker.WorkerID.String(),
		GroupName:  worker.GroupName,
		Name:       worker.Name,
		CLICommand: worker.CLICommand,
		Workspace:  worker.Workspace,
		GitRepoURL: worker.GitRepoURL,
		Status:     string(worker.Status),
	}, nil
}

func (s *workerService) ListWorkers(ctx context.Context) ([]*domains.WorkerResponse, error) {
	workers, err := s.workerRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]*domains.WorkerResponse, len(workers))
	for i, w := range workers {
		resp[i] = &domains.WorkerResponse{
			WorkerID:     w.WorkerID.String(),
			GroupName:    w.GroupName,
			Name:         w.Name,
			CLICommand:   w.CLICommand,
			Workspace:    w.Workspace,
			GitRepoURL:   w.GitRepoURL,
			Status:       string(w.Status),
			LastActiveAt: w.LastActiveAt,
		}
	}
	return resp, nil
}

func (s *workerService) GetWorker(ctx context.Context, workerID uuid.UUID) (*domains.WorkerResponse, error) {
	w, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, errors.New("worker not found")
	}
	return &domains.WorkerResponse{
		WorkerID:     w.WorkerID.String(),
		GroupName:    w.GroupName,
		Name:         w.Name,
		CLICommand:   w.CLICommand,
		Workspace:    w.Workspace,
		GitRepoURL:   w.GitRepoURL,
		Status:       string(w.Status),
		LastActiveAt: w.LastActiveAt,
	}, nil
}

func (s *workerService) DeleteWorker(ctx context.Context, workerID uuid.UUID) error {
	if err := s.workerRepo.DeleteWorker(ctx, workerID); err != nil {
		return errors.New("worker not found")
	}
	return nil
}

func (s *workerService) GetWorkerJSON(ctx context.Context, workerID uuid.UUID) (any, error) {
	return s.GetWorker(ctx, workerID)
}

func (s *workerService) PingWorker(ctx context.Context, workerID uuid.UUID) (*domains.PingWorkerResponse, error) {
	w, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, errors.New("worker not found")
	}

	cliCmd := w.CLICommand
	if cliCmd == "" {
		cliCmd = "claude"
	}

	cmd := exec.CommandContext(ctx, cliCmd, "--version")
	cmd.Dir = w.Workspace
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return &domains.PingWorkerResponse{OK: false, Output: output, Error: err.Error()}, nil
	}
	return &domains.PingWorkerResponse{OK: true, Output: output}, nil
}
