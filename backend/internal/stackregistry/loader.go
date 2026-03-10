package stackregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Registry struct {
	entries map[string]Definition
}

func LoadFromDir(dir string) (*Registry, error) {
	if dir == "" {
		dir = "./stack_registry"
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read stack registry dir: %w", err)
	}

	entries := make(map[string]Definition, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read stack registry file %s: %w", name, err)
		}

		var def Definition
		if err := json.Unmarshal(raw, &def); err != nil {
			return nil, fmt.Errorf("parse stack registry file %s: %w", name, err)
		}
		if err := validateDefinition(def); err != nil {
			return nil, fmt.Errorf("validate stack registry file %s: %w", name, err)
		}
		if _, exists := entries[def.StackID]; exists {
			return nil, fmt.Errorf("duplicate stack_id %q", def.StackID)
		}
		entries[def.StackID] = def
	}

	return &Registry{entries: entries}, nil
}

func (r *Registry) Get(stackID string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	def, ok := r.entries[stackID]
	return def, ok
}

func (r *Registry) List() []Definition {
	if r == nil {
		return nil
	}
	result := make([]Definition, 0, len(r.entries))
	for _, def := range r.entries {
		result = append(result, def)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StackID < result[j].StackID
	})
	return result
}

func validateDefinition(def Definition) error {
	if strings.TrimSpace(def.StackID) == "" {
		return fmt.Errorf("stack_id is required")
	}
	if strings.TrimSpace(def.DisplayName) == "" {
		return fmt.Errorf("display_name is required")
	}
	if strings.TrimSpace(def.DeploymentTemplate) == "" {
		return fmt.Errorf("deployment_template is required")
	}
	if len(def.Roles) == 0 {
		return fmt.Errorf("at least one role is required")
	}

	seenRoles := make(map[string]struct{}, len(def.Roles))
	for _, role := range def.Roles {
		if strings.TrimSpace(role.Role) == "" {
			return fmt.Errorf("role is required")
		}
		if _, exists := seenRoles[role.Role]; exists {
			return fmt.Errorf("duplicate role %q", role.Role)
		}
		seenRoles[role.Role] = struct{}{}
		if strings.TrimSpace(role.WorkerGroup) == "" {
			return fmt.Errorf("worker_group is required for role %q", role.Role)
		}
		if strings.TrimSpace(role.ServiceName) == "" {
			return fmt.Errorf("service_name is required for role %q", role.Role)
		}
		if strings.TrimSpace(role.HealthcheckPath) == "" {
			return fmt.Errorf("healthcheck_path is required for role %q", role.Role)
		}
	}

	return nil
}
