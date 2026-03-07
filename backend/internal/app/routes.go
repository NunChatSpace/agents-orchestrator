package app

import (
	"net/http"
	"os"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/controllers"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/ws"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

func buildRouter(
	authSvc services.AuthService,
	workerRepo repository.WorkerRepository,
	hub *ws.Hub,
	authCtrl *controllers.AuthController,
	jobCtrl *controllers.JobController,
	msgCtrl *controllers.MessageController,
	workerCtrl *controllers.WorkerController,
) http.Handler {
	r := mux.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)

	// Catch-all OPTIONS handler so CORS preflight works for all routes.
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods(http.MethodOptions)

	// Health check (public)
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	api := r.PathPrefix("/api/v1").Subrouter()

	// --- Public auth routes ---
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", authCtrl.Login).Methods(http.MethodPost)
	auth.HandleFunc("/logout", authCtrl.Logout).Methods(http.MethodPost)

	// --- Session-protected routes ---
	priv := api.NewRoute().Subrouter()
	priv.Use(middleware.RequireSession(authSvc))

	priv.HandleFunc("/auth/me", authCtrl.Me).Methods(http.MethodGet)

	// Jobs
	priv.HandleFunc("/jobs", jobCtrl.List).Methods(http.MethodGet)
	priv.HandleFunc("/jobs", jobCtrl.Create).Methods(http.MethodPost)
	priv.HandleFunc("/jobs/{job_id}", jobCtrl.Get).Methods(http.MethodGet)
	priv.HandleFunc("/jobs/{job_id}", jobCtrl.Patch).Methods(http.MethodPatch)
	priv.HandleFunc("/jobs/{job_id}", jobCtrl.Delete).Methods(http.MethodDelete)
	priv.HandleFunc("/jobs/{job_id}/submit", jobCtrl.Submit).Methods(http.MethodPost)
	priv.HandleFunc("/jobs/{job_id}/cancel", jobCtrl.Cancel).Methods(http.MethodPost)
	priv.HandleFunc("/jobs/{job_id}/finish", jobCtrl.Finish).Methods(http.MethodPost)
	priv.HandleFunc("/jobs/{job_id}/patch", jobCtrl.GetPatch).Methods(http.MethodGet)

	// Messages
	priv.HandleFunc("/jobs/{job_id}/messages", msgCtrl.List).Methods(http.MethodGet)
	priv.HandleFunc("/jobs/{job_id}/messages", msgCtrl.Send).Methods(http.MethodPost)

	// Workers
	priv.HandleFunc("/workers", workerCtrl.List).Methods(http.MethodGet)
	priv.HandleFunc("/workers", workerCtrl.Create).Methods(http.MethodPost)
	priv.HandleFunc("/workers/{worker_id}", workerCtrl.Get).Methods(http.MethodGet)
	priv.HandleFunc("/workers/{worker_id}", workerCtrl.Update).Methods(http.MethodPatch)
	priv.HandleFunc("/workers/{worker_id}", workerCtrl.Delete).Methods(http.MethodDelete)
	priv.HandleFunc("/workers/{worker_id}/ping", workerCtrl.Ping).Methods(http.MethodPost)
	priv.HandleFunc("/workers/{worker_id}/plan", workerCtrl.Plan).Methods(http.MethodPost)

	// WebSocket — registered at root (not under /api/v1) to match frontend expectation.
	wsHandler := middleware.RequireSession(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		hub.ServeWS(w, r, user.UserID)
	}))
	r.Handle("/ws", wsHandler).Methods(http.MethodGet)

	// --- Worker callback (X-Worker-Key auth) ---
	workerRoute := api.NewRoute().Subrouter()
	workerRoute.Use(middleware.RequireWorkerKey(workerRepo))
	workerRoute.HandleFunc("/workers/reply", workerCtrl.Reply).Methods(http.MethodPost)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	origin := os.Getenv("FRONTEND_ORIGIN")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Worker-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}
