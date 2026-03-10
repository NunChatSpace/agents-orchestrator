package app

import (
	"net/http"
	"os"

	"github.com/chatchawan/agent-orchestrator/internal/controllers"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/stackregistry"
	"github.com/chatchawan/agent-orchestrator/internal/ws"
	"github.com/jmoiron/sqlx"
	"go.uber.org/dig"
)

// BuildContainer assembles the full DI container and runs startup tasks.
func BuildContainer() (*dig.Container, error) {
	c := dig.New()

	provides := []interface{}{
		// Infrastructure
		NewDatabase,

		// Repositories (concrete → interface via return type)
		repository.NewUserRepo,
		repository.NewSessionRepo,
		repository.NewWorkerRepo,
		repository.NewJobRepo,
		repository.NewMessageRepo,
		repository.NewPlanSessionRepo,
		repository.NewPreviewBundleRepo,

		// File-based stack registry
		provideStackRegistry,

		// WS hub — provide both concrete *ws.Hub and services.WSBroadcaster
		ws.NewHub,
		provideWSBroadcaster,

		// Services
		services.NewAuthService,
		services.NewDispatcherService,
		services.NewSchedulerService,
		services.NewJobService,
		services.NewWorkerService,
		services.NewPlanSessionService,
		services.NewPreviewBundleService,

		// Controllers
		controllers.NewAuthController,
		controllers.NewJobController,
		controllers.NewMessageController,
		controllers.NewWorkerController,
		controllers.NewPlanSessionController,
		controllers.NewPreviewBundleController,

		// Router
		provideRouter,
	}

	for _, p := range provides {
		if err := c.Provide(p); err != nil {
			return nil, err
		}
	}

	// Wire dispatcher → job service result callback (breaks circular dependency).
	if err := c.Invoke(func(dispatcher services.DispatcherService, jobSvc services.JobService) {
		dispatcher.SetResultHandler(jobSvc)
	}); err != nil {
		return nil, err
	}

	// Wire hub fetchers so WS broadcasts include full payloads instead of stub {job_id}.
	if err := c.Invoke(func(hub *ws.Hub, jobSvc services.JobService, workerSvc services.WorkerService) {
		hub.SetFetchers(jobSvc, jobSvc, workerSvc)
	}); err != nil {
		return nil, err
	}

	// Startup: seed user
	if err := c.Invoke(func(db *sqlx.DB) error {
		return SeedUser(db)
	}); err != nil {
		return nil, err
	}

	return c, nil
}

// provideWSBroadcaster bridges *ws.Hub → services.WSBroadcaster for dig.
func provideWSBroadcaster(hub *ws.Hub) services.WSBroadcaster {
	return hub
}

// provideRouter is the dig-injectable router constructor.
func provideRouter(
	authSvc services.AuthService,
	workerRepo repository.WorkerRepository,
	hub *ws.Hub,
	authCtrl *controllers.AuthController,
	jobCtrl *controllers.JobController,
	msgCtrl *controllers.MessageController,
	workerCtrl *controllers.WorkerController,
	planCtrl *controllers.PlanSessionController,
	previewCtrl *controllers.PreviewBundleController,
) http.Handler {
	return buildRouter(authSvc, workerRepo, hub, authCtrl, jobCtrl, msgCtrl, workerCtrl, planCtrl, previewCtrl)
}

func provideStackRegistry() (*stackregistry.Registry, error) {
	return stackregistry.LoadFromDir(os.Getenv("STACK_REGISTRY_DIR"))
}
