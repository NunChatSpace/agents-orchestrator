package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// JobResultHandler is the subset of JobService the dispatcher calls back with results.
// Implemented by jobService; injected via SetResultHandler after both services are built
// (avoids the circular dependency dispatcher ↔ job service).
type JobResultHandler interface {
	HandleWorkerReply(ctx context.Context, req domains.WorkerReplyRequest) error
	HandleThinkingStep(ctx context.Context, jobID uuid.UUID, content string) error
	HandleFileChange(ctx context.Context, jobID uuid.UUID, content string) error
}

type dispatcherService struct {
	workerRepo repository.WorkerRepository
	mu         sync.RWMutex
	handler    JobResultHandler

	// active subprocess per job_id (for cancellation)
	processes sync.Map
}

func NewDispatcherService(workerRepo repository.WorkerRepository) DispatcherService {
	return &dispatcherService{workerRepo: workerRepo}
}

// SetResultHandler wires the callback after DI construction.
func (d *dispatcherService) SetResultHandler(h JobResultHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = h
}

func (d *dispatcherService) getHandler() JobResultHandler {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.handler
}

// SendToWorker spawns the CLI agent for a new or resumed job (async).
func (d *dispatcherService) SendToWorker(ctx context.Context, workerID uuid.UUID, job *models.Job) error {
	worker, err := d.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return fmt.Errorf("get worker: %w", err)
	}
	resumeID := ""
	if job.ResumeID != nil {
		resumeID = *job.ResumeID
	}
	go d.runCLI(worker, job.JobID, resumeID, models.WorkerInstructionFieldJob, job.Prompt)
	return nil
}

// NotifyUserReply forwards a user reply to the assigned worker (continue).
// resumeID is the Claude session ID from the previous turn; pass "" to start fresh.
func (d *dispatcherService) NotifyUserReply(ctx context.Context, workerID uuid.UUID, jobID uuid.UUID, resumeID string, message string) error {
	worker, err := d.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return fmt.Errorf("get worker: %w", err)
	}
	go d.runCLI(worker, jobID, resumeID, models.WorkerInstructionFieldJob, message)
	return nil
}

// CancelWorkerJob kills the running subprocess for the given job.
func (d *dispatcherService) CancelWorkerJob(ctx context.Context, workerID uuid.UUID, jobID uuid.UUID) error {
	if val, ok := d.processes.Load(jobID.String()); ok {
		if cmd, ok := val.(*exec.Cmd); ok && cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.Warn().Err(err).Str("job", jobID.String()).Msg("kill CLI process")
			}
		}
	}
	return nil
}

// RunBuildInstruction executes a synthetic build instruction, streaming each output line to logFn.
func (d *dispatcherService) RunBuildInstruction(ctx context.Context, workerID uuid.UUID, buildID uuid.UUID, message string, logFn func(string)) error {
	worker, err := d.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return fmt.Errorf("get worker: %w", err)
	}

	instruction, err := composeWorkerInstruction(worker, models.WorkerInstructionFieldJob, message)
	if err != nil {
		return err
	}

	cliCmd := worker.CLICommand
	if cliCmd == "" {
		cliCmd = "claude"
	}
	isClaude := cliCmd == "claude"

	var args []string
	if isClaude {
		args = []string{"-p", instruction, "--output-format", "stream-json"}
	} else {
		args = []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
	}

	cmd := exec.CommandContext(ctx, cliCmd, args...)
	cmd.Dir = worker.Workspace

	stdoutPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return fmt.Errorf("build CLI stdout pipe: %w", pipeErr)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("build CLI start: %w", startErr)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if logFn != nil {
			if text := extractBuildLogLine(line, isClaude); text != "" {
				logFn(text)
			}
		}
	}

	if runErr := cmd.Wait(); runErr != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		log.Error().
			Err(runErr).
			Str("worker_build_id", buildID.String()).
			Str("worker", worker.Name).
			Str("stderr", stderrStr).
			Msg("build CLI failed")
		if stderrStr != "" {
			return fmt.Errorf("build CLI failed: %s", stderrStr)
		}
		return fmt.Errorf("build CLI failed: %w", runErr)
	}
	return nil
}

