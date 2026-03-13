export type DeploymentPlanStatus = 'pending' | 'deploying' | 'running' | 'failed' | 'stopped';

export type DeploymentPlanBuildMode = 'fresh' | 'latest';

export type DeploymentPlanRoleBuildStatus = 'pending' | 'ready' | 'failed';

export type DeploymentPlanRoleDeployStatus = 'pending' | 'running' | 'failed';

export interface DeploymentPlanRole {
	id: string;
	role: string;
	worker_id: string;
	worker_build_id?: string;
	image_reference?: string;
	image_digest?: string;
	container_name?: string;
	build_status: DeploymentPlanRoleBuildStatus;
	deploy_status: DeploymentPlanRoleDeployStatus;
	worker_build_status?: string;
}

export interface DeploymentPlan {
	id: string;
	name: string;
	stack_id: string;
	build_mode: DeploymentPlanBuildMode;
	status: DeploymentPlanStatus;
	preview_url?: string;
	error?: string;
	roles: DeploymentPlanRole[];
	created_at: string;
	updated_at: string;
	stopped_at?: string;
}

export interface CreateDeploymentPlanRequest {
	name: string;
	stack_id: string;
	build_mode: DeploymentPlanBuildMode;
	roles: Array<{
		role: string;
		worker_id: string;
	}>;
}
