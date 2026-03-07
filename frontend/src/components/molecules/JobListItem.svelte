<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { Job } from '../../types/job';

	export let job: Job;
	export let active = false;

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

<button class="job-item {active ? 'active' : ''}" on:click>
	<div class="flex items-start justify-between gap-2">
		<p class="title truncate flex-1">{job.title}</p>
		<StatusBadge status={job.status} />
	</div>
	<div class="meta">
		<span>{job.target_group}</span>
		{#if job.assigned_worker_name}
			<span>· {job.assigned_worker_name}</span>
		{/if}
		<span class="ml-auto">{relativeTime(job.updated_at)}</span>
	</div>
</button>

<style>
	.job-item {
		width: 100%;
		text-align: left;
		padding: 11px 16px;
		border-bottom: 1px solid rgba(139,92,246,0.1);
		background: transparent;
		cursor: pointer;
		transition: background 0.18s;
		position: relative;
	}
	.job-item:hover {
		background: rgba(139,92,246,0.06);
	}
	.job-item.active {
		background: rgba(139,92,246,0.1);
	}
	.job-item.active::before {
		content: '';
		position: absolute;
		left: 0; top: 0; bottom: 0;
		width: 2.5px;
		background: #8b5cf6;
		border-radius: 0 2px 2px 0;
		box-shadow: 0 0 8px #8b5cf6;
	}
	.title {
		font-size: 13px;
		font-weight: 500;
		color: #f0f0ff;
	}
	.meta {
		margin-top: 4px;
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		color: rgba(196,181,253,0.4);
	}
</style>
