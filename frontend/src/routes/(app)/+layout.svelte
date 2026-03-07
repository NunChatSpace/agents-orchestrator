<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import { isSignedIn } from '../../stores/auth';
	import { allJobs } from '../../stores/jobs';
	import { allWorkers } from '../../stores/workers';
	import { listJobs } from '../../lib/apis/jobs';
	import { listWorkers } from '../../lib/apis/workers';
	import Sidebar from '../../components/organisms/Sidebar.svelte';
	import JobsPanel from '../../components/organisms/JobsPanel.svelte';

	onMount(async () => {
		if (!get(isSignedIn)) {
			goto('/login');
			return;
		}
		try {
			const [jobs, workers] = await Promise.all([listJobs({}), listWorkers()]);
			allJobs.set(jobs);
			allWorkers.set(workers);
		} catch {
			// stores remain empty; WS will hydrate on next events
		}
	});
</script>

<div class="flex h-screen overflow-hidden relative z-10">
	<Sidebar />
	<JobsPanel />
	<main class="flex-1 flex flex-col overflow-hidden relative z-10">
		<slot />
	</main>
</div>
