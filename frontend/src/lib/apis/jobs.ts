import { GET, POST, PATCH, DELETE } from './client';
import type { Job, CreateJobPayload, PatchJobPayload, JobStatus, TargetGroup } from '../../types/job';

interface ListJobsParams {
	status?: JobStatus;
	target_group?: TargetGroup;
	limit?: number;
	offset?: number;
}

export const listJobs = (params?: ListJobsParams): Promise<Job[]> => {
	const q = new URLSearchParams();
	if (params?.status) q.set('status', params.status);
	if (params?.target_group) q.set('target_group', params.target_group);
	if (params?.limit) q.set('limit', String(params.limit));
	if (params?.offset) q.set('offset', String(params.offset));
	const qs = q.toString();
	return GET<Job[]>(`/jobs${qs ? '?' + qs : ''}`);
};

export const createJob = (payload: CreateJobPayload) => POST<Job>('/jobs', payload);

export const getJob = (jobId: string) => GET<Job>(`/jobs/${jobId}`);

export const patchJob = (jobId: string, payload: PatchJobPayload) =>
	PATCH<Job>(`/jobs/${jobId}`, payload);

export const submitJob = (jobId: string, workerId: string) =>
	POST<Job>(`/jobs/${jobId}/submit`, { worker_id: workerId });

export const cancelJob = (jobId: string) => POST<Job>(`/jobs/${jobId}/cancel`);

export const finishJob = (jobId: string) => POST<Job>(`/jobs/${jobId}/finish`);

export const deleteJob = (jobId: string) => DELETE<null>(`/jobs/${jobId}`);
