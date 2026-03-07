<script lang="ts">
	import { goto } from '$app/navigation';
	import { allWorkers } from '../../stores/workers';
	import { selectedGroup } from '../../stores/selectedGroup';
	import { selectedWorker } from '../../stores/selectedWorker';
	import { upsertJob } from '../../stores/jobs';
	import { planWorker } from '../../lib/apis/workers';
	import { createJob, submitJob } from '../../lib/apis/jobs';
	import type { Worker } from '../../types/worker';

	let activeTab: 'agents' | 'plan' = 'agents';

	// Derive unique sorted groups from all workers
	$: groups = [...new Set($allWorkers.map((w) => w.group_name))].sort();

	// Auto-select first group when workers load
	$: if (groups.length > 0 && $selectedGroup === '') selectedGroup.set(groups[0]);

	// Filter workers by selected group
	$: groupWorkers = $allWorkers.filter((w) => w.group_name === $selectedGroup);

	// Plan tab state
	let planWorkerID = '';
	let taskText = '';
	let aiPrompt = '';
	let planLoading = false;
	let planError = '';
	let jobLoading = false;

	// Auto-select first worker when group changes or planWorkerID becomes stale
	$: if (
		groupWorkers.length > 0 &&
		(!planWorkerID || !groupWorkers.find((w) => w.worker_id === planWorkerID))
	) {
		planWorkerID = groupWorkers[0].worker_id;
	}

	function switchGroup(g: string) {
		selectedGroup.set(g);
		// Reset plan state when group changes
		planWorkerID = '';
		aiPrompt = '';
		planError = '';
	}

	async function handleGeneratePlan() {
		if (!planWorkerID || !taskText.trim()) return;
		planLoading = true;
		planError = '';
		aiPrompt = '';
		try {
			const result = await planWorker(planWorkerID, taskText);
			aiPrompt = result.prompt;
		} catch (e: unknown) {
			planError = e instanceof Error ? e.message : 'Plan generation failed';
		} finally {
			planLoading = false;
		}
	}

	async function handleCreateJob() {
		if (!aiPrompt.trim() || !planWorkerID) return;
		jobLoading = true;
		try {
			const job = await createJob({
				title: taskText.slice(0, 80),
				prompt: aiPrompt,
				target_group: $selectedGroup,
				manual_worker_override: planWorkerID
			});
			upsertJob(job);
			const submitted = await submitJob(job.job_id, planWorkerID);
			upsertJob(submitted);
			goto(`/jobs/${job.job_id}`);
		} catch {
			// ignore — WS will update state if backend succeeds
		} finally {
			jobLoading = false;
		}
	}

	function goToSettings(worker: Worker) {
		selectedWorker.set(worker);
		goto(`/agents/${worker.worker_id}`);
	}

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
</script>

