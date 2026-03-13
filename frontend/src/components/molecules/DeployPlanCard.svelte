<script lang="ts">
	import { createEventDispatcher, onDestroy } from 'svelte';
	import Badge from '../atoms/Badge.svelte';
	import Button from '../atoms/Button.svelte';
	import { allWorkers } from '../../stores/workers';
	import { getWorkerBuild } from '../../lib/apis/workerBuilds';
	import type {
		DeploymentPlan,
		DeploymentPlanRoleBuildStatus,
		DeploymentPlanRoleDeployStatus,
		DeploymentPlanStatus
	} from '../../types/deploymentPlan';

	type BadgeColor = 'gray' | 'blue' | 'yellow' | 'green' | 'red' | 'purple' | 'orange';

	const planStatusColors: Record<DeploymentPlanStatus, BadgeColor> = {
		pending: 'gray',
		deploying: 'yellow',
		running: 'green',
		failed: 'red',
		stopped: 'gray'
	};

	const buildStatusColors: Record<DeploymentPlanRoleBuildStatus, BadgeColor> = {
		pending: 'gray',
		ready: 'green',
		failed: 'red'
	};

	const deployStatusColors: Record<DeploymentPlanRoleDeployStatus, BadgeColor> = {
		pending: 'gray',
		running: 'green',
		failed: 'red'
	};

	export let plan: DeploymentPlan;
	export let pendingAction: 'stop' | 'retry' | 'delete' | null = null;

	const dispatch = createEventDispatcher<{
		stop: string;
		retry: string;
		delete: string;
	}>();

	// Per-role log accordion state: roleId → expanded
	let openLogs: Record<string, boolean> = {};
	// Per-buildId log content cache
	let buildLogs: Record<string, string> = {};
	let logPollHandle: ReturnType<typeof setInterval> | null = null;

	function formatLabel(value: string): string {
		return value.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
	}

	function workerName(workerId: string): string {
		return $allWorkers.find((worker) => worker.worker_id === workerId)?.name ?? workerId;
	}

	function handleStop() {
		dispatch('stop', plan.id);
	}

	function handleRetry() {
		dispatch('retry', plan.id);
	}

	function handleDelete() {
		dispatch('delete', plan.id);
	}

	function toggleLog(roleId: string) {
		openLogs = { ...openLogs, [roleId]: !openLogs[roleId] };
		syncLogPolling();
	}

	function hasActiveBuild(buildStatus?: string): boolean {
		return buildStatus === 'queued' || buildStatus === 'building';
	}

	function syncLogPolling() {
		const needsPoll = plan.roles.some(
			(r) => openLogs[r.id] && r.worker_build_id && hasActiveBuild(r.worker_build_status)
		);
		if (needsPoll && !logPollHandle) {
			logPollHandle = setInterval(pollOpenLogs, 10000);
		} else if (!needsPoll && logPollHandle) {
			clearInterval(logPollHandle);
			logPollHandle = null;
		}
	}

	async function pollOpenLogs() {
		for (const role of plan.roles) {
			if (!openLogs[role.id] || !role.worker_build_id) continue;
			try {
				const build = await getWorkerBuild(role.worker_build_id);
				if (build.build_log != null) {
					buildLogs = { ...buildLogs, [role.worker_build_id]: build.build_log };
				}
				// Stop polling this role once build is terminal
				if (build.status === 'ready' || build.status === 'failed') {
					syncLogPolling();
				}
			} catch {
				// ignore transient errors
			}
		}
	}

	// Fetch log immediately when accordion is opened for the first time
	async function fetchLogOnce(buildId: string) {
		if (buildLogs[buildId] != null) return;
		try {
			const build = await getWorkerBuild(buildId);
			if (build.build_log != null) {
				buildLogs = { ...buildLogs, [buildId]: build.build_log };
			}
		} catch {
			// ignore
		}
	}

	$: canStop = plan.status === 'pending' || plan.status === 'deploying' || plan.status === 'running';
	$: canRetry = plan.status === 'pending' || plan.status === 'failed';
	$: canDelete = plan.status === 'stopped' || plan.status === 'failed';

	// When plan data updates (from parent poll), re-sync log polling
	$: plan, syncLogPolling();

	onDestroy(() => {
		if (logPollHandle) clearInterval(logPollHandle);
	});
</script>

