<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getWorker, deleteWorker } from '../../../../lib/apis/workers';
	import { removeWorker } from '../../../../stores/workers';
	import { selectedWorker } from '../../../../stores/selectedWorker';
	import type { Worker } from '../../../../types/worker';

	let worker: Worker | null = null;
	let loadError = '';
	let deleting = false;
	let currentId = '';

	$: workerId = $page.params.worker_id;

	$: if (workerId && workerId !== currentId) {
		currentId = workerId;
		worker = null;
		loadError = '';
		getWorker(workerId)
			.then((w) => { worker = w; selectedWorker.set(w); })
			.catch(() => { loadError = 'Failed to load agent'; });
	}

	$: jobsActive = $page.url.pathname.endsWith('/jobs');
	$: settingsActive = $page.url.pathname.endsWith('/settings');

	async function handleDelete() {
		if (!worker || !confirm(`Remove agent "${worker.name}"? This cannot be undone.`)) return;
		deleting = true;
		try {
			await deleteWorker(workerId);
			removeWorker(workerId);
			selectedWorker.set(null);
			goto('/');
		} catch {
			deleting = false;
		}
	}

	function statusColor(status: string) {
		if (status === 'idle') return '#4ade80';
		if (status === 'busy' || status === 'pending_user') return '#facc15';
		return '#6b7280';
	}
</script>

<div class="agent-shell">
	{#if loadError}
		<p class="load-error">{loadError}</p>
	{:else if !worker}
		<p class="loading">Loading…</p>
	{:else}
		<div class="agent-header">
		<div class="header-inner">
			<a class="back-link" href="/">
				<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polyline points="15 18 9 12 15 6"/>
				</svg>
				Agents
			</a>
			<div class="header-body">
				<div class="agent-icon">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
						<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
					</svg>
				</div>
				<div class="header-meta">
					<div class="header-row">
						<h1 class="agent-name">{worker.name}</h1>
						<span class="status-pill" style="--dot:{statusColor(worker.status)}">
							<span class="status-dot"></span>{worker.status}
						</span>
					</div>
					<p class="agent-group">{worker.group_name} · {worker.workspace}</p>
				</div>
				<button class="delete-btn" on:click={handleDelete} disabled={deleting} title="Remove agent">
					{#if deleting}
						<span class="spinner"></span>
					{:else}
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
						</svg>
					{/if}
				</button>
			</div>
			<div class="sub-nav">
				<a class="sub-link" class:active={jobsActive} href="/agents/{workerId}/jobs">
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
						<rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
					</svg>
					Jobs
				</a>
				<a class="sub-link" class:active={settingsActive} href="/agents/{workerId}/settings">
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="3"/>
						<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
					</svg>
					Settings
				</a>
			</div>
		</div>
		</div>
		<div class="agent-content">
			<slot {worker} />
		</div>
	{/if}
</div>

<style>
	.agent-shell {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.loading, .load-error {
		font-size: 13px;
		color: rgba(196,181,253,0.4);
		padding: 40px;
		text-align: center;
	}
	.load-error { color: #fca5a5; }

	.agent-header {
		border-bottom: 1px solid rgba(139,92,246,0.12);
		flex-shrink: 0;
		padding: 0 32px;
	}

	.header-inner {
		max-width: 1200px;
		margin: 0 auto;
		padding-top: 20px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: rgba(196,181,253,0.38);
		text-decoration: none;
		transition: color 0.15s;
		width: fit-content;
	}
	.back-link:hover { color: rgba(196,181,253,0.7); }

	.header-body {
		display: flex;
		align-items: center;
		gap: 14px;
	}

	.agent-icon {
		width: 44px;
		height: 44px;
		border-radius: 12px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.header-meta { display: flex; flex-direction: column; gap: 3px; flex: 1; min-width: 0; }
	.header-row { display: flex; align-items: center; gap: 10px; }

	.agent-name {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0f0ff;
	}

	.agent-group {
		font-size: 11.5px;
		color: rgba(196,181,253,0.38);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.status-pill {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		font-size: 11px;
		font-weight: 500;
		padding: 3px 8px;
		border-radius: 20px;
		background: rgba(139,92,246,0.08);
		border: 1px solid rgba(139,92,246,0.18);
		color: rgba(196,181,253,0.7);
		flex-shrink: 0;
	}
	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--dot);
	}

	.delete-btn {
		width: 32px;
		height: 32px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		background: rgba(248,113,113,0.07);
		border: 1px solid rgba(248,113,113,0.2);
		color: #f87171;
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s;
		flex-shrink: 0;
	}
	.delete-btn:hover:not(:disabled) {
		background: rgba(248,113,113,0.15);
		border-color: rgba(248,113,113,0.35);
	}
	.delete-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	.spinner {
		width: 10px;
		height: 10px;
		border: 2px solid rgba(255,255,255,0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
		display: inline-block;
	}
	@keyframes spin { to { transform: rotate(360deg); } }

	.sub-nav {
		display: flex;
		gap: 2px;
	}

	.sub-link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 7px 16px;
		font-size: 12.5px;
		font-weight: 500;
		color: rgba(196,181,253,0.4);
		text-decoration: none;
		border-bottom: 2px solid transparent;
		transition: color 0.15s, border-color 0.15s;
	}
	.sub-link:hover { color: rgba(196,181,253,0.75); }
	.sub-link.active {
		color: #a78bfa;
		border-bottom-color: rgba(139,92,246,0.65);
	}

	.agent-content {
		flex: 1;
		overflow-y: auto;
	}
</style>
