export type JobStatus =
	| 'draft'
	| 'queued'
	| 'assigned'
	| 'busy'
	| 'pending_user'
	| 'done'
	| 'failed'
	| 'cancelled';

export type TargetGroup = string;

export interface Job {
	job_id: string;
	title: string;
	prompt: string;
	target_group: TargetGroup;
	assigned_worker_id?: string;
	assigned_worker_name?: string;
	manual_worker_override?: string;
	status: JobStatus;
	resume_id?: string;
	failure_reason?: string;
	message_count: number;
	created_at: string;
	updated_at: string;
}

export interface CreateJobPayload {
	title?: string;
	prompt: string;
	target_group: TargetGroup;
	manual_worker_override?: string;
}

export interface PatchJobPayload {
	title?: string;
	prompt?: string;
	target_group?: TargetGroup;
}

export const TARGET_GROUPS: TargetGroup[] = ['fi-backend', 'fi-frontend', 'ib-kha'];

export const JOB_STATUSES: JobStatus[] = [
	'draft', 'queued', 'assigned', 'busy', 'pending_user', 'done', 'failed', 'cancelled'
];

export function canCancel(status: JobStatus): boolean {
	return ['assigned', 'busy', 'pending_user'].includes(status);
}

// canFinish: user can mark job as done (active jobs or already done for navigation).
export function canFinish(status: JobStatus): boolean {
	return ['assigned', 'busy', 'pending_user', 'done'].includes(status);
}

export function canClose(status: JobStatus): boolean {
	return ['failed', 'cancelled'].includes(status);
}

export function isActive(status: JobStatus): boolean {
	return ['queued', 'assigned', 'busy', 'pending_user'].includes(status);
}
