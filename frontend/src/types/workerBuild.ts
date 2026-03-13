export type WorkerBuildStatus = 'queued' | 'building' | 'ready' | 'failed';

export interface WorkerBuild {
	id: string;
	worker_id: string;
	stack_id: string;
	role: string;
	build_mode: 'fresh' | 'latest';
	status: WorkerBuildStatus;
	image_reference?: string;
	image_digest?: string;
	error_message?: string;
	build_log?: string;
	created_at: string;
	completed_at?: string;
}

export interface TriggerBuildRequest {
	stack_id: string;
	role: string;
	mode: 'fresh' | 'latest';
}
