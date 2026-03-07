<script lang="ts">
	import { goto } from '$app/navigation';
	import { derived } from 'svelte/store';
	import { allJobs } from '../../stores/jobs';
	import { selectedWorker } from '../../stores/selectedWorker';
	import { activeJobId } from '../../stores/activeJob';
	import StatusBadge from '../molecules/StatusBadge.svelte';

	// Show todo (draft) and in-progress jobs for the selected worker
	const workerJobs = derived([allJobs, selectedWorker], ([$jobs, $worker]) => {
		if (!$worker) return [];
		return $jobs
			.filter((j) =>
				(j.target_group === $worker.group_name || j.assigned_worker_id === $worker.worker_id) &&
				['draft', 'assigned', 'busy', 'pending_user', 'done'].includes(j.status)
			)
			.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
	});

	function relativeTime(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		return `${Math.floor(hrs / 24)}d ago`;
	}
</script>

<div class="jobs-panel">
	{#if $selectedWorker}
		<div class="panel-header">
			<div class="worker-dot" class:idle={$selectedWorker.status === 'idle'} class:busy={$selectedWorker.status === 'busy' || $selectedWorker.status === 'pending_user'}></div>
			<div>
				<div class="panel-title">{$selectedWorker.name}</div>
				<div class="panel-sub">{$selectedWorker.group_name}</div>
			</div>
			<span class="job-count">{$workerJobs.length}</span>
		</div>

		<div class="job-list">
			{#each $workerJobs as job (job.job_id)}
				<!-- svelte-ignore a11y-click-events-have-key-events -->
				<!-- svelte-ignore a11y-no-static-element-interactions -->
				<div class="job-row" class:active={$activeJobId === job.job_id} on:click={() => goto(`/jobs/${job.job_id}`)}>
					<div class="job-main">
						<p class="job-title">{job.title || job.job_id.slice(0, 8)}</p>
						<div class="job-meta">
							<StatusBadge status={job.status} />
							<span class="job-time">{relativeTime(job.updated_at)}</span>
						</div>
					</div>
				</div>
			{:else}
				<p class="empty">No active jobs for this agent.</p>
			{/each}
		</div>
	{:else}
		<div class="empty-state">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
			</svg>
			<p>Select an agent to view its jobs</p>
		</div>
	{/if}
</div>

<style>
	.jobs-panel {
		display: flex;
		flex-direction: column;
		width: 16rem;
		min-width: 16rem;
		height: 100%;
		background: rgba(8,8,18,0.82);
		border-right: 1px solid rgba(139,92,246,0.15);
		overflow: hidden;
	}

	.panel-header {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 16px;
		border-bottom: 1px solid rgba(139,92,246,0.15);
		flex-shrink: 0;
	}

	.worker-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
		background: #6b7280;
	}
	.worker-dot.idle { background: #4ade80; }
	.worker-dot.busy { background: #facc15; }

	.panel-title {
		font-size: 13px;
		font-weight: 600;
		color: #f0f0ff;
	}

	.panel-sub {
		font-size: 10.5px;
		color: rgba(196,181,253,0.38);
		margin-top: 1px;
	}

	.job-count {
		margin-left: auto;
		font-size: 10px;
		background: rgba(139,92,246,0.12);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 10px;
		padding: 1px 6px;
		color: #a78bfa;
		font-weight: 600;
		flex-shrink: 0;
	}

	.job-list {
		flex: 1;
		overflow-y: auto;
	}

	.job-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		border-bottom: 1px solid rgba(139,92,246,0.08);
		cursor: pointer;
		transition: background 0.15s;
		position: relative;
	}
	.job-row:hover { background: rgba(139,92,246,0.06); }
	.job-row.active {
		background: rgba(139,92,246,0.1);
	}
	.job-row.active::before {
		content: '';
		position: absolute;
		left: 0; top: 0; bottom: 0;
		width: 2.5px;
		background: #8b5cf6;
		border-radius: 0 2px 2px 0;
	}

	.job-main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.job-title {
		font-size: 12.5px;
		font-weight: 500;
		color: #f0f0ff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.job-meta {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.job-time {
		font-size: 10.5px;
		color: rgba(196,181,253,0.3);
		margin-left: auto;
	}

	.empty {
		font-size: 12px;
		color: rgba(196,181,253,0.28);
		text-align: center;
		padding: 24px 12px;
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		color: rgba(196,181,253,0.22);
		padding: 24px;
		text-align: center;
	}

	.empty-state p {
		font-size: 12px;
	}
</style>
