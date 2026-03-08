<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { isSignedIn } from '../../stores/auth';
	import { allJobs } from '../../stores/jobs';
	import { allWorkers } from '../../stores/workers';
	import { selectedGroup } from '../../stores/selectedGroup';
	import { listJobs } from '../../lib/apis/jobs';
	import { listWorkers } from '../../lib/apis/workers';

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

	$: groups = [...new Set($allWorkers.map((w) => w.group_name))].sort();
	$: if (groups.length > 0 && $selectedGroup === '') selectedGroup.set(groups[0]);

	$: agentsActive = $page.url.pathname === '/' || $page.url.pathname.startsWith('/agents');
	$: plansActive = $page.url.pathname.startsWith('/plans');
</script>

<div class="app-shell">
	<header class="top-bar">
		<div class="top-bar-inner">
			<a class="logo" href="/">
				<span class="logo-icon">◈</span>
				NEXUS
			</a>
			<nav class="top-nav">
				<a class="nav-link" class:active={agentsActive} href="/">Agents</a>
				<a class="nav-link" class:active={plansActive} href="/plans">Plans</a>
			</nav>
			<div class="top-bar-right">
				{#if groups.length > 0}
					<select class="ws-select" bind:value={$selectedGroup}>
						{#each groups as g}
							<option value={g}>{g}</option>
						{/each}
					</select>
				{/if}
			</div>
		</div>
	</header>
	<main class="main-content">
		<slot />
	</main>
</div>

<style>
	.app-shell {
		display: flex;
		flex-direction: column;
		height: 100vh;
		overflow: hidden;
		position: relative;
		z-index: 10;
	}

	.top-bar {
		height: 48px;
		background: rgba(8,8,20,0.92);
		border-bottom: 1px solid rgba(139,92,246,0.14);
		flex-shrink: 0;
		z-index: 20;
		backdrop-filter: blur(8px);
		padding: 0 24px;
	}

	.top-bar-inner {
		max-width: 1200px;
		margin: 0 auto;
		height: 100%;
		display: flex;
		align-items: center;
		gap: 20px;
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 7px;
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 14.5px;
		font-weight: 700;
		letter-spacing: 0.07em;
		color: #a78bfa;
		text-decoration: none;
		flex-shrink: 0;
		transition: color 0.15s;
	}
	.logo-icon {
		font-size: 17px;
		line-height: 1;
		color: #7c3aed;
	}
	.logo:hover { color: #c4b5fd; }

	.top-nav {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
	}

	.top-bar-right { flex-shrink: 0; }

	.ws-select {
		background: rgba(13,13,30,0.85);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 7px;
		padding: 5px 10px;
		color: #f0f0ff;
		font-size: 12.5px;
		font-family: inherit;
		cursor: pointer;
		max-width: 200px;
	}
	.ws-select:focus { outline: none; border-color: rgba(139,92,246,0.45); }

	.nav-link {
		padding: 5px 14px;
		border-radius: 7px;
		font-size: 13px;
		font-weight: 500;
		color: rgba(196,181,253,0.45);
		text-decoration: none;
		transition: color 0.15s, background 0.15s;
	}
	.nav-link:hover { color: rgba(196,181,253,0.8); background: rgba(139,92,246,0.07); }
	.nav-link.active { color: #a78bfa; background: rgba(139,92,246,0.12); }

	.main-content {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
</style>