// extractBuildLogLine extracts a human-readable string from one JSON line of CLI output.
func extractBuildLogLine(line string, isClaude bool) string {
	var obj map[string]any
	if json.Unmarshal([]byte(line), &obj) != nil {
		return line
	}
	if isClaude {
		if obj["type"] == "assistant" {
			msg, _ := obj["message"].(map[string]any)
			if msg != nil {
				for _, c := range func() []any { v, _ := msg["content"].([]any); return v }() {
					item, _ := c.(map[string]any)
					if item == nil {
						continue
					}
					if item["type"] == "text" {
						if t, ok := item["text"].(string); ok && t != "" {
							return t
						}
					}
					if item["type"] == "tool_use" {
						toolName, _ := item["name"].(string)
						inputBytes, _ := json.Marshal(item["input"])
						return fmt.Sprintf("[tool] %s %s", toolName, truncateBytes(inputBytes, 160))
					}
				}
			}
		}
		if obj["type"] == "result" {
			if r, ok := obj["result"].(string); ok && r != "" {
				return "[done] " + truncateBytes([]byte(r), 200)
			}
		}
	} else {
		if obj["type"] == "item.completed" {
			item, _ := obj["item"].(map[string]any)
			if item != nil {
				if t, ok := item["text"].(string); ok && t != "" {
					return t
				}
			}
		}
	}
	return ""
}

