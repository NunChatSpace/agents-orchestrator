package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/rs/zerolog/log"
)

// NginxConfigService generates preview-upstreams.conf and reloads nginx.
type NginxConfigService interface {
	Reload(ctx context.Context) error
}

type nginxConfigService struct {
	sessionRepo repository.PreviewSessionRepository
	workerRepo  repository.WorkerRepository
}

func NewNginxConfigService(
	sessionRepo repository.PreviewSessionRepository,
	workerRepo repository.WorkerRepository,
) NginxConfigService {
	return &nginxConfigService{
		sessionRepo: sessionRepo,
		workerRepo:  workerRepo,
	}
}

// Reload regenerates preview-upstreams.conf from currently running roles, then signals nginx.
func (n *nginxConfigService) Reload(ctx context.Context) error {
	roles, err := n.sessionRepo.ListRunningRoles(ctx)
	if err != nil {
		return fmt.Errorf("list running roles: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# AUTO-GENERATED - do not edit manually\n")
	sb.WriteString("# Managed by NginxConfigService. Rewritten on each preview start/stop.\n\n")

	baseDomain := previewBaseDomain()
	tld := previewTLD()

	for _, role := range roles {
		if role.Port == nil {
			continue
		}

		worker, err := n.workerRepo.GetByID(ctx, role.WorkerID)
		if err != nil {
			log.Warn().Err(err).Str("worker_id", role.WorkerID.String()).Msg("nginx reload: worker not found, skipping role")
			continue
		}

		upstreamName := fmt.Sprintf("preview_%s", sanitizeNginxName(worker.Name))
		hostHeader := fmt.Sprintf("%s.%s.%s", baseDomain, worker.Name, tld)

		sb.WriteString(fmt.Sprintf("upstream %s {\n", upstreamName))
		sb.WriteString(fmt.Sprintf("    server backend:%d;\n", *role.Port))
		sb.WriteString("}\n\n")

		sb.WriteString("server {\n")
		sb.WriteString("    listen 80;\n")
		sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", hostHeader))
		sb.WriteString("    location / {\n")
		sb.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", upstreamName))
		sb.WriteString("        proxy_http_version 1.1;\n")
		sb.WriteString("        proxy_set_header Host $host;\n")
		sb.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		sb.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		sb.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		sb.WriteString("        proxy_read_timeout 3600s;\n")
		sb.WriteString("    }\n")
		sb.WriteString("}\n\n")
	}

	confPath := nginxPreviewConfPath()
	if err := os.WriteFile(confPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write nginx conf: %w", err)
	}

	// Signal nginx inside the proxy container.
	proxyContainer := proxyContainerName()
	cmd := exec.CommandContext(ctx, "docker", "exec", proxyContainer, "nginx", "-s", "reload")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx reload: %w — %s", err, strings.TrimSpace(string(out)))
	}

	log.Info().Int("roles", len(roles)).Msg("nginx preview config reloaded")
	return nil
}

func nginxPreviewConfPath() string {
	if v := os.Getenv("NGINX_PREVIEW_CONF"); v != "" {
		return v
	}
	return "/app/infra/nginx/preview-upstreams.conf"
}

func proxyContainerName() string {
	if v := os.Getenv("PROXY_CONTAINER_NAME"); v != "" {
		return v
	}
	return "proxy"
}

// sanitizeNginxName replaces characters that are invalid in nginx upstream names.
func sanitizeNginxName(name string) string {
	var sb strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteRune(c)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}
