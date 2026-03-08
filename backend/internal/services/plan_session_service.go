package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
)

// PlanSessionService manages plan sessions (discussion-first job creation).
type PlanSessionService interface {
	Create(ctx context.Context, userID uuid.UUID, workerID uuid.UUID) (*domains.PlanSessionResponse, error)
	Get(ctx context.Context, id uuid.UUID) (*domains.PlanSessionResponse, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domains.PlanSessionResponse, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) (*domains.PlanSessionResponse, error)
	// SendMessage appends a user message and streams the agent reply via SSE. Call after setting SSE headers.
	SendMessage(ctx context.Context, w http.ResponseWriter, id uuid.UUID, content string) error
	// Generate sends the "generate plan" instruction and streams the agent reply via SSE.
	// The generated prompt is stored on the session. Call after setting SSE headers.
	Generate(ctx context.Context, w http.ResponseWriter, id uuid.UUID) error
	// Complete marks the session as completed with the confirmed generated prompt.
	Complete(ctx context.Context, id uuid.UUID) (*domains.PlanSessionResponse, error)
	// Discard soft-deletes the session.
	Discard(ctx context.Context, id uuid.UUID) error
}

type planSessionService struct {
	repo       repository.PlanSessionRepository
	workerRepo repository.WorkerRepository
	dispatcher DispatcherService
}

func NewPlanSessionService(
	repo repository.PlanSessionRepository,
	workerRepo repository.WorkerRepository,
	dispatcher DispatcherService,
) PlanSessionService {
	return &planSessionService{repo: repo, workerRepo: workerRepo, dispatcher: dispatcher}
}

func (s *planSessionService) Create(ctx context.Context, userID uuid.UUID, workerID uuid.UUID) (*domains.PlanSessionResponse, error) {
	if _, err := s.workerRepo.GetByID(ctx, workerID); err != nil {
		return nil, fmt.Errorf("worker not found")
	}
	session := &models.PlanSession{
		ID:       uuid.New(),
		UserID:   userID,
		WorkerID: workerID,
		Title:    "",
		Status:   models.PlanSessionStatusPending,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create plan session: %w", err)
	}
	return toSessionResponse(session, nil), nil
}

func (s *planSessionService) Get(ctx context.Context, id uuid.UUID) (*domains.PlanSessionResponse, error) {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan session not found")
	}
	msgs, err := s.repo.ListMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSessionResponse(session, msgs), nil
}

func (s *planSessionService) List(ctx context.Context, userID uuid.UUID) ([]*domains.PlanSessionResponse, error) {
	sessions, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*domains.PlanSessionResponse, len(sessions))
	for i, sess := range sessions {
		result[i] = toSessionResponse(sess, nil)
	}
	return result, nil
}

func (s *planSessionService) UpdateTitle(ctx context.Context, id uuid.UUID, title string) (*domains.PlanSessionResponse, error) {
	if err := s.repo.UpdateTitle(ctx, id, title); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *planSessionService) SendMessage(ctx context.Context, w http.ResponseWriter, id uuid.UUID, content string) error {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("plan session not found")
	}
	if session.Status != models.PlanSessionStatusPending {
		return fmt.Errorf("session is not pending")
	}

	// Save user message.
	userMsg := &models.PlanMessage{
		ID:            uuid.New(),
		PlanSessionID: id,
		Role:          "user",
		Content:       content,
	}
	if err := s.repo.AddMessage(ctx, userMsg); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// Auto-set title from first user message if still empty.
	if session.Title == "" {
		title := content
		if len(title) > 80 {
			title = title[:80] + "…"
		}
		_ = s.repo.UpdateTitle(ctx, id, title)
	}

	resumeID := ""
	if session.ResumeID != nil {
		resumeID = *session.ResumeID
	}

	// Prepend scoping instruction so the agent stays concise and focused.
	framedContent := "You are in a task scoping conversation. Help the user define a clear, well-scoped task.\n\n" +
		"Rules:\n" +
		"- If the user's request is vague, ask ONE focused clarifying question at a time (2-4 sentences max).\n" +
		"- If the user says to start, go ahead, proceed, or gives an affirmative response to your last question — stop asking and provide the substantive answer immediately.\n" +
		"- Do not ask for confirmation you already have. Do not repeat questions already answered.\n" +
		"- You may use read-only shell commands (cat, rg, grep, find, ls, head, tail) to browse the codebase and understand context.\n" +
		"- Do NOT write or modify files, run tests, install packages, start servers, or execute any code.\n\n" +
		"User: " + content

	// Stream agent reply.
	reply, newResumeID, err := s.dispatcher.RunPlanStream(ctx, w, session.WorkerID, resumeID, framedContent)
	if err != nil {
		return err
	}

	// Persist agent reply. Use background context — the request ctx may already be
	// cancelled if the client disconnected while the CLI was still running.
	saveCtx := context.Background()
	if reply != "" {
		agentMsg := &models.PlanMessage{
			ID:            uuid.New(),
			PlanSessionID: id,
			Role:          "agent",
			Content:       reply,
		}
		_ = s.repo.AddMessage(saveCtx, agentMsg)
	}

	// Update resume_id.
	if newResumeID != "" {
		_ = s.repo.UpdateResumeID(saveCtx, id, newResumeID)
	}

	return nil
}

func (s *planSessionService) Generate(ctx context.Context, w http.ResponseWriter, id uuid.UUID) error {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("plan session not found")
	}
	if session.Status != models.PlanSessionStatusPending {
		return fmt.Errorf("session is not pending")
	}

	resumeID := ""
	if session.ResumeID != nil {
		resumeID = *session.ResumeID
	}

	instruction := "Based on our discussion, write a clear, detailed task prompt. Output ONLY the prompt text."

	reply, newResumeID, err := s.dispatcher.RunPlanStream(ctx, w, session.WorkerID, resumeID, instruction)
	if err != nil {
		return err
	}

	saveCtx := context.Background()
	if reply != "" {
		_ = s.repo.UpdateGeneratedPrompt(saveCtx, id, reply)
		agentMsg := &models.PlanMessage{
			ID:            uuid.New(),
			PlanSessionID: id,
			Role:          "agent",
			Content:       reply,
		}
		_ = s.repo.AddMessage(saveCtx, agentMsg)
	}

	if newResumeID != "" {
		_ = s.repo.UpdateResumeID(saveCtx, id, newResumeID)
	}

	return nil
}

func (s *planSessionService) Complete(ctx context.Context, id uuid.UUID) (*domains.PlanSessionResponse, error) {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan session not found")
	}
	if session.Status != models.PlanSessionStatusPending {
		return nil, fmt.Errorf("only pending sessions can be completed")
	}
	if err := s.repo.UpdateStatus(ctx, id, models.PlanSessionStatusCompleted); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *planSessionService) Discard(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("plan session not found")
	}
	return s.repo.SoftDelete(ctx, id)
}

func toSessionResponse(s *models.PlanSession, msgs []*models.PlanMessage) *domains.PlanSessionResponse {
	r := &domains.PlanSessionResponse{
		ID:              s.ID,
		WorkerID:        s.WorkerID,
		Title:           s.Title,
		Status:          s.Status,
		GeneratedPrompt: s.GeneratedPrompt,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
	if msgs != nil {
		r.Messages = make([]domains.PlanMessageResponse, len(msgs))
		for i, m := range msgs {
			r.Messages[i] = domains.PlanMessageResponse{
				ID:        m.ID,
				Role:      m.Role,
				Content:   m.Content,
				CreatedAt: m.CreatedAt,
			}
		}
	}
	return r
}