// runCLI executes the CLI agent, streams thinking steps, and calls back with the final result.
func (d *dispatcherService) runCLI(worker *models.Worker, jobID uuid.UUID, resumeID string, field models.WorkerInstructionField, message string) {
	jobKey := jobID.String()

	instruction, err := composeWorkerInstruction(worker, field, message)
	if err != nil {
		log.Error().Err(err).Str("job", jobKey).Msg("compose instruction failed")
		h := d.getHandler()
		if h != nil {
			req := domains.WorkerReplyRequest{
				JobID:     jobKey,
				WorkerID:  worker.WorkerID.String(),
				Status:    "offline",
				Message:   err.Error(),
				UpdatedAt: time.Now().UTC(),
			}
			_ = h.HandleWorkerReply(context.Background(), req)
		}
		return
	}

	cliCmd := worker.CLICommand
	if cliCmd == "" {
		cliCmd = "claude"
	}

	isClaude := cliCmd == "claude"

	// claude: claude -p [--resume {id}] {instruction} --output-format stream-json
	// codex: codex exec [resume <id>] --full-auto --json {instruction}
	var args []string
	if isClaude {
		args = []string{"-p"}
		if resumeID != "" {
			args = append(args, "--resume", resumeID)
		}
		args = append(args, instruction, "--output-format", "stream-json")
	} else {
		if resumeID != "" {
			args = []string{"exec", "resume", resumeID, "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
		} else {
			args = []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
		}
	}

	var stderr bytes.Buffer
	cmd := exec.Command(cliCmd, args...)
	cmd.Dir = worker.Workspace
	cmd.Stderr = &stderr

	stdoutPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		log.Error().Err(pipeErr).Str("job", jobKey).Msg("stdout pipe failed")
		return
	}

	// Snapshot HEAD before the agent runs so we can diff against it later,
	// even if the agent commits changes during the run.
	var beforeHash string
	if hashOut, hashErr := func() ([]byte, error) {
		c := exec.Command("git", "rev-parse", "HEAD")
		c.Dir = worker.Workspace
		return c.Output()
	}(); hashErr == nil {
		beforeHash = strings.TrimSpace(string(hashOut))
	}

	d.processes.Store(jobKey, cmd)
	if startErr := cmd.Start(); startErr != nil {
		d.processes.Delete(jobKey)
		log.Error().Err(startErr).Str("job", jobKey).Msg("CLI start failed")
		h := d.getHandler()
		if h != nil {
			req := domains.WorkerReplyRequest{
				JobID:     jobKey,
				WorkerID:  worker.WorkerID.String(),
				Status:    "offline",
				Message:   startErr.Error(),
				UpdatedAt: time.Now().UTC(),
			}
			_ = h.HandleWorkerReply(context.Background(), req)
		}
		return
	}

	var finalResult, finalResumeID string
	var rawLines []string

	h := d.getHandler()
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		rawLines = append(rawLines, line)

		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		if isClaude {
			switch obj["type"] {
			case "assistant":
				if h == nil {
					break
				}
				msg, _ := obj["message"].(map[string]any)
				if msg == nil {
					break
				}
				contents, _ := msg["content"].([]any)
				for _, c := range contents {
					item, _ := c.(map[string]any)
					if item == nil || item["type"] != "tool_use" {
						continue
					}
					toolName, _ := item["name"].(string)
					inputBytes, _ := json.Marshal(item["input"])
					step := fmt.Sprintf("%s: %s", toolName, truncateBytes(inputBytes, 300))
					_ = h.HandleThinkingStep(context.Background(), jobID, step)
				}
			case "result":
				if r, ok := obj["result"].(string); ok && r != "" {
					finalResult = r
				}
				if sid, ok := obj["session_id"].(string); ok {
					finalResumeID = sid
				}
			}
		} else {
			// codex JSONL events:
			// {"type":"thread.started","thread_id":"<uuid>"}
			// {"type":"item.completed","item":{"type":"reasoning","text":"..."}}
			// {"type":"item.completed","item":{"type":"agent_message","text":"..."}}
			switch obj["type"] {
			case "thread.started":
				if tid, ok := obj["thread_id"].(string); ok && tid != "" {
					finalResumeID = tid
				}
			case "item.completed":
				item, _ := obj["item"].(map[string]any)
				if item == nil {
					break
				}
				itemType, _ := item["type"].(string)
				text, _ := item["text"].(string)
				switch itemType {
				case "reasoning":
					if h != nil && text != "" {
						_ = h.HandleThinkingStep(context.Background(), jobID, text)
					}
				case "agent_message":
					if text != "" {
						finalResult = text
					}
				}
			}
		}
	}

	runErr := cmd.Wait()
	d.processes.Delete(jobKey)

	if h == nil {
		log.Error().Str("job", jobKey).Msg("no result handler — dropping CLI result")
		return
	}

	req := domains.WorkerReplyRequest{
		JobID:     jobKey,
		WorkerID:  worker.WorkerID.String(),
		UpdatedAt: time.Now().UTC(),
	}

	if runErr != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		stdoutStr := strings.TrimSpace(strings.Join(rawLines, "\n"))
		log.Error().Err(runErr).Str("job", jobKey).Str("worker", worker.Name).
			Str("stderr", stderrStr).Str("stdout", truncateBytes([]byte(stdoutStr), 500)).
			Str("cmd", cliCmd).Strs("args", args).Str("dir", worker.Workspace).
			Msg("CLI failed")
		req.Status = "offline"
		req.Message = fmt.Sprintf("%s\n%s", runErr.Error(), stderrStr)
	} else {
		// Use structured result if parsed; otherwise fall back to raw output.
		if finalResult == "" && len(rawLines) > 0 {
			finalResult = strings.TrimSpace(strings.Join(rawLines, "\n"))
		}
		req.Status = "idle"
		req.Message = finalResult
		if finalResumeID != "" {
			req.ResumeID = &finalResumeID
		}
		// Capture git diff for all files changed during this run.
		d.captureGitDiff(worker, jobID, beforeHash)
	}

	if err := h.HandleWorkerReply(context.Background(), req); err != nil {
		log.Error().Err(err).Str("job", jobKey).Msg("HandleWorkerReply failed")
	}
}

