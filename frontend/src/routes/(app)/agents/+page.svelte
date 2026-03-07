<script lang="ts">
	import { goto } from '$app/navigation';
	import { allWorkers } from '../../../stores/workers';
	import { selectedWorker } from '../../../stores/selectedWorker';
	import type { Worker } from '../../../types/worker';

	function statusColor(status: Worker['status']): string {
		if (status === 'idle') return '#4ade80';
		if (status === 'busy' || status === 'pending_user') return '#facc15';
		return '#6b7280';
	}

	function statusLabel(status: Worker['status']): string {
		if (status === 'idle') return 'Idle';
		if (status === 'busy') return 'In Progress';
		if (status === 'pending_user') return 'Waiting';
		return 'Offline';
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

	function goToSettings(worker: Worker) {
		selectedWorker.set(worker);
		goto(`/agents/${worker.worker_id}`);
	}
</script>

<div class="page">
	<div class="page-header">
		<div class="header-icon">
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
				<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
			</svg>
		</div>
		<div>
			<h1 class="page-title">All Agents</h1>
			<p class="page-sub">{$allWorkers.length} registered agent{$allWorkers.length !== 1 ? 's' : ''}</p>
		</div>
	</div>

	{#if $allWorkers.length === 0}
		<div class="empty-state">
			<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2">
				<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
			</svg>
			<p>No agents registered yet.</p>
		</div>
	{:else}
		<div class="agent-grid">
			{#each $allWorkers as worker (worker.worker_id)}
				<div class="agent-card">
					<div class="card-header">
						<div class="agent-icon">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
								<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
							</svg>
						</div>
						<div class="agent-meta">
							<div class="agent-name">{worker.name}</div>
							<div class="agent-group">{worker.group_name}</div>
						</div>
						<span class="status-pill" style="--dot:{statusColor(worker.status)}">
							<span class="status-dot"></span>
							{statusLabel(worker.status)}
						</span>
					</div>

					<div class="card-body">
						<div class="info-row">
							<span class="info-label">Workspace</span>
							<span class="info-val mono">{worker.workspace}</span>
						</div>
						<div class="info-row">
							<span class="info-label">CLI</span>
							<span class="info-val mono">{worker.cli_command}</span>
						</div>
						<div class="info-row">
							<span class="info-label">Last active</span>
							<span class="info-val">{worker.last_active_at ? relativeTime(worker.last_active_at) : 'Never'}</span>
						</div>
					</div>

					<div class="card-footer">
						<button class="settings-btn" on:click={() => goToSettings(worker)}>
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="3"/>
								<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
							</svg>
							Settings
						</button>
						<a class="vscode-btn" href="vscode://file{worker.workspace}" title="Open in VSCode">
							<svg width="13" height="13" viewBox="0 0 100 100" fill="currentColor">
								<path d="M74.9 7.3L51.4 28.8 32.6 13.5 7.1 28.2v43.6l25.5 14.7 18.8-15.3 23.5 21.5L92.9 85V15L74.9 7.3zm0 62.2L57.3 50l17.6-19.5v38.9zm-49.8 7.9V22.6l17.6 13.9L25 50l17.7 13.5-17.6 13.9z"/>
							</svg>
							VSCode
						</a>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page {
		flex: 1;
		overflow-y: auto;
		padding: 32px;
		display: flex;
		flex-direction: column;
		gap: 28px;
	}

	.page-header {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.header-icon {
		width: 48px;
		height: 48px;
		border-radius: 13px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.page-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0f0ff;
	}

	.page-sub {
		font-size: 12px;
		color: rgba(196,181,253,0.4);
		margin-top: 3px;
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: rgba(196,181,253,0.22);
	}
	.empty-state p { font-size: 13px; }

	.agent-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 16px;
	}

	.agent-card {
		background: rgba(13,13,30,0.6);
		border: 1px solid rgba(139,92,246,0.15);
		border-radius: 12px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		transition: border-color 0.18s;
	}
	.agent-card:hover {
		border-color: rgba(139,92,246,0.3);
	}

	.card-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 16px;
		border-bottom: 1px solid rgba(139,92,246,0.1);
	}

	.agent-icon {
		width: 36px;
		height: 36px;
		border-radius: 9px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.18);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.agent-meta {
		flex: 1;
		min-width: 0;
	}

	.agent-name {
		font-size: 14px;
		font-weight: 600;
		color: #f0f0ff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.agent-group {
		font-size: 11px;
		color: rgba(196,181,253,0.4);
		margin-top: 2px;
	}

	.status-pill {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		font-size: 10.5px;
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

	.card-body {
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex: 1;
	}

	.info-row {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}

	.info-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.3);
		flex-shrink: 0;
		width: 72px;
	}

	.info-val {
		font-size: 12px;
		color: rgba(196,181,253,0.7);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.info-val.mono {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 11px;
		color: #a78bfa;
	}

	.card-footer {
		padding: 10px 16px;
		border-top: 1px solid rgba(139,92,246,0.08);
		display: flex;
		gap: 8px;
	}

	.settings-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border-radius: 7px;
		background: transparent;
		border: 1px solid rgba(139,92,246,0.2);
		color: rgba(167,139,250,0.6);
		font-size: 12px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}
	.settings-btn:hover {
		background: rgba(139,92,246,0.1);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.35);
	}
	.vscode-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border-radius: 7px;
		background: transparent;
		border: 1px solid rgba(139,92,246,0.2);
		color: rgba(167,139,250,0.6);
		font-size: 12px;
		font-weight: 500;
		font-family: inherit;
		text-decoration: none;
		cursor: pointer;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}
	.vscode-btn:hover {
		background: rgba(0,122,204,0.15);
		color: #60a5fa;
		border-color: rgba(0,122,204,0.4);
	}
</style>
