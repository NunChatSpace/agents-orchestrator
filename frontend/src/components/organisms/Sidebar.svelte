<script lang="ts">
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import Modal from '../atoms/Modal.svelte';
	import NewJobForm from './NewJobForm.svelte';
	import NewAgentForm from './NewAgentForm.svelte';
	import { allJobs, upsertJob } from '../../stores/jobs';
	import { allWorkers } from '../../stores/workers';
	import { selectedWorker } from '../../stores/selectedWorker';
	import type { Worker } from '../../types/worker';
	import { createJob } from '../../lib/apis/jobs';
	import type { CreateJobPayload } from '../../types/job';

	let showJobModal = false;
	let showAgentModal = false;
	let jobLoading = false;

	// Group workers by group_name (workspace)
	$: workspaces = Object.entries(
		$allWorkers.reduce((acc: Record<string, Worker[]>, w) => {
			(acc[w.group_name] ??= []).push(w);
			return acc;
		}, {})
	).sort(([a], [b]) => a.localeCompare(b));

	// Track expanded workspaces — default all open
	let expanded = new Set<string>();
	$: if ($allWorkers.length && expanded.size === 0) {
		expanded = new Set($allWorkers.map((w) => w.group_name));
	}

	function toggle(name: string) {
		const s = new Set(expanded);
		if (s.has(name)) s.delete(name); else s.add(name);
		expanded = s;
	}

	function statusColor(status: Worker['status']): string {
		if (status === 'idle') return '#4ade80';
		if (status === 'busy' || status === 'pending_user') return '#facc15';
		return '#6b7280';
	}

	function handleWorkerClick(worker: Worker) {
		selectedWorker.set(worker);
		// If the worker is busy, jump directly to the active job chat
		if (worker.status === 'busy' || worker.status === 'pending_user') {
			const jobs = get(allJobs);
			const active = jobs.find(
				(j) =>
					j.assigned_worker_id === worker.worker_id &&
					['assigned', 'busy', 'pending_user'].includes(j.status)
			);
			if (active) {
				goto(`/jobs/${active.job_id}`);
				return;
			}
		}
		goto(`/agents/${worker.worker_id}`);
	}

	async function handleJobDraft(e: CustomEvent<CreateJobPayload>) {
		jobLoading = true;
		try {
			const job = await createJob(e.detail);
			upsertJob(job);
			showJobModal = false;
			goto(`/jobs/${job.job_id}`);
		} catch { /* ignore */ } finally { jobLoading = false; }
	}
</script>

