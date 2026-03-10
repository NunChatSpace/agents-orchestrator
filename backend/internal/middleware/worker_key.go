package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/views"
)

const WorkerContextKey contextKey = "worker"

// RequireWorkerKey validates the X-Worker-Key header using SHA-256 hash lookup.
func RequireWorkerKey(workerRepo repository.WorkerRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Worker-Key")
			if key == "" {
				views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-Worker-Key", requestID(r))
				return
			}
			hash := sha256Hex(key)
			worker, err := workerRepo.GetByAPIKeyHash(r.Context(), hash)
			if err != nil {
				views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid worker key", requestID(r))
				return
			}
			ctx := context.WithValue(r.Context(), WorkerContextKey, worker)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func WorkerFromContext(ctx context.Context) *models.Worker {
	worker, _ := ctx.Value(WorkerContextKey).(*models.Worker)
	return worker
}
