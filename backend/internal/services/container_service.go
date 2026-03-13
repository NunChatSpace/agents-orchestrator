package services

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ContainerService wraps preview container lifecycle through the Docker CLI.
type ContainerService interface {
	RunContainer(ctx context.Context, opts RunContainerOptions) (containerName string, err error)
	StopContainer(ctx context.Context, name string) error
	RemoveContainer(ctx context.Context, name string) error
	IsContainerRunning(ctx context.Context, name string) (bool, error)
	GetContainerIP(ctx context.Context, name string) (string, error)
	ExecInContainer(ctx context.Context, name string, cmd []string) (string, error)
}

// RunContainerOptions describes one preview container launch.
type RunContainerOptions struct {
	Name          string
	Image         string
	Env           map[string]string
	NetworkName   string
	Labels        map[string]string
	HostPort      int
	ContainerPort int
}

type containerService struct{}

// NewContainerService constructs the Docker CLI-backed preview container service.
func NewContainerService() ContainerService {
	return &containerService{}
}

func (s *containerService) RunContainer(ctx context.Context, opts RunContainerOptions) (string, error) {
	if strings.TrimSpace(opts.Name) == "" {
		return "", fmt.Errorf("container name is required")
	}
	if strings.TrimSpace(opts.Image) == "" {
		return "", fmt.Errorf("container image is required")
	}
	if opts.HostPort <= 0 || opts.ContainerPort <= 0 {
		return "", fmt.Errorf("host and container ports must be greater than zero")
	}

	args := []string{
		"run", "-d",
		"--name", opts.Name,
		"--network", networkNameOrDefault(opts.NetworkName),
		"-p", fmt.Sprintf("%d:%d", opts.HostPort, opts.ContainerPort),
	}

	for _, key := range sortedKeys(opts.Labels) {
		args = append(args, "--label", key+"="+opts.Labels[key])
	}
	for _, key := range sortedKeys(opts.Env) {
		args = append(args, "-e", key+"="+opts.Env[key])
	}

	args = append(args, opts.Image)
	if _, err := runDockerCommand(ctx, args...); err != nil {
		return "", err
	}
	return opts.Name, nil
}

func (s *containerService) StopContainer(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	_, err := runDockerCommandAllowMissing(ctx, "stop", name)
	return err
}

func (s *containerService) RemoveContainer(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	_, err := runDockerCommandAllowMissing(ctx, "rm", "-f", name)
	return err
}

func (s *containerService) IsContainerRunning(ctx context.Context, name string) (bool, error) {
	output, err := runDockerCommandAllowMissing(ctx, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(output), "true"), nil
}

func (s *containerService) GetContainerIP(ctx context.Context, name string) (string, error) {
	output, err := runDockerCommand(ctx, "inspect", "-f", "{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *containerService) ExecInContainer(ctx context.Context, name string, cmd []string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("container name is required")
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("container exec command is required")
	}

	args := append([]string{"exec", name}, cmd...)
	return runDockerCommand(ctx, args...)
}

func networkNameOrDefault(name string) string {
	if strings.TrimSpace(name) == "" {
		return "bridge"
	}
	return strings.TrimSpace(name)
}

func sortedKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func runDockerCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, trimmed)
	}
	return trimmed, nil
}

func runDockerCommandAllowMissing(ctx context.Context, args ...string) (string, error) {
	output, err := runDockerCommand(ctx, args...)
	if err == nil {
		return output, nil
	}
	if dockerObjectMissing(err.Error()) {
		return "", nil
	}
	return "", err
}

func dockerObjectMissing(message string) bool {
	trimmed := strings.TrimSpace(message)
	return strings.Contains(trimmed, "No such container") || strings.Contains(trimmed, "No such object")
}

var _ ContainerService = (*containerService)(nil)
