import { writable, derived } from 'svelte/store';
import type { Job, JobStatus, TargetGroup } from '../types/job';

export const allJobs = writable<Job[]>([]);
export const statusFilter = writable<JobStatus | null>(null);
export const groupFilter = writable<TargetGroup | null>(null);

export const filteredJobs = derived(
	[allJobs, statusFilter, groupFilter],
	([$jobs, $status, $group]) =>
		$jobs
			.filter((j) => !$status || j.status === $status)
			.filter((j) => !$group || j.target_group === $group)
			.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
);

/** Upsert a job into the allJobs store (insert or replace by job_id). */
export function upsertJob(job: Job) {
	allJobs.update((jobs) => {
		const idx = jobs.findIndex((j) => j.job_id === job.job_id);
		if (idx >= 0) {
			const copy = [...jobs];
			copy[idx] = { ...copy[idx], ...job };
			return copy;
		}
		return [job, ...jobs];
	});
}

/** Remove a job from the store by job_id. */
export function removeJob(jobId: string) {
	allJobs.update((jobs) => jobs.filter((j) => j.job_id !== jobId));
}
