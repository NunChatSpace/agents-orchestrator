<script lang="ts">
	import StatusBadge from '../molecules/StatusBadge.svelte';
	import Button from '../atoms/Button.svelte';
	import { createEventDispatcher } from 'svelte';
	import type { Job } from '../../types/job';
	import { canCancel, canClose, canFinish } from '../../types/job';

	export let job: Job;

	const dispatch = createEventDispatcher<{ cancel: void; close: void; finish: void }>();
</script>

<div class="topbar">
	<!-- Job info -->
	<div class="job-info">
		<p class="job-title">{job.title}</p>
		<div class="job-meta">
			<span>{job.target_group}</span>
			{#if job.assigned_worker_name}
				<span class="sep">·</span>
				<span>{job.assigned_worker_name}</span>
			{/if}
		</div>
	</div>

	<StatusBadge status={job.status} />

	<div class="actions">
		{#if canFinish(job.status)}
			<Button variant="ghost" on:click={() => dispatch('finish')}>Done</Button>
		{/if}
		{#if canCancel(job.status)}
			<Button variant="danger" on:click={() => dispatch('cancel')}>Cancel</Button>
		{/if}
		{#if canClose(job.status)}
			<Button variant="ghost" on:click={() => dispatch('close')}>Close</Button>
		{/if}
	</div>
</div>

<style>
	.topbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 20px;
		min-height: 56px;
		background: rgba(5,5,12,0.8);
		backdrop-filter: blur(20px);
		border-bottom: 1px solid rgba(139,92,246,0.18);
		position: sticky;
		top: 0;
		z-index: 20;
	}

	.job-info {
		flex: 1;
		min-width: 0;
	}

	.job-title {
		font-size: 14px;
		font-weight: 600;
		color: #f0f0ff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-family: 'Space Grotesk', system-ui, sans-serif;
	}

	.job-meta {
		display: flex;
		align-items: center;
		gap: 5px;
		margin-top: 2px;
		font-size: 11.5px;
		color: rgba(196,181,253,0.4);
	}

	.sep { opacity: 0.5; }

	.actions {
		display: flex;
		gap: 8px;
		align-items: center;
	}
</style>