<div class="home-wrap">
	{#if $allWorkers.length === 0}
		<!-- Empty state: no agents registered -->
		<div class="empty-state">
			<div class="empty-icon">
				<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
				</svg>
			</div>
			<p class="empty-title">No agents registered</p>
			<p class="empty-sub">Add an agent in the sidebar to get started.</p>
		</div>
	{:else}
		<!-- Workspace (group) selector -->
		<div class="group-bar">
			{#each groups as g}
				<button
					class="group-btn"
					class:active={g === $selectedGroup}
					on:click={() => switchGroup(g)}
				>{g}</button>
			{/each}
		</div>

		<!-- Tab bar -->
		<div class="tab-bar">
			<button
				class="tab"
				class:active={activeTab === 'agents'}
				on:click={() => (activeTab = 'agents')}
			>
				<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
				</svg>
				Agents
				<span class="tab-count">{groupWorkers.length}</span>
			</button>
			<button
				class="tab"
				class:active={activeTab === 'plan'}
				on:click={() => (activeTab = 'plan')}
			>
				<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
				</svg>
				Plan
			</button>
		</div>

		<!-- Tab 1: Agents grid -->
		{#if activeTab === 'agents'}
			{#if groupWorkers.length === 0}
				<div class="tab-empty">
					<p>No agents in this workspace.</p>
				</div>
			{:else}
				<div class="agent-grid">
					{#each groupWorkers as worker (worker.worker_id)}
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

		<!-- Tab 2: Plan mode -->
		{:else}
			<div class="plan-section">
				<div class="plan-intro">
					<p class="plan-desc">Describe your goal. The selected agent will generate a detailed task prompt you can review and submit as a job.</p>
				</div>

				<div class="plan-form">
					<div class="form-field">
						<label class="field-label" for="plan-worker">Agent</label>
						<select id="plan-worker" class="field-select" bind:value={planWorkerID}>
							{#each groupWorkers as w}
								<option value={w.worker_id}>{w.name}</option>
							{/each}
						</select>
					</div>

					<div class="form-field">
						<label class="field-label" for="plan-task">Goal</label>
						<textarea
							id="plan-task"
							class="field-textarea"
							bind:value={taskText}
							placeholder="Describe what you want the agent to accomplish…"
							rows="5"
						></textarea>
					</div>

					<button
						class="generate-btn"
						on:click={handleGeneratePlan}
						disabled={!taskText.trim() || !planWorkerID || planLoading}
					>
						{#if planLoading}
							<span class="btn-spinner"></span>
							Generating…
						{:else}
							<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
								<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
							</svg>
							Generate Plan
						{/if}
					</button>

					{#if planError}
						<p class="plan-error">{planError}</p>
					{/if}

					{#if aiPrompt}
						<div class="result-section">
							<div class="result-header">
								<span class="result-label">Generated Prompt</span>
								<span class="result-hint">Edit before submitting</span>
							</div>
							<textarea
								class="field-textarea result-textarea"
								bind:value={aiPrompt}
								rows="12"
							></textarea>
							<button
								class="submit-btn"
								on:click={handleCreateJob}
								disabled={!aiPrompt.trim() || jobLoading}
							>
								{#if jobLoading}
									<span class="btn-spinner"></span>
									Creating…
								{:else}
									<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
										<line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
									</svg>
									Create & Submit Job
								{/if}
							</button>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.home-wrap {
		flex: 1;
		overflow-y: auto;
		padding: 28px 32px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	/* Empty state */
	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		color: rgba(196,181,253,0.22);
	}
	.empty-icon {
		width: 52px;
		height: 52px;
		border-radius: 13px;
		background: rgba(139,92,246,0.08);
		border: 1px solid rgba(139,92,246,0.18);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		margin-bottom: 4px;
	}
	.empty-title {
		font-size: 15px;
		font-weight: 600;
		color: rgba(240,240,255,0.5);
	}
	.empty-sub { font-size: 12.5px; }

	/* Group / workspace selector */
	.group-bar {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.group-btn {
		padding: 5px 18px;
		border-radius: 20px;
		font-size: 12.5px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		background: rgba(139,92,246,0.07);
		border: 1px solid rgba(139,92,246,0.18);
		color: rgba(196,181,253,0.55);
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}
	.group-btn:hover:not(.active) {
		background: rgba(139,92,246,0.12);
		color: rgba(196,181,253,0.85);
	}
	.group-btn.active {
		background: rgba(139,92,246,0.2);
		border-color: rgba(139,92,246,0.5);
		color: #f0f0ff;
	}

	/* Tab bar */
	.tab-bar {
		display: flex;
		border-bottom: 1px solid rgba(139,92,246,0.15);
		flex-shrink: 0;
	}
	.tab {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 8px 20px;
		font-size: 12.5px;
		font-weight: 500;
		font-family: inherit;
		color: rgba(196,181,253,0.45);
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: color 0.15s, border-color 0.15s;
	}
	.tab.active {
		color: rgba(167,139,250,0.95);
		border-bottom-color: rgba(139,92,246,0.7);
	}
	.tab:hover:not(.active) { color: rgba(196,181,253,0.75); }
	.tab-count {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 16px;
		height: 16px;
		padding: 0 4px;
		border-radius: 8px;
		background: rgba(139,92,246,0.18);
		font-size: 10px;
		font-weight: 700;
		color: rgba(167,139,250,0.85);
	}

	/* Tab empty */
	.tab-empty {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 13px;
		color: rgba(196,181,253,0.25);
		font-style: italic;
	}

	/* Agent grid (same as /agents page) */
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
	.agent-card:hover { border-color: rgba(139,92,246,0.3); }
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
	.agent-meta { flex: 1; min-width: 0; }
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
	.info-row { display: flex; align-items: baseline; gap: 8px; }
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

	/* Plan mode */
	.plan-section {
		display: flex;
		flex-direction: column;
		gap: 20px;
		max-width: 660px;
	}
	.plan-intro {}
	.plan-desc {
		font-size: 13px;
		color: rgba(196,181,253,0.45);
		line-height: 1.6;
	}
	.plan-form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.form-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.field-label {
		font-size: 10.5px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(139,92,246,0.7);
	}
	.field-select {
		background: rgba(13,13,30,0.85);
		border: 1px solid rgba(139,92,246,0.22);
		border-radius: 8px;
		padding: 9px 12px;
		color: #f0f0ff;
		font-size: 13px;
		font-family: inherit;
		cursor: pointer;
	}
	.field-select:focus { outline: none; border-color: rgba(139,92,246,0.5); }
	.field-textarea {
		background: rgba(13,13,30,0.85);
		border: 1px solid rgba(139,92,246,0.22);
		border-radius: 8px;
		padding: 10px 14px;
		color: #f0f0ff;
		font-size: 13px;
		font-family: inherit;
		line-height: 1.6;
		resize: vertical;
	}
	.field-textarea::placeholder { color: rgba(196,181,253,0.25); }
	.field-textarea:focus { outline: none; border-color: rgba(139,92,246,0.5); }
	.generate-btn {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 10px 20px;
		border-radius: 8px;
		background: linear-gradient(135deg, rgba(124,58,237,0.75), rgba(109,40,217,0.65));
		border: 1px solid rgba(139,92,246,0.4);
		color: #f0f0ff;
		font-size: 13px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		align-self: flex-start;
		transition: opacity 0.15s, box-shadow 0.15s;
		box-shadow: 0 0 14px rgba(124,58,237,0.2);
	}
	.generate-btn:hover:not(:disabled) { box-shadow: 0 0 22px rgba(124,58,237,0.4); }
	.generate-btn:disabled { opacity: 0.45; cursor: not-allowed; }
	.plan-error {
		font-size: 12.5px;
		color: rgba(252,165,165,0.85);
		background: rgba(239,68,68,0.08);
		border: 1px solid rgba(239,68,68,0.2);
		border-radius: 7px;
		padding: 8px 12px;
	}
	.result-section {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding-top: 8px;
		border-top: 1px solid rgba(139,92,246,0.12);
	}
	.result-header {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.result-label {
		font-size: 10.5px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(139,92,246,0.7);
	}
	.result-hint {
		font-size: 11px;
		color: rgba(196,181,253,0.3);
		font-style: italic;
	}
	.result-textarea { min-height: 200px; }
	.submit-btn {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 10px 20px;
		border-radius: 8px;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.35);
		color: #c4b5fd;
		font-size: 13px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		align-self: flex-start;
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}
	.submit-btn:hover:not(:disabled) {
		background: rgba(139,92,246,0.25);
		border-color: rgba(139,92,246,0.55);
		color: #f0f0ff;
	}
	.submit-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* Spinner */
	.btn-spinner {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		border: 2px solid rgba(255,255,255,0.2);
		border-top-color: #fff;
		animation: spin 0.7s linear infinite;
		flex-shrink: 0;
	}
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
