<script lang="ts">
	import { createEventDispatcher, onDestroy, onMount } from 'svelte';
	import type { Job } from '../../types/job';
	import type { Worker } from '../../types/worker';

	export let worker: Worker;
	export let activeJob: Job | null = null;
	export let recentJobs: Job[] = [];
	export let createdJobId: string | null = null;

	const dispatch = createEventDispatcher<{
		close: void;
		newjob: void;
		viewjobs: void;
		settings: void;
	}>();

	function onKeyDown(event: KeyboardEvent) {
		if (event.key === 'Escape') dispatch('close');
	}

	onMount(() => window.addEventListener('keydown', onKeyDown));
	onDestroy(() => window.removeEventListener('keydown', onKeyDown));

	function statusClass(status: string): string {
		if (status === 'done') return 'ok';
		if (status === 'failed' || status === 'cancelled') return 'fail';
		if (status === 'pending_user') return 'wait';
		if (status === 'busy' || status === 'assigned') return 'busy';
		return 'idle';
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
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="panel-backdrop" on:click={(e) => e.target === e.currentTarget && dispatch('close')}>
	<aside class="panel nx-card">
		<header class="panel-header">
			<div class="worker-name">{worker.name}</div>
			<div class="worker-meta">
				<span>{worker.group_name}</span>
				<span class="dot">•</span>
				<span class="mono">{worker.workspace}</span>
			</div>
			<div class="status-row">
				<span class="status-pill {statusClass(worker.status)}">{worker.status}</span>
				<button class="close-btn" on:click={() => dispatch('close')} aria-label="Close panel">×</button>
			</div>
		</header>

		<section class="section">
			<div class="section-label">Active Job</div>
			{#if activeJob}
				<a class="job-item" href="/jobs/{activeJob.job_id}">
					<div class="job-title">{activeJob.title || activeJob.prompt}</div>
					<div class="job-sub">{activeJob.status}</div>
				</a>
			{:else}
				<p class="empty-text">No active job right now.</p>
			{/if}
		</section>

		<section class="section">
			<div class="section-label">Recent Jobs</div>
			{#if recentJobs.length > 0}
				<div class="recent-list">
					{#each recentJobs as job (job.job_id)}
						<a class="job-item" href="/jobs/{job.job_id}">
							<div class="job-title">{job.title || job.prompt}</div>
							<div class="job-sub">{job.status} · {relativeTime(job.updated_at)}</div>
						</a>
					{/each}
				</div>
			{:else}
				<p class="empty-text">No recent jobs.</p>
			{/if}
		</section>

		<section class="section">
			<div class="section-label">Actions</div>
			<div class="actions">
				<button class="action-btn primary" on:click={() => dispatch('newjob')}>New Job →</button>
				<button class="action-btn" on:click={() => dispatch('viewjobs')}>View All Jobs →</button>
				<button class="action-btn" on:click={() => dispatch('settings')}>Settings →</button>
			</div>
			{#if createdJobId}
				<div class="created-notice">
					Job created. <a href="/jobs/{createdJobId}">Open chat</a>
				</div>
			{/if}
		</section>
	</aside>
</div>

<style>
	.panel-backdrop {
		position: fixed;
		inset: 48px 0 0 0;
		background: rgba(0,0,0,0.26);
		backdrop-filter: blur(4px);
		display: flex;
		justify-content: flex-end;
		z-index: 70;
	}

	.panel {
		width: min(420px, 100vw);
		height: calc(100vh - 48px);
		border-radius: 0;
		border-left: 1px solid rgba(139,92,246,0.28);
		animation: slide-in 180ms ease-out;
		padding: 18px;
		overflow-y: auto;
	}

	@keyframes slide-in {
		from { transform: translateX(24px); opacity: 0; }
		to { transform: translateX(0); opacity: 1; }
	}

	.panel-header {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-bottom: 12px;
		border-bottom: 1px solid rgba(139,92,246,0.14);
	}

	.worker-name {
		font-family: 'Space Grotesk', sans-serif;
		font-size: 20px;
		font-weight: 700;
		color: #efe8ff;
	}

	.worker-meta {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		color: rgba(196,181,253,0.6);
	}

	.mono {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 10px;
	}

	.dot { opacity: 0.5; }

	.status-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 2px;
	}

	.status-pill {
		padding: 3px 8px;
		border-radius: 999px;
		font-size: 11px;
		text-transform: lowercase;
		border: 1px solid transparent;
	}

	.status-pill.ok {
		background: rgba(74,222,128,0.12);
		border-color: rgba(74,222,128,0.28);
		color: #4ade80;
	}

	.status-pill.fail {
		background: rgba(248,113,113,0.12);
		border-color: rgba(248,113,113,0.28);
		color: #f87171;
	}

	.status-pill.busy {
		background: rgba(251,146,60,0.12);
		border-color: rgba(251,146,60,0.28);
		color: #fb923c;
	}

	.status-pill.wait {
		background: rgba(96,165,250,0.12);
		border-color: rgba(96,165,250,0.28);
		color: #60a5fa;
	}

	.status-pill.idle {
		background: rgba(139,92,246,0.12);
		border-color: rgba(139,92,246,0.26);
		color: #c4b5fd;
	}

	.close-btn {
		width: 28px;
		height: 28px;
		border-radius: 8px;
		border: 1px solid rgba(139,92,246,0.24);
		background: rgba(139,92,246,0.08);
		color: #c4b5fd;
		cursor: pointer;
		font-size: 18px;
		line-height: 1;
	}

	.section {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding-top: 14px;
	}

	.section-label {
		font-size: 11px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.46);
	}

	.empty-text {
		font-size: 12px;
		color: rgba(196,181,253,0.45);
	}

	.recent-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.job-item {
		display: block;
		padding: 10px 11px;
		border-radius: 10px;
		background: rgba(13,13,30,0.7);
		border: 1px solid rgba(139,92,246,0.18);
		text-decoration: none;
	}

	.job-item:hover {
		border-color: rgba(139,92,246,0.35);
	}

	.job-title {
		font-size: 12.5px;
		color: #e9ddff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.job-sub {
		margin-top: 3px;
		font-size: 11px;
		color: rgba(196,181,253,0.55);
	}

	.actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.action-btn {
		width: 100%;
		text-align: left;
		padding: 9px 11px;
		border-radius: 9px;
		border: 1px solid rgba(139,92,246,0.22);
		background: rgba(13,13,30,0.7);
		color: rgba(196,181,253,0.88);
		font-size: 12.5px;
		cursor: pointer;
	}

	.action-btn.primary {
		background: linear-gradient(135deg, rgba(124,58,237,0.26), rgba(109,40,217,0.2));
		border-color: rgba(139,92,246,0.4);
		color: #efe8ff;
	}

	.created-notice {
		margin-top: 6px;
		font-size: 12px;
		color: #86efac;
	}

	.created-notice a {
		color: #c4b5fd;
		text-decoration: underline;
	}
</style>