// captureGitDiff stages all workspace changes, diffs them against beforeHash,
// then resets the index — capturing both modified and new (untracked) files.
func (d *dispatcherService) captureGitDiff(worker *models.Worker, jobID uuid.UUID, beforeHash string) {
	ref := "HEAD"
	if beforeHash != "" {
		ref = beforeHash
	}
	// Stage everything (including new untracked files) so they appear in --cached diff.
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = worker.Workspace
	_ = addCmd.Run()

	diffCmd := exec.Command("git", "diff", "--cached", ref)
	diffCmd.Dir = worker.Workspace
	out, err := diffCmd.Output()

	// Always reset the index to leave the workspace clean.
	resetCmd := exec.Command("git", "reset", "HEAD")
	resetCmd.Dir = worker.Workspace
	_ = resetCmd.Run()

	if err != nil || len(out) == 0 {
		return
	}

	// Save a physical patch file the user can apply with "git apply".
	patchDir := filepath.Join(worker.Workspace, ".agent", "patches")
	if mkErr := os.MkdirAll(patchDir, 0o755); mkErr == nil {
		_ = os.WriteFile(filepath.Join(patchDir, jobID.String()+".patch"), out, 0o644)
	}

	h := d.getHandler()
	if h == nil {
		return
	}
	for filePath, diff := range splitGitDiff(string(out)) {
		content, _ := json.Marshal(map[string]string{"path": filePath, "diff": diff})
		_ = h.HandleFileChange(context.Background(), jobID, string(content))
	}
}

// splitGitDiff splits a unified diff into per-file sections keyed by file path.
// "diff --git a/foo b/foo" → key "foo", value = full section text.
func splitGitDiff(raw string) map[string]string {
	result := map[string]string{}
	var current strings.Builder
	var currentPath string
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if currentPath != "" {
				result[currentPath] = strings.TrimRight(current.String(), "\n")
			}
			current.Reset()
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentPath = strings.TrimPrefix(parts[3], "b/")
			}
		}
		current.WriteString(line + "\n")
	}
	if currentPath != "" {
		result[currentPath] = strings.TrimRight(current.String(), "\n")
	}
	return result
}

