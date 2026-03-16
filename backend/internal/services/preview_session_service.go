package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// PreviewSessionService manages process-based preview sessions.
type PreviewSessionService interface {
	Create(ctx context.Context, userID uuid.UUID, req domains.CreatePreviewSessionRequest) (*domains.PreviewSessionResponse, error)
	Get(ctx context.Context, sessionID uuid.UUID) (*domains.PreviewSessionResponse, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domains.PreviewSessionResponse, error)
	Stop(ctx context.Context, sessionID uuid.UUID) error
	Delete(ctx context.Context, sessionID uuid.UUID) error
	GetRoleLog(ctx context.Context, roleID uuid.UUID) (string, error)
}

type previewSessionService struct {
	sessionRepo repository.PreviewSessionRepository
	workerRepo  repository.WorkerRepository
	nginxSvc    NginxConfigService

	// processes tracks running preview subprocesses keyed by role ID string.
	processes sync.Map
}

func NewPreviewSessionService(
	sessionRepo repository.PreviewSessionRepository,
	workerRepo repository.WorkerRepository,
	nginxSvc NginxConfigService,
) PreviewSessionService {
	return &previewSessionService{
		sessionRepo: sessionRepo,
		workerRepo:  workerRepo,
		nginxSvc:    nginxSvc,
	}
}

