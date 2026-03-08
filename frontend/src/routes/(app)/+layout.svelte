<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { user, isSignedIn } from '../../stores/auth';
	import { allJobs } from '../../stores/jobs';
	import { allWorkers } from '../../stores/workers';
	import { allGroups } from '../../stores/groups';
	import { selectedGroup } from '../../stores/selectedGroup';
	import { listJobs } from '../../lib/apis/jobs';
	import { listWorkers } from '../../lib/apis/workers';
	import { listGroups, createGroup } from '../../lib/apis/groups';
	import { getMe } from '../../lib/apis/auth';
	import { connectWS } from '../../stores/ws';

	onMount(async () => {
		if (!get(isSignedIn)) {
			try {
				const u = await getMe();
				user.set(u);
				connectWS();
			} catch {
				goto('/login');
				return;
			}
		}
		try {
			const [jobs, workers, groups] = await Promise.all([listJobs({}), listWorkers(), listGroups()]);
			allJobs.set(jobs);
			allWorkers.set(workers);
			allGroups.set(groups);
		} catch {
			// stores remain empty; WS will hydrate on next events
		}
	});

	// Merge groups from workers + standalone groups store, deduplicated, sorted.
	$: workerGroupNames = $allWorkers.map((w) => w.group_name);
	$: groups = [
		...[...new Set([...$allGroups.map((g) => g.name), ...workerGroupNames])].sort()
	];
	$: if (groups.length > 0 && $selectedGroup === '') selectedGroup.set(groups[0]);

	$: agentsActive = $page.url.pathname === '/' || $page.url.pathname.startsWith('/agents');
	$: plansActive = $page.url.pathname.startsWith('/plans');
	$: officeActive = $page.url.pathname.startsWith('/office');

	// Workspace dropdown state
	let dropdownOpen = false;
	let creatingWorkspace = false;
	let newWsName = '';
	let wsError = '';
	let wsLoading = false;
	let inputEl: HTMLInputElement;
	let dropdownEl: HTMLDivElement;

	function openDropdown() {
		dropdownOpen = true;
	}

	function closeDropdown() {
		dropdownOpen = false;
		cancelCreate();
	}

	function selectGroup(g: string) {
		selectedGroup.set(g);
		closeDropdown();
	}

	function startCreate() {
		creatingWorkspace = true;
		newWsName = '';
		wsError = '';
		setTimeout(() => inputEl?.focus(), 0);
	}

	function cancelCreate() {
		creatingWorkspace = false;
		newWsName = '';
		wsError = '';
	}

	async function submitCreate() {
		const name = newWsName.trim();
		if (!name) return;
		wsLoading = true;
		wsError = '';
		try {
			const group = await createGroup(name);
			allGroups.update((gs) => {
				if (gs.find((g) => g.group_id === group.group_id)) return gs;
				return [...gs, group].sort((a, b) => a.name.localeCompare(b.name));
			});
			selectedGroup.set(name);
			closeDropdown();
		} catch (e: unknown) {
			wsError = e instanceof Error ? e.message : 'Failed to create workspace';
		} finally {
			wsLoading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') closeDropdown();
		if (e.key === 'Enter' && creatingWorkspace) submitCreate();
	}

	function handleOutsideClick(e: MouseEvent) {
		if (dropdownEl && !dropdownEl.contains(e.target as Node)) closeDropdown();
	}
</script>

<svelte:window on:keydown={handleKeydown} />

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
				<a class="nav-link" class:active={officeActive} href="/office">Office</a>
			</nav>
			<div class="top-bar-right">
				<!-- svelte-ignore a11y-no-static-element-interactions -->
				<div class="ws-dropdown" bind:this={dropdownEl} on:click|stopPropagation>
					<button class="ws-trigger" on:click={openDropdown}>
						<span class="ws-label">{$selectedGroup || 'Workspace'}</span>
						<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
							<polyline points="6 9 12 15 18 9"/>
						</svg>
					</button>

					{#if dropdownOpen}
						<!-- svelte-ignore a11y-no-static-element-interactions -->
						<div class="ws-menu" on:click|stopPropagation>
							{#each groups as g}
								<button
									class="ws-option"
									class:selected={g === $selectedGroup}
									on:click={() => selectGroup(g)}
								>
									{g}
								</button>
							{/each}

							{#if groups.length > 0}
								<div class="ws-divider"></div>
							{/if}

							{#if creatingWorkspace}
								<div class="ws-create">
									<input
										bind:this={inputEl}
										bind:value={newWsName}
										class="ws-input"
										placeholder="Workspace name…"
										disabled={wsLoading}
									/>
									{#if wsError}
										<p class="ws-err">{wsError}</p>
									{/if}
									<div class="ws-create-actions">
										<button class="ws-btn-cancel" on:click={cancelCreate} disabled={wsLoading}>
											Cancel
										</button>
										<button
											class="ws-btn-submit"
											on:click={submitCreate}
											disabled={wsLoading || !newWsName.trim()}
										>
											{wsLoading ? 'Creating…' : 'Create'}
										</button>
									</div>
								</div>
							{:else}
								<button class="ws-option ws-new" on:click={startCreate}>
									<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
										<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
									</svg>
									New Workspace
								</button>
							{/if}
						</div>

						<!-- Backdrop to close on outside click -->
						<!-- svelte-ignore a11y-no-static-element-interactions -->
						<div class="ws-backdrop" on:click={closeDropdown}></div>
					{/if}
				</div>
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
	.logo-icon { font-size: 17px; line-height: 1; color: #7c3aed; }
	.logo:hover { color: #c4b5fd; }

	.top-nav {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
	}

	.top-bar-right { flex-shrink: 0; position: relative; }

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

	/* Workspace dropdown */
	.ws-dropdown { position: relative; }

	.ws-trigger {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: rgba(13,13,30,0.85);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 7px;
		padding: 5px 10px;
		color: #f0f0ff;
		font-size: 12.5px;
		font-family: inherit;
		cursor: pointer;
		max-width: 200px;
		transition: border-color 0.15s;
	}
	.ws-trigger:hover { border-color: rgba(139,92,246,0.4); }

	.ws-label {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 140px;
	}

	.ws-menu {
		position: absolute;
		top: calc(100% + 6px);
		right: 0;
		min-width: 180px;
		background: rgba(10,10,22,0.97);
		border: 1px solid rgba(139,92,246,0.22);
		border-radius: 10px;
		padding: 4px;
		z-index: 100;
		box-shadow: 0 8px 32px rgba(0,0,0,0.5);
	}

	.ws-option {
		display: flex;
		align-items: center;
		gap: 7px;
		width: 100%;
		padding: 7px 10px;
		border-radius: 6px;
		background: transparent;
		border: none;
		color: rgba(196,181,253,0.7);
		font-size: 12.5px;
		font-family: inherit;
		cursor: pointer;
		text-align: left;
		transition: background 0.12s, color 0.12s;
	}
	.ws-option:hover { background: rgba(139,92,246,0.1); color: #c4b5fd; }
	.ws-option.selected { color: #a78bfa; background: rgba(139,92,246,0.12); }

	.ws-new { color: rgba(167,139,250,0.5); }
	.ws-new:hover { color: #a78bfa; }

	.ws-divider {
		height: 1px;
		background: rgba(139,92,246,0.12);
		margin: 3px 0;
	}

	.ws-create {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.ws-input {
		width: 100%;
		background: rgba(5,5,12,0.8);
		border: 1px solid rgba(139,92,246,0.3);
		border-radius: 6px;
		padding: 6px 9px;
		color: #f0f0ff;
		font-size: 12.5px;
		font-family: inherit;
		box-sizing: border-box;
	}
	.ws-input:focus { outline: none; border-color: rgba(139,92,246,0.6); }
	.ws-input::placeholder { color: rgba(196,181,253,0.3); }

	.ws-err {
		font-size: 11px;
		color: #fca5a5;
		padding: 0 2px;
	}

	.ws-create-actions {
		display: flex;
		gap: 6px;
		justify-content: flex-end;
	}

	.ws-btn-cancel {
		padding: 4px 10px;
		border-radius: 5px;
		background: transparent;
		border: 1px solid rgba(139,92,246,0.2);
		color: rgba(196,181,253,0.5);
		font-size: 11.5px;
		font-family: inherit;
		cursor: pointer;
	}
	.ws-btn-cancel:hover { color: rgba(196,181,253,0.8); border-color: rgba(139,92,246,0.4); }

	.ws-btn-submit {
		padding: 4px 10px;
		border-radius: 5px;
		background: rgba(139,92,246,0.2);
		border: 1px solid rgba(139,92,246,0.4);
		color: #a78bfa;
		font-size: 11.5px;
		font-family: inherit;
		cursor: pointer;
		transition: background 0.12s;
	}
	.ws-btn-submit:hover:not(:disabled) { background: rgba(139,92,246,0.35); }
	.ws-btn-submit:disabled { opacity: 0.45; cursor: default; }

	.ws-backdrop {
		position: fixed;
		inset: 0;
		z-index: 99;
	}
</style>
