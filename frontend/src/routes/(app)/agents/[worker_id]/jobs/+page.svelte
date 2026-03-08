<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { allJobs, upsertJob } from '../../../../../stores/jobs';
	import { allWorkers } from '../../../../../stores/workers';
	import { selectedWorker } from '../../../../../stores/selectedWorker';
	import { createJob } from '../../../../../lib/apis/jobs';
	import Modal from '../../../../../components/atoms/Modal.svelte';
	import NewJobForm from '../../../../../components/organisms/NewJobForm.svelte';
	import type { CreateJobPayload } from '../../../../../types/job';

	$: workerId = $page.params.worker_id;

	$: workerJobs = $allJobs
		.filter((j) => j.assigned_worker_id === workerId)
		.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());

	let showJobModal = false;
	let jobLoading = false;

	function jobStatusColor(status: string) {
		if (['assigned', 'busy', 'pending_user'].includes(status)) return '#facc15';
		if (status === 'done') return '#4ade80';
		if (['failed', 'cancelled'].includes(status)) return '#f87171';
		return '#6b7280';
	}

	function relativeTime(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		return `${Math.floor(hrs / 24)}d ago`;
	}

	async function handleJobDraft(e: CustomEvent<CreateJobPayload>) {
		jobLoading = true;
		try {
			const job = await createJob(e.detail);
			upsertJob(job);
			showJobModal = false;
			goto(`/jobs/${job.job_id}`);
		} catch { /* ignore */ } finally { jobLoading = false; }
	}
</script>

<div class="page">
	<div class="page-actions">
		<button class="new-job-btn" on:click={() => showJobModal = true}>
			<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
			New Job
		</button>
	</div>

	<div class="section">
		<div class="section-header">
			<div class="section-title">
				<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
					<rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
				</svg>
				Jobs
				{#if workerJobs.length > 0}<span class="job-count">{workerJobs.length}</span>{/if}
			</div>
		</div>

		{#if workerJobs.length === 0}
			<p class="jobs-empty">No jobs yet. Click <strong>New Job</strong> to dispatch one.</p>
		{:else}
			<div class="job-list">
				{#each workerJobs as job}
					<!-- svelte-ignore a11y-click-events-have-key-events -->
					<!-- svelte-ignore a11y-no-static-element-interactions -->
					<div class="job-row" on:click={() => goto(`/jobs/${job.job_id}`)}>
						<span class="job-dot" style="background:{jobStatusColor(job.status)}"></span>
						<span class="job-title">{job.title || job.prompt}</span>
						<span class="job-status">{job.status}</span>
						<span class="job-time">{relativeTime(job.updated_at)}</span>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

{#if showJobModal && $selectedWorker}
	<Modal on:close={() => showJobModal = false}>
		<div class="modal-header">
			<div class="modal-icon">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
				</svg>
			</div>
			<div>
				<h2 class="modal-title">New Job</h2>
				<p class="modal-sub">Dispatch a task to {$selectedWorker.name}</p>
			</div>
		</div>
		<NewJobForm
			workers={$allWorkers}
			loading={jobLoading}
			initialTargetGroup={$selectedWorker.group_name}
			initialWorkerOverride={$selectedWorker.worker_id}
			on:draft={handleJobDraft}
			on:submit={handleJobDraft}
		/>
	</Modal>
{/if}

<style>
	.page {
		padding: 24px 32px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		max-width: 760px;
		margin: 0 auto;
	}

	.page-actions { display: flex; justify-content: flex-end; }

	.new-job-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border-radius: 8px;
		background: linear-gradient(135deg, #7c3aed, #6d28d9);
		color: white;
		font-size: 12.5px;
		font-weight: 500;
		border: none;
		cursor: pointer;
		box-shadow: 0 0 16px rgba(124,58,237,0.3);
		transition: box-shadow 0.2s, transform 0.2s;
		font-family: inherit;
	}
	.new-job-btn:hover {
		box-shadow: 0 0 26px rgba(124,58,237,0.55);
		transform: translateY(-1px);
	}

	.section {
		background: rgba(13,13,30,0.6);
		border: 1px solid rgba(139,92,246,0.15);
		border-radius: 12px;
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12.5px;
		font-weight: 600;
		color: rgba(196,181,253,0.6);
		letter-spacing: 0.02em;
	}

	.job-count {
		font-size: 10px;
		font-weight: 700;
		padding: 1px 6px;
		border-radius: 10px;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.25);
		color: rgba(167,139,250,0.85);
	}

	.jobs-empty {
		font-size: 12px;
		color: rgba(196,181,253,0.28);
		font-style: italic;
		padding: 4px 0;
	}
	.jobs-empty strong { font-style: normal; color: rgba(196,181,253,0.45); }

	.job-list { display: flex; flex-direction: column; gap: 2px; }

	.job-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 10px;
		border-radius: 7px;
		cursor: pointer;
		transition: background 0.15s;
	}
	.job-row:hover { background: rgba(139,92,246,0.07); }

	.job-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }

	.job-title {
		flex: 1;
		font-size: 12.5px;
		color: rgba(196,181,253,0.75);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0;
	}

	.job-status { font-size: 10.5px; color: rgba(196,181,253,0.35); flex-shrink: 0; }
	.job-time { font-size: 10.5px; color: rgba(196,181,253,0.28); flex-shrink: 0; min-width: 50px; text-align: right; }

	.modal-header { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; padding-right: 32px; }
	.modal-icon {
		width: 36px;
		height: 36px;
		border-radius: 9px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}
	.modal-title { font-family: 'Space Grotesk', system-ui, sans-serif; font-size: 17px; font-weight: 700; letter-spacing: -0.01em; color: #f0f0ff; }
	.modal-sub { font-size: 12px; color: rgba(196,181,253,0.4); margin-top: 2px; }
</style>
