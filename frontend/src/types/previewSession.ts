export type PreviewRoleStatus = 'starting' | 'running' | 'stopped' | 'failed';
export type PreviewSessionStatus = 'starting' | 'running' | 'stopped' | 'failed';

export interface PreviewSessionRole {
	id: string;
	worker_id: string;
	worker_name: string;
	role: string;
	port?: number;
	status: PreviewRoleStatus;
	preview_url?: string;
	error_message?: string;
	updated_at: string;
}

export interface PreviewSession {
	id: string;
	status: PreviewSessionStatus;
	error?: string;
	roles: PreviewSessionRole[];
	created_at: string;
	stopped_at?: string;
}

export interface CreatePreviewSessionRolePayload {
	worker_id: string;
	role: string;
}

export interface CreatePreviewSessionPayload {
	roles: CreatePreviewSessionRolePayload[];
}
