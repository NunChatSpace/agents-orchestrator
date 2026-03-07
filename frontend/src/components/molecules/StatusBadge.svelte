<script lang="ts">
	import Badge from '../atoms/Badge.svelte';
	import type { JobStatus } from '../../types/job';

	export let status: JobStatus | string;

	type Color = 'gray' | 'blue' | 'yellow' | 'green' | 'red' | 'purple' | 'orange';

	const colorMap: Record<string, Color> = {
		draft: 'gray',
		queued: 'blue',
		assigned: 'yellow',
		busy: 'yellow',
		pending_user: 'orange',
		done: 'green',
		failed: 'red',
		cancelled: 'gray'
	};

	const labelMap: Record<string, string> = {
		draft: 'Todo',
		queued: 'Queued',
		assigned: 'In Progress',
		busy: 'In Progress',
		pending_user: 'Waiting',
		done: 'Done',
		failed: 'Failed',
		cancelled: 'Cancelled'
	};
</script>

<Badge color={colorMap[status] ?? 'gray'}>
	{labelMap[status] ?? status}
</Badge>
