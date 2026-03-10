package stackregistry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDir(t *testing.T) {
	dir := t.TempDir()
	content := `{
	  "stack_id": "fi-webapp",
	  "display_name": "FI Webapp",
	  "deployment_template": "subdomain-preview",
	  "roles": [
	    {
	      "role": "backend",
	      "worker_group": "fi-backend",
	      "service_name": "backend",
	      "healthcheck_path": "/healthz",
	      "required_image_labels": {
	        "preview.role": "backend"
	      }
	    }
	  ]
	}`
	if err := os.WriteFile(filepath.Join(dir, "fi-webapp.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write registry file: %v", err)
	}

	registry, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	def, ok := registry.Get("fi-webapp")
	if !ok {
		t.Fatalf("expected fi-webapp stack")
	}
	if def.DisplayName != "FI Webapp" {
		t.Fatalf("unexpected display name: %q", def.DisplayName)
	}
	if len(def.Roles) != 1 || def.Roles[0].Role != "backend" {
		t.Fatalf("unexpected roles: %+v", def.Roles)
	}
}

func TestLoadFromDirRejectsDuplicateRole(t *testing.T) {
	dir := t.TempDir()
	content := `{
	  "stack_id": "bad-stack",
	  "display_name": "Bad Stack",
	  "deployment_template": "subdomain-preview",
	  "roles": [
	    {
	      "role": "backend",
	      "worker_group": "fi-backend",
	      "service_name": "backend",
	      "healthcheck_path": "/healthz",
	      "required_image_labels": {}
	    },
	    {
	      "role": "backend",
	      "worker_group": "fi-backend",
	      "service_name": "backend",
	      "healthcheck_path": "/healthz",
	      "required_image_labels": {}
	    }
	  ]
	}`
	if err := os.WriteFile(filepath.Join(dir, "bad-stack.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write registry file: %v", err)
	}

	if _, err := LoadFromDir(dir); err == nil {
		t.Fatalf("expected duplicate role validation error")
	}
}
