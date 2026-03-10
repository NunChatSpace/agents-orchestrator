export type PreviewBundleStatus =
	| 'pending_build'
	| 'building'
	| 'ready_to_deploy'
	| 'deploying'
	| 'healthy'
	| 'failed'
	| 'destroyed';

export type PreviewRoleBuildStatus = 'requested' | 'ready' | 'failed';

export interface PreviewStackRole {
	role: string;
	worker_group: string;
	service_name: string;
	healthcheck_path: string;
	required_image_labels: Record<string, string>;
}

export interface PreviewStack {
	stack_id: string;
	display_name: string;
	deployment_template: string;
	roles: PreviewStackRole[];
}

export interface PreviewBundleRoleState {
	id: string;
	role: string;
	worker_group: string;
	assigned_worker_id?: string;
	build_status: PreviewRoleBuildStatus;
	latest_manifest_id?: string;
	image_reference?: string;
	image_digest?: string;
	error_message?: string;
	created_at: string;
	updated_at: string;
}

export interface PreviewBuildManifest {
	id: string;
	role: string;
	worker_id: string;
	status: PreviewRoleBuildStatus;
	image_reference: string;
	image_digest?: string;
	metadata: Record<string, string>;
	error_message?: string;
	created_at: string;
}

export interface PreviewBundle {
	id: string;
	stack_id: string;
	task_id: string;
	status: PreviewBundleStatus;
	preview_url?: string;
	created_at: string;
	updated_at: string;
	destroyed_at?: string;
	roles: PreviewBundleRoleState[];
	manifests?: PreviewBuildManifest[];
}

export interface RoleOverride {
	role: string;
	worker_id: string;
}

export interface CreatePreviewBundleRequest {
	stack_id: string;
	task_id: string;
	role_overrides?: RoleOverride[];
}