<aside class="sidebar">
	<!-- Brand header -->
	<div class="sidebar-header">
		<div class="logo-row">
			<div class="logo-icon">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.2">
					<polygon points="3 11 22 2 13 21 11 13 3 11"/>
				</svg>
			</div>
			<div>
				<div class="logo-text">NEX<span>US</span></div>
				<div class="logo-sub">Agent Orchestrator</div>
			</div>
		</div>
		<button class="new-job-btn" on:click={() => showJobModal = true}>
			<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
			New Job
		</button>
	</div>

	<!-- Workspace tree -->
	<div class="section-label ws-section-label">Workspaces</div>
	<div class="workspace-tree">
		{#each workspaces as [wsName, workers]}
			<div class="ws-group">
				<!-- svelte-ignore a11y-click-events-have-key-events -->
				<!-- svelte-ignore a11y-no-static-element-interactions -->
				<div class="ws-header" on:click={() => toggle(wsName)}>
					<svg
						class="ws-chevron"
						class:open={expanded.has(wsName)}
						width="10" height="10" viewBox="0 0 24 24"
						fill="none" stroke="currentColor" stroke-width="2.5"
					>
						<polyline points="9 18 15 12 9 6"/>
					</svg>
					<span class="ws-name">{wsName}</span>
					<span class="ws-count">{workers.length}</span>
				</div>
				{#if expanded.has(wsName)}
					<div class="ws-workers">
						{#each workers as worker}
							<!-- svelte-ignore a11y-click-events-have-key-events -->
							<!-- svelte-ignore a11y-no-static-element-interactions -->
							<div
								class="worker-item"
								class:selected={$selectedWorker?.worker_id === worker.worker_id}
								on:click={() => handleWorkerClick(worker)}
							>
								<span class="worker-dot" style="background:{statusColor(worker.status)}"></span>
								<span class="worker-name">{worker.name}</span>
							<!-- svelte-ignore a11y-click-events-have-key-events -->
							<!-- svelte-ignore a11y-no-static-element-interactions -->
							<button
								class="settings-btn"
								title="Agent settings"
								on:click|stopPropagation={() => { selectedWorker.set(worker); goto(`/agents/${worker.worker_id}`); }}
							>
								<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<circle cx="12" cy="12" r="3"/>
									<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
								</svg>
							</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{:else}
			<p class="empty">No agents yet.</p>
		{/each}
	</div>

	<!-- Footer -->
	<div class="sidebar-footer">
		<button class="all-agents-btn" on:click={() => goto('/agents')}>
			<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
				<rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
			</svg>
			All Agents
		</button>
		<button class="new-agent-btn" on:click={() => showAgentModal = true}>
			<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="8" r="4"/>
				<path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
			</svg>
			New Agent
		</button>
	</div>
</aside>

{#if showJobModal}
	<Modal on:close={() => showJobModal = false}>
		<div class="modal-header">
			<div class="modal-icon">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
				</svg>
			</div>
			<div>
				<h2 class="modal-title">New Job</h2>
				<p class="modal-sub">Dispatch a task to a worker agent</p>
			</div>
		</div>
		<NewJobForm workers={$allWorkers} loading={jobLoading} on:draft={handleJobDraft} on:submit={handleJobDraft} />
	</Modal>
{/if}

{#if showAgentModal}
	<Modal on:close={() => showAgentModal = false}>
		<div class="modal-header">
			<div class="modal-icon">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
				</svg>
			</div>
			<div>
				<h2 class="modal-title">New Agent</h2>
				<p class="modal-sub">Register a worker agent with a git repository</p>
			</div>
		</div>
		<NewAgentForm on:created={() => showAgentModal = false} on:cancel={() => showAgentModal = false} />
	</Modal>
{/if}

<style>
	.sidebar {
		display: flex;
		flex-direction: column;
		width: 18rem;
		min-width: 18rem;
		height: 100%;
		background: rgba(10,10,22,0.88);
		backdrop-filter: blur(24px);
		border-right: 1px solid rgba(139,92,246,0.18);
		position: relative;
		z-index: 10;
	}

	.sidebar-header {
		padding: 20px 16px;
		border-bottom: 1px solid rgba(139,92,246,0.18);
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.logo-row {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.logo-icon {
		width: 34px;
		height: 34px;
		border-radius: 9px;
		background: linear-gradient(135deg, #7c3aed, #4f46e5);
		box-shadow: 0 0 20px rgba(124,58,237,0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.logo-text {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 15px;
		font-weight: 700;
		letter-spacing: 0.13em;
		color: #f0f0ff;
	}
	.logo-text :global(span) { color: #a78bfa; }

	.logo-sub {
		font-size: 9.5px;
		color: rgba(196,181,253,0.38);
		letter-spacing: 0.1em;
		text-transform: uppercase;
		margin-top: 1px;
	}

	.new-job-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		width: 100%;
		padding: 9px 14px;
		border-radius: 8px;
		background: linear-gradient(135deg, #7c3aed, #6d28d9);
		color: white;
		font-size: 13px;
		font-weight: 500;
		border: none;
		cursor: pointer;
		box-shadow: 0 0 18px rgba(124,58,237,0.32);
		transition: box-shadow 0.22s, transform 0.22s;
		font-family: inherit;
	}
	.new-job-btn:hover {
		box-shadow: 0 0 28px rgba(124,58,237,0.55);
		transform: translateY(-1px);
	}

	.section-label {
		font-size: 9.5px;
		font-weight: 600;
		letter-spacing: 0.16em;
		color: rgba(196,181,253,0.38);
		text-transform: uppercase;
		display: flex;
		align-items: center;
		gap: 7px;
	}

	.ws-section-label {
		padding: 12px 16px 6px;
	}

	.workspace-tree {
		flex: 1;
		overflow-y: auto;
		padding-bottom: 8px;
	}

	.ws-group {
		border-bottom: 1px solid rgba(139,92,246,0.08);
	}

	.ws-header {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 8px 16px;
		cursor: pointer;
		user-select: none;
		transition: background 0.15s;
	}
	.ws-header:hover {
		background: rgba(139,92,246,0.06);
	}

	.ws-chevron {
		color: rgba(139,92,246,0.5);
		flex-shrink: 0;
		transition: transform 0.18s;
	}
	.ws-chevron.open {
		transform: rotate(90deg);
	}

	.ws-name {
		flex: 1;
		font-size: 12px;
		font-weight: 600;
		color: rgba(196,181,253,0.75);
		letter-spacing: 0.02em;
	}

	.ws-count {
		font-size: 10px;
		background: rgba(139,92,246,0.12);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 10px;
		padding: 1px 6px;
		color: #a78bfa;
		font-weight: 600;
	}

	.ws-workers {
		padding: 2px 0 6px 32px;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.worker-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 5px 16px 5px 0;
		border-radius: 6px;
		transition: background 0.15s;
	}
	.worker-item:hover {
		background: rgba(139,92,246,0.06);
	}
	.worker-item.selected {
		background: rgba(139,92,246,0.12);
	}
	.worker-item.selected .worker-name {
		color: #c4b5fd;
	}

	.worker-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.worker-name {
		flex: 1;
		font-size: 12px;
		color: rgba(196,181,253,0.6);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.settings-btn {
		width: 22px;
		height: 22px;
		border-radius: 5px;
		background: transparent;
		border: none;
		color: rgba(167,139,250,0.3);
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		opacity: 0;
		transition: opacity 0.15s, background 0.15s, color 0.15s;
		padding: 0;
	}
	.worker-item:hover .settings-btn,
	.worker-item.selected .settings-btn {
		opacity: 1;
	}
	.settings-btn:hover {
		background: rgba(139,92,246,0.15);
		color: #a78bfa;
	}

	.empty {
		font-size: 12px;
		color: rgba(196,181,253,0.28);
		text-align: center;
		padding: 24px 12px;
	}

	.sidebar-footer {
		padding: 12px 16px;
		border-top: 1px solid rgba(139,92,246,0.12);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.all-agents-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		width: 100%;
		padding: 7px 14px;
		border-radius: 8px;
		background: transparent;
		color: rgba(167,139,250,0.5);
		font-size: 12px;
		font-weight: 500;
		border: 1px solid rgba(139,92,246,0.15);
		cursor: pointer;
		transition: background 0.18s, color 0.18s, border-color 0.18s;
		font-family: inherit;
	}
	.all-agents-btn:hover {
		background: rgba(139,92,246,0.07);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.28);
	}

	.new-agent-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		width: 100%;
		padding: 8px 14px;
		border-radius: 8px;
		background: transparent;
		color: rgba(167,139,250,0.6);
		font-size: 12px;
		font-weight: 500;
		border: 1px solid rgba(139,92,246,0.2);
		cursor: pointer;
		transition: background 0.18s, color 0.18s, border-color 0.18s;
		font-family: inherit;
	}
	.new-agent-btn:hover {
		background: rgba(139,92,246,0.08);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.35);
	}

	.modal-header {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 20px;
		padding-right: 32px;
	}

	.modal-icon {
		width: 36px;
		height: 36px;
		border-radius: 9px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.modal-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 17px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0f0ff;
	}

	.modal-sub {
		font-size: 12px;
		color: rgba(196,181,253,0.4);
		margin-top: 2px;
	}
</style>