func (s *previewSessionService) Create(ctx context.Context, userID uuid.UUID, req domains.CreatePreviewSessionRequest) (*domains.PreviewSessionResponse, error) {
	if len(req.Roles) == 0 {
		return nil, fmt.Errorf("at least one role is required")
	}

	// Validate all workers exist and have preview_command set.
	workersByID := make(map[uuid.UUID]*models.Worker, len(req.Roles))
	for _, r := range req.Roles {
		wid, err := uuid.Parse(r.WorkerID)
		if err != nil {
			return nil, fmt.Errorf("invalid worker_id %q: %w", r.WorkerID, err)
		}
		w, err := s.workerRepo.GetByID(ctx, wid)
		if err != nil {
			return nil, fmt.Errorf("worker %s not found", r.WorkerID)
		}
		if w.PreviewCommand == nil || strings.TrimSpace(*w.PreviewCommand) == "" {
			return nil, fmt.Errorf("worker %q has no preview_command configured", w.Name)
		}
		workersByID[wid] = w
	}

	// Create session record.
	now := time.Now().UTC()
	session := &models.PreviewSession{
		ID:        uuid.New(),
		CreatedBy: userID,
		Status:    models.PreviewSessionStatusStarting,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create preview session: %w", err)
	}

	// Build role map by role type so we can inject CORS env.
	roleByType := make(map[string]*models.Worker, len(req.Roles))
	for _, r := range req.Roles {
		wid, _ := uuid.Parse(r.WorkerID)
		roleByType[r.Role] = workersByID[wid]
	}

	// Allocate ports and create role records.
	roles := make([]*models.PreviewSessionRole, 0, len(req.Roles))
	for _, r := range req.Roles {
		wid, _ := uuid.Parse(r.WorkerID)

		port, err := s.sessionRepo.AllocatePort(ctx)
		if err != nil {
			return nil, fmt.Errorf("allocate port for role %q: %w", r.Role, err)
		}

		baseDomain := previewBaseDomain()
		tld := previewTLD()
		w := workersByID[wid]
		previewURL := fmt.Sprintf("http://%s.%s.%s", baseDomain, w.Name, tld)

		role := &models.PreviewSessionRole{
			ID:         uuid.New(),
			SessionID:  session.ID,
			WorkerID:   wid,
			Role:       r.Role,
			Port:       &port,
			Status:     models.PreviewRoleStatusStarting,
			PreviewURL: &previewURL,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.sessionRepo.CreateRole(ctx, role); err != nil {
			return nil, fmt.Errorf("create role record: %w", err)
		}
		roles = append(roles, role)
	}

	// Launch each preview process in background.
	for _, role := range roles {
		role := role
		w := workersByID[role.WorkerID]
		go s.startPreviewProcess(w, role, roleByType)
	}

	return s.buildResponse(ctx, session, roles, workersByID)
}

// startPreviewProcess starts the preview_command subprocess and updates status after health check.
func (s *previewSessionService) startPreviewProcess(w *models.Worker, role *models.PreviewSessionRole, roleByType map[string]*models.Worker) {
	ctx := context.Background()
	port := *role.Port

	// Build env vars to inject.
	env := os.Environ()
	env = append(env, fmt.Sprintf("PORT=%d", port))

	// CORS injection: backend gets CORS_ALLOWED_ORIGINS, frontend gets PUBLIC_API_BASE_URL.
	baseDomain := previewBaseDomain()
	tld := previewTLD()
	if role.Role == "backend" {
		if fw, ok := roleByType["frontend"]; ok {
			env = append(env, fmt.Sprintf("CORS_ALLOWED_ORIGINS=http://%s.%s.%s", baseDomain, fw.Name, tld))
		}
	}
	if role.Role == "frontend" {
		if bw, ok := roleByType["backend"]; ok {
			env = append(env, fmt.Sprintf("PUBLIC_API_BASE_URL=http://%s.%s.%s", baseDomain, bw.Name, tld))
		}
	}

	// Pipe both stdout and stderr through a single reader so all output is captured.
	cmd := exec.Command("sh", "-c", *w.PreviewCommand)
	cmd.Dir = w.Workspace
	cmd.Env = env

	outPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		errMsg := pipeErr.Error()
		_ = s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusFailed, nil, nil, &errMsg)
		s.maybeMarkSessionFailed(ctx, role.SessionID)
		return
	}
	errPipe, pipeErr2 := cmd.StderrPipe()
	if pipeErr2 != nil {
		errMsg := pipeErr2.Error()
		_ = s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusFailed, nil, nil, &errMsg)
		s.maybeMarkSessionFailed(ctx, role.SessionID)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Str("role", role.ID.String()).Str("worker", w.Name).Msg("preview start failed")
		errMsg := err.Error()
		_ = s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusFailed, nil, nil, &errMsg)
		s.maybeMarkSessionFailed(ctx, role.SessionID)
		return
	}

	s.processes.Store(role.ID.String(), cmd)

	// Stream stdout+stderr lines into DB in a background goroutine.
	var lastOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(outPipe, errPipe))
		var buf strings.Builder
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			buf.WriteString(line)
			lastOutput.WriteString(line)
			if buf.Len() >= 1024 {
				_ = s.sessionRepo.AppendRoleLog(ctx, role.ID, buf.String())
				buf.Reset()
			}
		}
		if buf.Len() > 0 {
			_ = s.sessionRepo.AppendRoleLog(ctx, role.ID, buf.String())
		}
	}()

	// Wait for health before marking running.
	healthURL := fmt.Sprintf("http://localhost:%d/", port)
	retries := healthCheckRetries()
	interval := healthCheckInterval()
	healthy := false
	for i := 0; i < retries; i++ {
		time.Sleep(interval)
		resp, err := http.Get(healthURL) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			healthy = true
			break
		}
	}

	if !healthy {
		log.Warn().Str("role", role.ID.String()).Str("worker", w.Name).Msg("preview health check failed; marking running anyway")
	}

	previewURL := *role.PreviewURL
	if err := s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusRunning, role.Port, &previewURL, nil); err != nil {
		log.Error().Err(err).Str("role", role.ID.String()).Msg("update role status failed")
	}

	// Update session status if all roles are now running.
	s.maybeMarkSessionRunning(ctx, role.SessionID)

	// Reload nginx.
	if err := s.nginxSvc.Reload(ctx); err != nil {
		log.Error().Err(err).Msg("nginx reload failed after preview start")
	}

	// Wait for process exit (e.g., crash).
	runErr := cmd.Wait()
	s.processes.Delete(role.ID.String())

	captured := strings.TrimSpace(lastOutput.String())
	errMsg := "process exited unexpectedly"
	if captured != "" {
		last := captured
		if len(last) > 300 {
			last = last[len(last)-300:]
		}
		errMsg = last
	}
	if runErr != nil {
		log.Error().Err(runErr).Str("role", role.ID.String()).Str("worker", w.Name).Str("output", captured).Msg("preview process exited")
	}
	_ = s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusFailed, nil, nil, &errMsg)
	s.maybeMarkSessionFailed(ctx, role.SessionID)
	_ = s.nginxSvc.Reload(ctx)
}

func (s *previewSessionService) maybeMarkSessionRunning(ctx context.Context, sessionID uuid.UUID) {
	roles, err := s.sessionRepo.GetRolesBySession(ctx, sessionID)
	if err != nil {
		return
	}
	for _, r := range roles {
		if r.Status != models.PreviewRoleStatusRunning {
			return
		}
	}
	_ = s.sessionRepo.UpdateStatus(ctx, sessionID, models.PreviewSessionStatusRunning, nil, nil)
}

func (s *previewSessionService) maybeMarkSessionFailed(ctx context.Context, sessionID uuid.UUID) {
	errMsg := "one or more roles failed to start"
	_ = s.sessionRepo.UpdateStatus(ctx, sessionID, models.PreviewSessionStatusFailed, &errMsg, nil)
}

