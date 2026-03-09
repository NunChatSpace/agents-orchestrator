<script lang="ts">
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import NewJobForm from '../../../../components/organisms/NewJobForm.svelte';
	import { allWorkers } from '../../../../stores/workers';
	import { upsertJob } from '../../../../stores/jobs';
	import { createJob, submitJob } from '../../../../lib/apis/jobs';
	import type { CreateJobPayload } from '../../../../types/job';

	let loading = false;

	async function handleDraft(e: CustomEvent<CreateJobPayload>) {
		loading = true;
		try {
			const job = await createJob(e.detail);
			upsertJob(job);
			goto(`/jobs/${job.job_id}`);
		} catch {
			// ignore
		} finally {
			loading = false;
		}
	}

	async function handleSubmit(e: CustomEvent<CreateJobPayload>) {
		loading = true;
		try {
			const job = await createJob(e.detail);
			upsertJob(job);
			if (e.detail.manual_worker_override) {
				const queued = await submitJob(job.job_id, e.detail.manual_worker_override);
				upsertJob(queued);
			}
			goto(`/jobs/${job.job_id}`);
		} catch {
			// ignore
		} finally {
			loading = false;
		}
	}
</script>

<div class="page">
	<div class="page-inner">
		<div class="page-header">
			<div class="page-icon">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
				</svg>
			</div>
			<div>
				<h1 class="page-title">New Job</h1>
				<p class="page-sub">Dispatch a task to a worker agent</p>
			</div>
		</div>
		<div class="form-card nx-card">
			<NewJobForm workers={$allWorkers} {loading} on:draft={handleDraft} on:submit={handleSubmit} />
		</div>
	</div>
</div>

<style>
	.page {
		flex: 1;
		overflow-y: auto;
		padding: 32px;
	}

	.page-inner {
		max-width: 600px;
		margin: 0 auto;
	}

	.page-header {
		display: flex;
		align-items: center;
		gap: 14px;
		margin-bottom: 24px;
	}

	.page-icon {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.page-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0f0ff;
	}

	.page-sub {
		font-size: 12.5px;
		color: rgba(196,181,253,0.4);
		margin-top: 2px;
	}

	.form-card {
		padding: 28px;
	}
</style>
