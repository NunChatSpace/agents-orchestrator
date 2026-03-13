<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '../../components/atoms/Button.svelte';
	import WorkerBuildStatus from '../../components/molecules/WorkerBuildStatus.svelte';
	import { listWorkerBuilds, triggerBuild } from '../../lib/apis/workerBuilds';
	import { allJobs } from '../../stores/jobs';
	import { allWorkers } from '../../stores/workers';
	import { selectedGroup } from '../../stores/selectedGroup';
	import type { JobStatus } from '../../types/job';
	import type { TriggerBuildRequest, WorkerBuild } from '../../types/workerBuild';
	import type { Worker } from '../../types/worker';

	const WORKSPACES_PATH = import.meta.env.WORKSPACES_PATH as string | undefined;
	const ACTIVE_BUILD_JOB_STATUSES: JobStatus[] = ['assigned', 'busy', 'pending_user'];
	const ACTIVE_BUILD_JOB_STATUS_SET = new Set<JobStatus>(ACTIVE_BUILD_JOB_STATUSES);
	const WORKER_GROUP_TO_BUILD_TARGET: Record<string, { stack_id: string; role: string }> = {
		// Phase 12 placeholder mapping until stack selection is introduced in the plan UI.
		'fi-backend': { stack_id: 'fi-web-app', role: 'backend' },
		'fi-frontend': { stack_id: 'fi-web-app', role: 'frontend' }
	};

	let latestBuilds: Record<string, WorkerBuild | undefined> = {};
	let buildPickerOpenByWorkerId: Record<string, boolean> = {};
	let buildModeByWorkerId: Record<string, TriggerBuildRequest['mode']> = {};
	let buildLoadingByWorkerId: Record<string, boolean> = {};
	let buildErrorByWorkerId: Record<string, string | undefined> = {};
	let buildHistoryLoadedByWorkerId: Record<string, boolean> = {};

	function toHostPath(containerPath: string): string {
		if (!WORKSPACES_PATH) return containerPath;
		return containerPath.replace(/^\/workspaces/, WORKSPACES_PATH);
	}

	$: groupWorkers = $allWorkers.filter((w) => w.group_name === $selectedGroup);
	$: activeAssignedWorkerIds = Array.from(
		new Set(
			$allJobs
				.filter(
					(job) =>
						!!job.assigned_worker_id &&
						ACTIVE_BUILD_JOB_STATUS_SET.has(job.status)
				)
				.map((job) => job.assigned_worker_id as string)
		)
	).sort();
	$: activeWorkers = $allWorkers.filter((worker) => activeAssignedWorkerIds.includes(worker.worker_id));
	$: if (activeWorkers.length > 0) {
		void loadBuildsForActiveWorkers(activeWorkers);
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

	async function loadBuildsForActiveWorkers(workers: Worker[]) {
		const workersToFetch = workers.filter((worker) => !buildHistoryLoadedByWorkerId[worker.worker_id]);
		if (workersToFetch.length === 0) return;

		buildHistoryLoadedByWorkerId = {
			...buildHistoryLoadedByWorkerId,
			...Object.fromEntries(workersToFetch.map((worker) => [worker.worker_id, true]))
		};

		const results = await Promise.allSettled(
			workersToFetch.map((worker) => listWorkerBuilds(worker.worker_id))
		);

		const nextBuilds = { ...latestBuilds };
		results.forEach((result, index) => {
			const workerId = workersToFetch[index].worker_id;
			if (result.status === 'fulfilled') {
				const fetchedBuild = result.value[0];
				const currentBuild = nextBuilds[workerId];
				if (
					fetchedBuild &&
					(!currentBuild ||
						new Date(fetchedBuild.created_at).getTime() >= new Date(currentBuild.created_at).getTime())
				) {
					nextBuilds[workerId] = fetchedBuild;
				}
				return;
			}

			buildHistoryLoadedByWorkerId = {
				...buildHistoryLoadedByWorkerId,
				[workerId]: false
			};
		});
		latestBuilds = nextBuilds;
	}

	function hasActiveBuildJob(workerId: string): boolean {
		return activeAssignedWorkerIds.includes(workerId);
	}

	function getWorkerBuildTarget(worker: Worker) {
		return WORKER_GROUP_TO_BUILD_TARGET[worker.group_name];
	}

	function toggleBuildPicker(workerId: string) {
		buildErrorByWorkerId = {
			...buildErrorByWorkerId,
			[workerId]: undefined
		};
		buildModeByWorkerId = {
			...buildModeByWorkerId,
			[workerId]: buildModeByWorkerId[workerId] ?? 'fresh'
		};
		buildPickerOpenByWorkerId = {
			...buildPickerOpenByWorkerId,
			[workerId]: !buildPickerOpenByWorkerId[workerId]
		};
	}

	function closeBuildPicker(workerId: string) {
		buildPickerOpenByWorkerId = {
			...buildPickerOpenByWorkerId,
			[workerId]: false
		};
		buildErrorByWorkerId = {
			...buildErrorByWorkerId,
			[workerId]: undefined
		};
	}

	function selectBuildMode(workerId: string, mode: TriggerBuildRequest['mode']) {
		buildModeByWorkerId = {
			...buildModeByWorkerId,
			[workerId]: mode
		};
	}

	async function confirmBuild(worker: Worker) {
		const target = getWorkerBuildTarget(worker);
		if (!target) return;

		const workerId = worker.worker_id;
		buildLoadingByWorkerId = {
			...buildLoadingByWorkerId,
			[workerId]: true
		};
		buildErrorByWorkerId = {
			...buildErrorByWorkerId,
			[workerId]: undefined
		};

		try {
			const build = await triggerBuild(workerId, {
				stack_id: target.stack_id,
				role: target.role,
				mode: buildModeByWorkerId[workerId] ?? 'fresh'
			});
			latestBuilds = {
				...latestBuilds,
				[workerId]: build
			};
			buildPickerOpenByWorkerId = {
				...buildPickerOpenByWorkerId,
				[workerId]: false
			};
			buildHistoryLoadedByWorkerId = {
				...buildHistoryLoadedByWorkerId,
				[workerId]: true
			};
		} catch (error: unknown) {
			buildErrorByWorkerId = {
				...buildErrorByWorkerId,
				[workerId]: error instanceof Error ? error.message : 'Failed to trigger build'
			};
		} finally {
			buildLoadingByWorkerId = {
				...buildLoadingByWorkerId,
				[workerId]: false
			};
		}
	}
</script>

<div class="home-wrap">
<div class="page-inner">
	<div class="page-header">
		<h1 class="page-title">Agents</h1>
		<button class="btn-new" on:click={() => goto('/agents/new')}>
			<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
			New Agent
		</button>
	</div>
	{#if $allWorkers.length === 0}
		<div class="empty-state">
			<div class="empty-icon">
				<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
				</svg>
			</div>
			<p class="empty-title">No agents registered</p>
			<p class="empty-sub">Register an agent to get started.</p>
		</div>
	{:else if groupWorkers.length === 0}
		<div class="empty-state">
			<p class="empty-title">No agents in this workspace.</p>
		</div>
	{:else}
		<div class="agent-grid">
			{#each groupWorkers as worker (worker.worker_id)}
				<div class="agent-card">
					<!-- svelte-ignore a11y-click-events-have-key-events -->
					<!-- svelte-ignore a11y-no-static-element-interactions -->
					<div class="card-clickable" on:click={() => goto(`/agents/${worker.worker_id}/jobs`)}>
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
					</div>

					{#if hasActiveBuildJob(worker.worker_id)}
						{@const buildTarget = getWorkerBuildTarget(worker)}
						<div class="build-section">
							<div class="build-section-header">
								<p class="build-label">Last build</p>
								<WorkerBuildStatus build={latestBuilds[worker.worker_id]} />
							</div>

							<div class="build-actions">
								<span
									class="build-button-wrap"
									title={!buildTarget ? 'Build not supported for this worker group' : ''}
								>
									<Button
										variant="secondary"
										disabled={!buildTarget || buildLoadingByWorkerId[worker.worker_id]}
										on:click={() => toggleBuildPicker(worker.worker_id)}
									>
										{buildLoadingByWorkerId[worker.worker_id] ? 'Building…' : 'Build'}
									</Button>
								</span>
							</div>

							{#if buildPickerOpenByWorkerId[worker.worker_id]}
								<div class="build-mode-panel border border-purple-500/20 rounded-xl p-3 bg-purple-900/10">
									<p class="build-mode-title">Build mode</p>
									<div class="build-mode-options">
										<button
											type="button"
											class:selected={buildModeByWorkerId[worker.worker_id] !== 'latest'}
											class="build-mode-option"
											on:click={() => selectBuildMode(worker.worker_id, 'fresh')}
										>
											<span>Fresh</span>
											<span>Force a new build from the current workspace</span>
										</button>
										<button
											type="button"
											class:selected={buildModeByWorkerId[worker.worker_id] === 'latest'}
											class="build-mode-option"
											on:click={() => selectBuildMode(worker.worker_id, 'latest')}
										>
											<span>Latest</span>
											<span>Use the most recent ready build when one exists</span>
										</button>
									</div>

									<div class="build-mode-actions">
										<Button
											variant="ghost"
											disabled={buildLoadingByWorkerId[worker.worker_id]}
											on:click={() => closeBuildPicker(worker.worker_id)}
										>
											Cancel
										</Button>
										<Button
											variant="secondary"
											disabled={!buildTarget || buildLoadingByWorkerId[worker.worker_id]}
											on:click={() => confirmBuild(worker)}
										>
											Confirm
										</Button>
									</div>
								</div>
							{/if}

							{#if buildErrorByWorkerId[worker.worker_id]}
								<p class="build-error">{buildErrorByWorkerId[worker.worker_id]}</p>
							{/if}
						</div>
					{/if}

					<div class="card-footer">
						<button class="footer-btn" on:click={() => goto(`/agents/${worker.worker_id}/jobs`)}>
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
								<rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
							</svg>
							Jobs
						</button>
						<button class="footer-btn" on:click={() => goto(`/agents/${worker.worker_id}/settings`)}>
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="3"/>
								<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
							</svg>
							Settings
						</button>
						<a class="footer-btn vscode-btn" href="vscode://file{toHostPath(worker.workspace)}" title="Open in VSCode">
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
</div>

<style>
	.home-wrap {
		flex: 1;
		overflow-y: auto;
		padding: 28px 32px;
	}

	.page-inner {
		max-width: 1200px;
		margin: 0 auto;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 20px;
	}
	.page-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 18px;
		font-weight: 700;
		color: #f0f0ff;
		letter-spacing: -0.01em;
	}
	.btn-new {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 7px 14px;
		border-radius: 8px;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.35);
		color: #a78bfa;
		font-size: 13px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}
	.btn-new:hover {
		background: rgba(139,92,246,0.25);
		border-color: rgba(139,92,246,0.55);
		color: #c4b5fd;
	}

	.empty-state {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		color: rgba(196,181,253,0.22);
		padding-top: 80px;
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

	.card-clickable {
		cursor: pointer;
	}

	.build-section {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 0 16px 14px;
	}

	.build-section-header {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding-top: 2px;
	}

	.build-label,
	.build-mode-title {
		margin: 0;
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.35);
	}

	.build-actions {
		display: flex;
		align-items: center;
	}

	.build-button-wrap {
		display: inline-flex;
	}

	.build-mode-panel {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.build-mode-options {
		display: grid;
		gap: 10px;
	}

	.build-mode-option {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		padding: 10px 12px;
		border-radius: 10px;
		border: 1px solid rgba(139,92,246,0.16);
		background: rgba(5,5,12,0.35);
		color: rgba(240,240,255,0.88);
		font-family: inherit;
		text-align: left;
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s, color 0.15s;
	}

	.build-mode-option span:first-child {
		font-size: 13px;
		font-weight: 600;
		color: #f0f0ff;
	}

	.build-mode-option span:last-child {
		font-size: 11px;
		line-height: 1.45;
		color: rgba(196,181,253,0.5);
	}

	.build-mode-option:hover,
	.build-mode-option.selected {
		border-color: rgba(139,92,246,0.38);
		background: rgba(139,92,246,0.12);
	}

	.build-mode-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.build-error {
		margin: 0;
		font-size: 12px;
		line-height: 1.45;
		color: #fca5a5;
	}

	.card-footer {
		padding: 10px 16px;
		border-top: 1px solid rgba(139,92,246,0.08);
		display: flex;
		gap: 8px;
	}
	.footer-btn {
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
		text-decoration: none;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}
	.footer-btn:hover {
		background: rgba(139,92,246,0.1);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.35);
	}
	.vscode-btn:hover {
		background: rgba(0,122,204,0.15);
		color: #60a5fa;
		border-color: rgba(0,122,204,0.4);
	}
</style>