func (s *previewSessionService) Get(ctx context.Context, sessionID uuid.UUID) (*domains.PreviewSessionResponse, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}
	roles, err := s.sessionRepo.GetRolesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	workersByID, err := s.fetchWorkers(ctx, roles)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, session, roles, workersByID)
}

func (s *previewSessionService) List(ctx context.Context, userID uuid.UUID) ([]*domains.PreviewSessionResponse, error) {
	sessions, err := s.sessionRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*domains.PreviewSessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		roles, err := s.sessionRepo.GetRolesBySession(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		workersByID, err := s.fetchWorkers(ctx, roles)
		if err != nil {
			return nil, err
		}
		resp, err := s.buildResponse(ctx, sess, roles, workersByID)
		if err != nil {
			return nil, err
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *previewSessionService) Stop(ctx context.Context, sessionID uuid.UUID) error {
	roles, err := s.sessionRepo.GetRolesBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get roles: %w", err)
	}
	for _, role := range roles {
		if val, ok := s.processes.Load(role.ID.String()); ok {
			if cmd, ok := val.(*exec.Cmd); ok && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			s.processes.Delete(role.ID.String())
		}
		_ = s.sessionRepo.UpdateRoleStatus(ctx, role.ID, models.PreviewRoleStatusStopped, nil, nil, nil)
	}
	now := time.Now().UTC()
	if err := s.sessionRepo.UpdateStatus(ctx, sessionID, models.PreviewSessionStatusStopped, nil, &now); err != nil {
		return err
	}
	return s.nginxSvc.Reload(ctx)
}

func (s *previewSessionService) Delete(ctx context.Context, sessionID uuid.UUID) error {
	_ = s.Stop(ctx, sessionID)
	now := time.Now().UTC()
	return s.sessionRepo.UpdateStatus(ctx, sessionID, models.PreviewSessionStatusStopped, nil, &now)
}

func (s *previewSessionService) GetRoleLog(ctx context.Context, roleID uuid.UUID) (string, error) {
	return s.sessionRepo.GetRoleLog(ctx, roleID)
}

// fetchWorkers resolves all unique worker IDs from roles into a map.
func (s *previewSessionService) fetchWorkers(ctx context.Context, roles []*models.PreviewSessionRole) (map[uuid.UUID]*models.Worker, error) {
	m := make(map[uuid.UUID]*models.Worker)
	for _, r := range roles {
		if _, ok := m[r.WorkerID]; ok {
			continue
		}
		w, err := s.workerRepo.GetByID(ctx, r.WorkerID)
		if err != nil {
			return nil, fmt.Errorf("worker %s not found: %w", r.WorkerID, err)
		}
		m[r.WorkerID] = w
	}
	return m, nil
}

func (s *previewSessionService) buildResponse(_ context.Context, session *models.PreviewSession, roles []*models.PreviewSessionRole, workers map[uuid.UUID]*models.Worker) (*domains.PreviewSessionResponse, error) {
	roleResps := make([]domains.PreviewSessionRoleResponse, 0, len(roles))
	for _, r := range roles {
		workerName := ""
		if w, ok := workers[r.WorkerID]; ok {
			workerName = w.Name
		}
		roleResps = append(roleResps, domains.PreviewSessionRoleResponse{
			ID:         r.ID.String(),
			WorkerID:   r.WorkerID.String(),
			WorkerName: workerName,
			Role:       r.Role,
			Port:       r.Port,
			Status:     string(r.Status),
			PreviewURL: r.PreviewURL,
			Error:      r.ErrorMessage,
			UpdatedAt:  r.UpdatedAt,
		})
	}
	return &domains.PreviewSessionResponse{
		ID:        session.ID.String(),
		Status:    string(session.Status),
		Error:     session.Error,
		Roles:     roleResps,
		CreatedAt: session.CreatedAt,
		StoppedAt: session.StoppedAt,
	}, nil
}

func previewBaseDomain() string {
	if v := os.Getenv("PREVIEW_BASE_DOMAIN"); v != "" {
		return v
	}
	return "shiphide"
}

func previewTLD() string {
	if v := os.Getenv("PREVIEW_TLD"); v != "" {
		return v
	}
	return "preview"
}

func healthCheckRetries() int {
	if v := os.Getenv("HEALTH_CHECK_RETRIES"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			return n
		}
	}
	return 20
}

func healthCheckInterval() time.Duration {
	if v := os.Getenv("HEALTH_CHECK_INTERVAL_SECONDS"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 3 * time.Second
}
