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

	function statusLabel(status: string): string {
		return status.replace('_', ' ').toUpperCase();
	}

	function statusClass(status: string): string {
		if (status === 'idle') return 'idle';
		if (status === 'busy' || status === 'assigned') return 'busy';
		if (status === 'pending_user') return 'pending_user';
		return 'offline';
	}

	function relativeTime(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const minutes = Math.floor(diff / 60000);
		if (minutes < 1) return 'JUST NOW';
		if (minutes < 60) return `${minutes}M AGO`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}H AGO`;
		return `${Math.floor(hours / 24)}D AGO`;
	}
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="panel-backdrop" on:click={(event) => event.target === event.currentTarget && dispatch('close')}>
	<aside class="panel">
		<button class="panel-close" on:click={() => dispatch('close')} aria-label="Close panel">✕</button>

		<div class="panel-name">{worker.name.toUpperCase()}</div>
		<div class="panel-group">{worker.group_name.toUpperCase()} · {worker.workspace}</div>

		<div class="panel-label">STATUS</div>
		<div class="panel-status {statusClass(worker.status)}">
			<span class="dot"></span>
			<span>{statusLabel(worker.status)}</span>
		</div>

		{#if activeJob}
			<hr class="panel-divider" />
			<div class="panel-label">ACTIVE JOB</div>
			<a class="panel-job active" href="/jobs/{activeJob.job_id}">
				<span>{activeJob.title || activeJob.prompt}</span>
				<span class="job-status-pill {activeJob.status}">{activeJob.status}</span>
			</a>
		{/if}

		<hr class="panel-divider" />
		<div class="panel-label">RECENT JOBS</div>
		<div class="panel-jobs">
			{#if recentJobs.length > 0}
				{#each recentJobs as job (job.job_id)}
					<a class="panel-job" href="/jobs/{job.job_id}">
						<span class="job-title">{job.title || job.prompt}</span>
						<span class="job-meta">
							<span>{relativeTime(job.updated_at)}</span>
							<span class="job-status-pill {job.status}">{job.status}</span>
						</span>
					</a>
				{/each}
			{:else}
				<div class="empty-state">NO JOB HISTORY AVAILABLE</div>
			{/if}
		</div>

		<div class="panel-btns">
			<button class="panel-btn primary" on:click={() => dispatch('newjob')}>⊕ NEW JOB →</button>
			<button class="panel-btn secondary" on:click={() => dispatch('viewjobs')}>VIEW ALL JOBS</button>
			<button class="panel-btn secondary" on:click={() => dispatch('settings')}>SETTINGS</button>
		</div>

		{#if createdJobId}
			<div class="created-notice">
				JOB CREATED <a href="/jobs/{createdJobId}">OPEN CHAT</a>
			</div>
		{/if}
	</aside>
</div>

<style>
	.panel-backdrop {
		position: fixed;
		inset: 48px 0 0 0;
		display: flex;
		justify-content: flex-end;
		background: rgba(0, 0, 0, 0);
		z-index: 70;
	}

	.panel {
		position: relative;
		width: min(320px, calc(100vw - 24px));
		height: fit-content;
		max-height: calc(100vh - 84px);
		margin: 20px;
		padding: 20px;
		overflow-y: auto;
		background: rgba(3, 2, 10, 0.92);
		border: 1px solid rgba(0, 240, 255, 0.3);
		clip-path: polygon(16px 0%, 100% 0%, 100% calc(100% - 16px), calc(100% - 16px) 100%, 0% 100%, 0% 16px);
		box-shadow: 0 24px 70px rgba(0, 0, 0, 0.42);
		animation: panel-in 180ms cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes panel-in {
		from {
			transform: translateX(28px);
			opacity: 0;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	.panel-close {
		position: absolute;
		top: 14px;
		right: 14px;
		border: none;
		background: none;
		color: rgba(0, 240, 255, 0.35);
		font-size: 14px;
		font-family: 'Courier New', monospace;
		cursor: pointer;
		transition: color 0.2s;
	}

	.panel-close:hover {
		color: #00f0ff;
	}

	.panel-name {
		margin-bottom: 4px;
		font-size: 14px;
		font-weight: 700;
		letter-spacing: 0.12em;
		color: #00f0ff;
		text-shadow: 0 0 10px rgba(0, 240, 255, 0.5);
	}

	.panel-group {
		margin-bottom: 14px;
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(255, 42, 109, 0.7);
		word-break: break-word;
	}

	.panel-label {
		margin-bottom: 4px;
		font-size: 9px;
		letter-spacing: 0.15em;
		text-transform: uppercase;
		color: rgba(0, 240, 255, 0.35);
	}

	.panel-status {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		margin-bottom: 10px;
		padding: 3px 10px;
		border: 1px solid;
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.panel-status .dot {
		width: 6px;
		height: 6px;
		border-radius: 999px;
		background: currentColor;
	}

	.panel-status.idle {
		border-color: rgba(0, 255, 136, 0.4);
		color: #00ff88;
	}

	.panel-status.busy {
		border-color: rgba(255, 160, 0, 0.4);
		color: #ffa000;
	}

	.panel-status.pending_user {
		border-color: rgba(96, 165, 250, 0.4);
		color: #60a5fa;
	}

	.panel-status.offline {
		border-color: rgba(100, 100, 100, 0.4);
		color: #666666;
	}

	.panel-divider {
		margin: 14px 0;
		border: none;
		border-top: 1px solid rgba(0, 240, 255, 0.08);
	}

	.panel-jobs {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 16px;
	}

	.panel-job {
		display: flex;
		justify-content: space-between;
		gap: 10px;
		padding: 7px 10px;
		border: 1px solid rgba(0, 240, 255, 0.1);
		background: rgba(0, 240, 255, 0.03);
		color: rgba(220, 225, 255, 0.68);
		font-size: 10px;
		letter-spacing: 0.05em;
		text-decoration: none;
	}

	.panel-job.active {
		border-color: rgba(255, 42, 109, 0.25);
		background: rgba(255, 42, 109, 0.05);
	}

	.job-title {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.job-meta {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.empty-state {
		padding: 8px 10px;
		border: 1px solid rgba(0, 240, 255, 0.1);
		background: rgba(0, 240, 255, 0.03);
		font-size: 10px;
		letter-spacing: 0.08em;
		color: rgba(200, 200, 255, 0.5);
	}

	.job-status-pill {
		padding: 2px 6px;
		font-size: 9px;
		border-radius: 2px;
		text-transform: uppercase;
	}

	.job-status-pill.done {
		background: rgba(0, 255, 136, 0.15);
		color: #00ff88;
	}

	.job-status-pill.busy,
	.job-status-pill.assigned {
		background: rgba(255, 160, 0, 0.15);
		color: #ffa000;
	}

	.job-status-pill.pending_user {
		background: rgba(96, 165, 250, 0.15);
		color: #60a5fa;
	}

	.job-status-pill.queued,
	.job-status-pill.draft {
		background: rgba(0, 240, 255, 0.15);
		color: #00f0ff;
	}

	.job-status-pill.failed,
	.job-status-pill.cancelled {
		background: rgba(255, 42, 109, 0.15);
		color: #ff6a96;
	}

	.panel-btns {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.panel-btn {
		padding: 8px 14px;
		border: 1px solid;
		background: transparent;
		font-family: 'Courier New', monospace;
		font-size: 10px;
		letter-spacing: 0.15em;
		text-transform: uppercase;
		cursor: pointer;
		transition: all 0.2s;
		clip-path: polygon(8px 0%, 100% 0%, 100% calc(100% - 8px), calc(100% - 8px) 100%, 0% 100%, 0% 8px);
	}

	.panel-btn.primary {
		border-color: rgba(255, 42, 109, 0.5);
		color: #ff2a6d;
	}

	.panel-btn.primary:hover {
		background: rgba(255, 42, 109, 0.1);
		box-shadow: 0 0 12px rgba(255, 42, 109, 0.3);
		color: #ff6a96;
	}

	.panel-btn.secondary {
		border-color: rgba(0, 240, 255, 0.25);
		color: rgba(0, 240, 255, 0.5);
	}

	.panel-btn.secondary:hover {
		background: rgba(0, 240, 255, 0.05);
		border-color: rgba(0, 240, 255, 0.5);
		color: #00f0ff;
	}

	.created-notice {
		margin-top: 12px;
		font-size: 10px;
		letter-spacing: 0.1em;
		color: rgba(0, 255, 136, 0.78);
	}

	.created-notice a {
		color: #00f0ff;
		text-decoration: none;
	}

	.created-notice a:hover {
		text-decoration: underline;
	}

	@media (max-width: 700px) {
		.panel-backdrop {
			inset: 48px 0 0 0;
			align-items: flex-start;
		}

		.panel {
			width: calc(100vw - 20px);
			margin: 10px;
			max-height: calc(100vh - 68px);
		}
	}
</style>
