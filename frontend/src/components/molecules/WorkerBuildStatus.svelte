<script lang="ts">
	import Badge from '../atoms/Badge.svelte';
	import type { WorkerBuild, WorkerBuildStatus } from '../../types/workerBuild';

	export let build: WorkerBuild | undefined;

	type BadgeColor = 'gray' | 'yellow' | 'green' | 'red';

	const badgeColorMap: Record<WorkerBuildStatus, BadgeColor> = {
		queued: 'gray',
		building: 'yellow',
		ready: 'green',
		failed: 'red'
	};

	function relativeTime(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		return `${Math.floor(hrs / 24)}d ago`;
	}

	function truncateImageReference(imageReference?: string): string {
		if (!imageReference) return 'No image yet';
		if (imageReference.length <= 30) return imageReference;
		return `...${imageReference.slice(-30)}`;
	}
</script>

{#if build}
	<div class="build-status">
		<Badge color={badgeColorMap[build.status]}>{build.status}</Badge>
		<span class="image-ref" title={build.image_reference ?? 'No image reference yet'}>
			{truncateImageReference(build.image_reference)}
		</span>
		<span class="time-ago">{relativeTime(build.completed_at ?? build.created_at)}</span>
	</div>
{:else}
	<p class="build-empty">No builds yet</p>
{/if}

<style>
	.build-status {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
		flex-wrap: wrap;
	}

	.image-ref {
		min-width: 0;
		font-size: 12px;
		color: rgba(196,181,253,0.68);
		font-family: 'SFMono-Regular', Consolas, monospace;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.time-ago,
	.build-empty {
		font-size: 11px;
		color: rgba(196,181,253,0.4);
	}

	.build-empty {
		margin: 0;
	}
</style>