<article class="deploy-card nx-card">
	<div class="card-header">
		<div>
			<h2 class="plan-name">{plan.name}</h2>
			<p class="plan-meta">
				stack: <code>{plan.stack_id}</code>
				<span class="meta-dot">•</span>
				mode: <code>{plan.build_mode}</code>
			</p>
		</div>
		<Badge color={planStatusColors[plan.status]}>{formatLabel(plan.status)}</Badge>
	</div>

	{#if plan.status === 'running' && plan.preview_url}
		<div class="preview-block">
			<p class="nx-label">Preview URL</p>
			<div class="preview-row">
				<a class="preview-link" href={plan.preview_url} target="_blank" rel="noreferrer">
					{plan.preview_url}
				</a>
				<a class="open-link" href={plan.preview_url} target="_blank" rel="noreferrer">open↗</a>
			</div>
		</div>
	{/if}

	<div class="roles-block">
		<p class="nx-label">Roles</p>
		<div class="roles-list">
			{#each plan.roles as role}
				{@const isActive = hasActiveBuild(role.worker_build_status)}
				{@const isOpen = !!openLogs[role.id]}
				{@const log = role.worker_build_id ? buildLogs[role.worker_build_id] : undefined}

				<div class="role-item">
					<!-- Role row -->
					<div class="role-row">
						<span class="role-name">{formatLabel(role.role)}</span>

						<div class="role-badges">
							{#if role.build_status === 'pending' && role.worker_build_status}
								<Badge color={role.worker_build_status === 'building' ? 'yellow' : 'gray'}>
									{formatLabel(role.worker_build_status)}
								</Badge>
							{:else}
								<Badge color={buildStatusColors[role.build_status]}>
									{formatLabel(role.build_status)}
								</Badge>
							{/if}
							<Badge color={deployStatusColors[role.deploy_status]}>
								{formatLabel(role.deploy_status)}
							</Badge>
						</div>

						<span class="role-worker">{workerName(role.worker_id)}</span>

						{#if role.worker_build_id}
							<button
								class="log-toggle"
								class:open={isOpen}
								on:click={() => {
									if (role.worker_build_id) fetchLogOnce(role.worker_build_id);
									toggleLog(role.id);
								}}
								title={isOpen ? 'Hide build log' : 'Show build log'}
							>
								{isOpen ? '▲' : '▼'} Log
								{#if isActive}<span class="pulse-dot" />{/if}
							</button>
						{/if}
					</div>

					<!-- Log accordion -->
					{#if isOpen}
						<div class="log-panel">
							{#if log == null || log === ''}
								<p class="log-empty">
									{isActive ? 'Waiting for output…' : 'No log output recorded.'}
								</p>
							{:else}
								<pre class="log-pre">{log}</pre>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</div>

	{#if plan.status === 'failed' && plan.error}
		<div class="error-box">{plan.error}</div>
	{/if}

	{#if canStop || canRetry || canDelete}
		<div class="card-actions">
			{#if canStop}
				<Button variant="danger" on:click={handleStop} disabled={pendingAction !== null}>
					{pendingAction === 'stop' ? 'Stopping…' : 'Stop'}
				</Button>
			{/if}
			{#if canRetry}
				<Button variant="secondary" on:click={handleRetry} disabled={pendingAction !== null}>
					{pendingAction === 'retry' ? 'Retrying…' : 'Retry'}
				</Button>
			{/if}
			{#if canDelete}
				<Button variant="secondary" on:click={handleDelete} disabled={pendingAction !== null}>
					{pendingAction === 'delete' ? 'Deleting…' : 'Delete'}
				</Button>
			{/if}
		</div>
	{/if}
</article>

<style>
	.deploy-card {
		display: flex;
		flex-direction: column;
		gap: 18px;
		padding: 20px;
	}

	.card-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
	}

	.plan-name {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--nx-white);
	}

	.plan-meta {
		margin-top: 6px;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
		font-size: 12.5px;
		line-height: 1.5;
		color: var(--nx-muted);
	}

	.plan-meta code {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		color: rgba(240, 240, 255, 0.82);
	}

	.meta-dot {
		color: rgba(139, 92, 246, 0.35);
	}

	.preview-block,
	.roles-block {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.preview-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.preview-link,
	.open-link {
		color: var(--nx-soft);
		text-decoration: none;
		word-break: break-all;
	}

	.preview-link:hover,
	.open-link:hover {
		text-decoration: underline;
	}

	/* Roles list */
	.roles-list {
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.role-item {
		border-bottom: 1px solid rgba(139, 92, 246, 0.12);
	}

	.role-item:last-child {
		border-bottom: none;
	}

	.role-row {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 0;
		flex-wrap: wrap;
	}

	.role-name {
		font-size: 13px;
		color: var(--nx-white);
		min-width: 64px;
	}

	.role-badges {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.role-worker {
		font-size: 12.5px;
		color: rgba(240, 240, 255, 0.6);
		flex: 1;
	}

	.log-toggle {
		display: flex;
		align-items: center;
		gap: 5px;
		background: rgba(139, 92, 246, 0.1);
		border: 1px solid rgba(139, 92, 246, 0.25);
		border-radius: 6px;
		color: rgba(240, 240, 255, 0.7);
		font-size: 11px;
		font-weight: 500;
		padding: 3px 8px;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.15s;
	}

	.log-toggle:hover,
	.log-toggle.open {
		background: rgba(139, 92, 246, 0.22);
		color: var(--nx-white);
	}

	.pulse-dot {
		display: inline-block;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #facc15;
		animation: pulse 1.2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.3; }
	}

	/* Log panel */
	.log-panel {
		background: rgba(0, 0, 0, 0.35);
		border: 1px solid rgba(139, 92, 246, 0.15);
		border-radius: 8px;
		margin-bottom: 10px;
		padding: 10px 12px;
		max-height: 280px;
		overflow-y: auto;
	}

	.log-pre {
		margin: 0;
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 11.5px;
		line-height: 1.6;
		color: rgba(220, 220, 255, 0.85);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.log-empty {
		font-size: 12px;
		color: var(--nx-muted);
		margin: 0;
	}

	/* Error box */
	.error-box {
		border-radius: 12px;
		border: 1px solid rgba(239, 68, 68, 0.28);
		background: rgba(239, 68, 68, 0.09);
		padding: 12px 14px;
		font-size: 13px;
		line-height: 1.5;
		color: #fecaca;
	}

	.card-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 10px;
	}

	@media (max-width: 720px) {
		.card-header,
		.card-actions {
			flex-direction: column;
			align-items: stretch;
		}

		.role-row {
			flex-wrap: wrap;
		}

		.card-actions :global(.btn) {
			width: 100%;
			justify-content: center;
		}
	}
</style>
