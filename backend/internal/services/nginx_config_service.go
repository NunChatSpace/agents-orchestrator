package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
)

type NginxConfigService interface {
	WriteAndReload(ctx context.Context) error
}

type nginxConfigService struct {
	repo            repository.DeploymentPlanRepository
	containerSvc    ContainerService
	previewConfPath string
	baseDomain      string
	tld             string
}

func NewNginxConfigService(
	repo repository.DeploymentPlanRepository,
	containerSvc ContainerService,
) NginxConfigService {
	return &nginxConfigService{
		repo:            repo,
		containerSvc:    containerSvc,
		previewConfPath: previewConfPath(),
		baseDomain:      previewBaseDomain(),
		tld:             previewTLD(),
	}
}

func (s *nginxConfigService) WriteAndReload(ctx context.Context) error {
	plans, err := s.repo.ListRunning(ctx)
	if err != nil {
		return fmt.Errorf("list running deployment plans: %w", err)
	}

	details := make([]*DeploymentPlanDetail, 0, len(plans))
	for _, plan := range plans {
		if plan == nil {
			continue
		}

		loadedPlan, roles, err := s.repo.GetByID(ctx, plan.ID)
		if err != nil {
			return fmt.Errorf("load running deployment plan %s: %w", plan.ID.String(), err)
		}
		details = append(details, &DeploymentPlanDetail{Plan: loadedPlan, Roles: roles})
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].Plan.Name < details[j].Plan.Name
	})

	if err := os.MkdirAll(filepath.Dir(s.previewConfPath), 0o755); err != nil {
		return fmt.Errorf("create nginx config dir: %w", err)
	}

	content := s.renderPreviewUpstreams(details)
	if err := os.WriteFile(s.previewConfPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write nginx config: %w", err)
	}

	if _, err := s.containerSvc.ExecInContainer(ctx, "proxy", []string{"nginx", "-s", "reload"}); err != nil {
		return fmt.Errorf("reload nginx: %w", err)
	}
	return nil
}

func (s *nginxConfigService) renderPreviewUpstreams(details []*DeploymentPlanDetail) string {
	var builder strings.Builder
	builder.WriteString("# AUTO-GENERATED - do not edit manually\n")
	builder.WriteString("# Managed by DeploymentPlanService. Rewritten on each deploy/stop.\n")

	for _, detail := range details {
		if detail == nil || detail.Plan == nil {
			continue
		}

		upstreams := make(map[string]string)
		activeRoles := make([]*models.DeploymentPlanRole, 0, len(detail.Roles))
		for _, role := range detail.Roles {
			if role == nil || role.HostPort == nil || role.DeployStatus != models.DeploymentPlanRoleDeployStatusRunning {
				continue
			}

			upstream := upstreamName(detail.Plan.Name, role.Role)
			upstreams[role.Role] = upstream
			activeRoles = append(activeRoles, role)

			builder.WriteString("\n")
			builder.WriteString(fmt.Sprintf("# Plan: %s role: %s\n", detail.Plan.Name, role.Role))
			builder.WriteString(fmt.Sprintf("upstream %s {\n", upstream))
			builder.WriteString(fmt.Sprintf("    server host.docker.internal:%d;\n", *role.HostPort))
			builder.WriteString("}\n")
		}

		if len(activeRoles) == 0 {
			continue
		}

		builder.WriteString("\n")
		builder.WriteString("server {\n")
		builder.WriteString("    listen 80;\n")
		builder.WriteString(fmt.Sprintf("    server_name %s;\n", buildPreviewHostname(s.baseDomain, detail.Plan.Name, s.tld)))
		builder.WriteString("\n")

		// Pragmatic v1 assumption: the stack registry only exposes the string
		// template name "subdomain-preview", so nginx owns the template-specific
		// routing rule directly. When both backend + frontend roles exist, route
		// /api/ to backend and / to frontend. When only one role exists, route
		// every request to that single upstream.
		backendUpstream, hasBackend := upstreams["backend"]
		frontendUpstream, hasFrontend := upstreams["frontend"]

		switch {
		case hasBackend && hasFrontend:
			builder.WriteString("    location /api/ {\n")
			builder.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", backendUpstream))
			builder.WriteString("        proxy_set_header Host $host;\n")
			builder.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
			builder.WriteString("    }\n")
			builder.WriteString("\n")
			builder.WriteString("    location / {\n")
			builder.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", frontendUpstream))
			builder.WriteString("        proxy_set_header Host $host;\n")
			builder.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
			builder.WriteString("    }\n")
		default:
			target := frontendUpstream
			if target == "" {
				target = backendUpstream
			}
			builder.WriteString("    location / {\n")
			builder.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", target))
			builder.WriteString("        proxy_set_header Host $host;\n")
			builder.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			builder.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
			builder.WriteString("    }\n")
		}

		builder.WriteString("}\n")
	}

	builder.WriteString("\n")
	return builder.String()
}

func previewConfPath() string {
	if value := strings.TrimSpace(os.Getenv("NGINX_PREVIEW_CONF")); value != "" {
		return value
	}
	if _, err := os.Stat("/app/infra/nginx"); err == nil {
		return "/app/infra/nginx/preview-upstreams.conf"
	}
	return "./infra/nginx/preview-upstreams.conf"
}

func upstreamName(planName, role string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return "preview_" + replacer.Replace(planName) + "_" + replacer.Replace(role)
}
