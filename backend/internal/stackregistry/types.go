package stackregistry

type Definition struct {
	StackID            string           `json:"stack_id"`
	DisplayName        string           `json:"display_name"`
	DeploymentTemplate string           `json:"deployment_template"`
	Roles              []RoleDefinition `json:"roles"`
}

type RoleDefinition struct {
	Role                string            `json:"role"`
	WorkerGroup         string            `json:"worker_group"`
	ServiceName         string            `json:"service_name"`
	HealthcheckPath     string            `json:"healthcheck_path"`
	RequiredImageLabels map[string]string `json:"required_image_labels"`
}