func truncateBytes(b []byte, max int) string {
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// RunPlanStream runs the worker's CLI and streams the reply via SSE to w.
// It returns the full reply text and the new resume_id so the caller can persist them.
func (d *dispatcherService) RunPlanStream(ctx context.Context, w http.ResponseWriter, workerID uuid.UUID, resumeID string, field models.WorkerInstructionField, message string) (string, string, error) {
	worker, err := d.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return "", "", fmt.Errorf("get worker: %w", err)
	}
	instruction, err := composeWorkerInstruction(worker, field, message)
	if err != nil {
		return "", "", err
	}

	cliCmd := worker.CLICommand
	if cliCmd == "" {
		cliCmd = "claude"
	}
	isClaude := cliCmd == "claude"

	var args []string
	if isClaude {
		args = []string{"-p"}
		if resumeID != "" {
			args = append(args, "--resume", resumeID)
		}
		args = append(args, instruction, "--output-format", "stream-json")
	} else {
		if resumeID != "" {
			args = []string{"exec", "resume", resumeID, "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
		} else {
			args = []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
		}
	}

	// Use a background context so the CLI process survives HTTP client disconnects.
	// The reply is still saved to DB after the process finishes.
	cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cmdCancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, cliCmd, args...)
	cmd.Dir = worker.Workspace
	cmd.Stderr = &stderr

	stdoutPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return "", "", fmt.Errorf("stdout pipe: %w", pipeErr)
	}
	if startErr := cmd.Start(); startErr != nil {
		return "", "", fmt.Errorf("CLI start: %w", startErr)
	}

	flusher, canFlush := w.(http.Flusher)

	writeSSE := func(event, data string) {
		// Encode data as JSON string to avoid SSE multiline issues.
		encoded, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(encoded))
		if canFlush {
			flusher.Flush()
		}
	}

	var finalResult, finalResumeID string
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) != nil {
			continue
		}

		if isClaude {
			switch obj["type"] {
			case "assistant":
				msg, _ := obj["message"].(map[string]any)
				if msg == nil {
					break
				}
				contents, _ := msg["content"].([]any)
				for _, c := range contents {
					item, _ := c.(map[string]any)
					if item == nil {
						continue
					}
					if item["type"] == "tool_use" {
						toolName, _ := item["name"].(string)
						inputBytes, _ := json.Marshal(item["input"])
						writeSSE("thinking", fmt.Sprintf("%s: %s", toolName, truncateBytes(inputBytes, 300)))
					}
				}
			case "result":
				if r, ok := obj["result"].(string); ok && r != "" {
					finalResult = r
				}
				if sid, ok := obj["session_id"].(string); ok {
					finalResumeID = sid
				}
			}
		} else {
			switch obj["type"] {
			case "thread.started":
				if tid, ok := obj["thread_id"].(string); ok && tid != "" {
					finalResumeID = tid
				}
			case "item.completed":
				item, _ := obj["item"].(map[string]any)
				if item == nil {
					break
				}
				itemType, _ := item["type"].(string)
				text, _ := item["text"].(string)
				switch itemType {
				case "reasoning":
					if text != "" {
						writeSSE("thinking", text)
					}
				case "agent_message":
					if text != "" {
						finalResult = text
					}
				}
			}
		}
	}

	if runErr := cmd.Wait(); runErr != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", "", fmt.Errorf("agent CLI failed: %s", stderrStr)
		}
		return "", "", fmt.Errorf("agent CLI exited with code %d", cmd.ProcessState.ExitCode())
	}

	return finalResult, finalResumeID, nil
}

// RunPlan runs the worker's CLI synchronously with a planning prompt and returns the refined prompt text.
func (d *dispatcherService) RunPlan(ctx context.Context, workerID uuid.UUID, task string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	worker, err := d.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return "", fmt.Errorf("get worker: %w", err)
	}

	cliCmd := worker.CLICommand
	if cliCmd == "" {
		cliCmd = "claude"
	}
	isClaude := cliCmd == "claude"

	instruction, err := composeWorkerInstruction(worker, models.WorkerInstructionFieldPlan, "Please write a clear, detailed task prompt for the following goal. Output ONLY the prompt text:\n\n"+task)
	if err != nil {
		return "", err
	}

	var args []string
	if isClaude {
		args = []string{"-p", instruction, "--output-format", "stream-json"}
	} else {
		args = []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--json", "--skip-git-repo-check", instruction}
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, cliCmd, args...)
	cmd.Dir = worker.Workspace
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("plan generation timed out after 2 minutes")
		}
		if stderrStr != "" {
			return "", fmt.Errorf("agent CLI failed: %s", stderrStr)
		}
		return "", fmt.Errorf("agent CLI exited with code %d", cmd.ProcessState.ExitCode())
	}

	var finalResult string
	var rawLines []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		rawLines = append(rawLines, line)
		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) != nil {
			continue
		}
		if isClaude {
			if obj["type"] == "result" {
				if r, ok := obj["result"].(string); ok && r != "" {
					finalResult = r
				}
			}
		} else {
			if obj["type"] == "item.completed" {
				item, _ := obj["item"].(map[string]any)
				if item != nil && item["type"] == "agent_message" {
					if t, ok := item["text"].(string); ok && t != "" {
						finalResult = t
					}
				}
			}
		}
	}
	if finalResult == "" && len(rawLines) > 0 {
		finalResult = strings.TrimSpace(strings.Join(rawLines, "\n"))
	}
	return finalResult, nil
}
