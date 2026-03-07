<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getWorker, pingWorker, deleteWorker } from '../../../../lib/apis/workers';
	import { removeWorker, allWorkers } from '../../../../stores/workers';
	import { selectedWorker } from '../../../../stores/selectedWorker';
	import { allJobs, upsertJob } from '../../../../stores/jobs';
	import { createJob } from '../../../../lib/apis/jobs';
	import Modal from '../../../../components/atoms/Modal.svelte';
	import NewJobForm from '../../../../components/organisms/NewJobForm.svelte';
	import type { Worker, PingResult } from '../../../../types/worker';
	import type { CreateJobPayload } from '../../../../types/job';

	let worker: Worker | null = null;
	let loadError = '';
	let pinging = false;
	let pingResult: PingResult | null = null;
	let currentId = '';
	let deleting = false;
	let showJobModal = false;
	let jobLoading = false;

	$: workerId = $page.params.worker_id;

	$: workerJobs = $allJobs
		.filter((j) => j.assigned_worker_id === workerId)
		.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());

	$: if (workerId && workerId !== currentId) {
		currentId = workerId;
		worker = null;
		loadError = '';
		pingResult = null;
		getWorker(workerId).then((w) => { worker = w; selectedWorker.set(w); }).catch(() => { loadError = 'Failed to load agent'; });
	}

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

	async function runPing() {
		pinging = true;
		pingResult = null;
		try {
			pingResult = await pingWorker(workerId);
		} catch {
			pingResult = { ok: false, output: '', error: 'Request failed' };
		} finally {
			pinging = false;
		}
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

	function statusColor(status: string) {
		if (status === 'idle') return '#4ade80';
		if (status === 'busy' || status === 'pending_user') return '#facc15';
		return '#6b7280';
	}

	function jobStatusColor(status: string) {
		if (['assigned', 'busy', 'pending_user'].includes(status)) return '#facc15';
		if (status === 'done') return '#4ade80';
		if (['failed', 'cancelled'].includes(status)) return '#f87171';
		return '#6b7280';
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

<div class="page">
	{#if loadError}
		<p class="load-error">{loadError}</p>
	{:else if !worker}
		<p class="loading">Loading…</p>
	{:else}
		<!-- Header -->
		<div class="page-header">
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
			<button class="new-job-btn" on:click={() => showJobModal = true}>
				<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
					<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
				</svg>
				New Job
			</button>
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

		<!-- Jobs list -->
		<div class="section">
			<div class="section-header">
				<div class="section-title">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
						<rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
					</svg>
					Jobs
					{#if workerJobs.length > 0}<span class="job-count">{workerJobs.length}</span>{/if}
				</div>
			</div>

			{#if workerJobs.length === 0}
				<p class="jobs-empty">No jobs yet. Click <strong>New Job</strong> to dispatch one.</p>
			{:else}
				<div class="job-list">
					{#each workerJobs as job}
						<!-- svelte-ignore a11y-click-events-have-key-events -->
						<!-- svelte-ignore a11y-no-static-element-interactions -->
						<div class="job-row" on:click={() => goto(`/jobs/${job.job_id}`)}>
							<span class="job-dot" style="background:{jobStatusColor(job.status)}"></span>
							<span class="job-title">{job.title || job.prompt}</span>
							<span class="job-status">{job.status}</span>
							<span class="job-time">{relativeTime(job.updated_at)}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Agent details -->
		<div class="section">
			<div class="section-header">
				<div class="section-title">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="3"/>
						<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
					</svg>
					Settings
				</div>
			</div>
			<div class="info-grid">
				<div class="info-card">
					<div class="info-label">CLI Command</div>
					<div class="info-value mono">{worker.cli_command}</div>
				</div>
				<div class="info-card">
					<div class="info-label">Last Active</div>
					<div class="info-value">{worker.last_active_at ? relativeTime(worker.last_active_at) : 'Never'}</div>
				</div>
				<div class="info-card wide">
					<div class="info-label">Workspace</div>
					<div class="info-value mono">{worker.workspace}</div>
				</div>
				<div class="info-card wide">
					<div class="info-label">Git Repository</div>
					<div class="info-value mono">{worker.git_repo_url || '—'}</div>
				</div>
			</div>
		</div>

		<!-- Health Check -->
		<div class="section">
			<div class="section-header">
				<div class="section-title">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
					</svg>
					Health Check
				</div>
				<button class="ping-btn" on:click={runPing} disabled={pinging}>
					{#if pinging}
						<span class="spinner"></span> Pinging…
					{:else}
						<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
							<polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-.08-4.36"/>
						</svg>
						Run Health Check
					{/if}
				</button>
			</div>

			{#if pingResult}
				<div class="ping-result" class:ok={pingResult.ok} class:fail={!pingResult.ok}>
					<div class="ping-status">
						{#if pingResult.ok}
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
							<span class="ping-ok">CLI is reachable</span>
						{:else}
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="2.5">
								<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
							</svg>
							<span class="ping-fail">CLI unreachable</span>
						{/if}
					</div>
					{#if pingResult.output}
						<pre class="ping-output">{pingResult.output}</pre>
					{/if}
					{#if pingResult.error}
						<pre class="ping-error">{pingResult.error}</pre>
					{/if}
				</div>
			{:else}
				<p class="ping-hint">Runs <code>{worker.cli_command} --version</code> inside the agent's workspace to verify the CLI is installed and reachable.</p>
			{/if}
		</div>
	{/if}
</div>

{#if showJobModal && worker}
	<Modal on:close={() => showJobModal = false}>
		<div class="modal-header">
			<div class="modal-icon">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
				</svg>
			</div>
			<div>
				<h2 class="modal-title">New Job</h2>
				<p class="modal-sub">Dispatch a task to {worker.name}</p>
			</div>
		</div>
		<NewJobForm
			workers={$allWorkers}
			loading={jobLoading}
			initialTargetGroup={worker.group_name}
			initialWorkerOverride={worker.worker_id}
			on:draft={handleJobDraft}
			on:submit={handleJobDraft}
		/>
	</Modal>
{/if}

<style>
	.page {
		flex: 1;
		overflow-y: auto;
		padding: 28px 32px;
		display: flex;
		flex-direction: column;
		gap: 20px;
		max-width: 760px;
	}

	.loading, .load-error {
		font-size: 13px;
		color: rgba(196,181,253,0.4);
		padding: 40px;
		text-align: center;
	}
	.load-error { color: #fca5a5; }

	.page-header {
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

	.new-job-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border-radius: 8px;
		background: linear-gradient(135deg, #7c3aed, #6d28d9);
		color: white;
		font-size: 12.5px;
		font-weight: 500;
		border: none;
		cursor: pointer;
		box-shadow: 0 0 16px rgba(124,58,237,0.3);
		transition: box-shadow 0.2s, transform 0.2s;
		font-family: inherit;
		flex-shrink: 0;
	}
	.new-job-btn:hover {
		box-shadow: 0 0 26px rgba(124,58,237,0.55);
		transform: translateY(-1px);
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

	.section {
		background: rgba(13,13,30,0.6);
		border: 1px solid rgba(139,92,246,0.15);
		border-radius: 12px;
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12.5px;
		font-weight: 600;
		color: rgba(196,181,253,0.6);
		letter-spacing: 0.02em;
	}

	.job-count {
		font-size: 10px;
		font-weight: 700;
		padding: 1px 6px;
		border-radius: 10px;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.25);
		color: rgba(167,139,250,0.85);
	}

	.jobs-empty {
		font-size: 12px;
		color: rgba(196,181,253,0.28);
		font-style: italic;
		padding: 4px 0;
	}
	.jobs-empty strong { font-style: normal; color: rgba(196,181,253,0.45); }

	.job-list { display: flex; flex-direction: column; gap: 2px; }

	.job-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 10px;
		border-radius: 7px;
		cursor: pointer;
		transition: background 0.15s;
	}
	.job-row:hover { background: rgba(139,92,246,0.07); }

	.job-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }

	.job-title {
		flex: 1;
		font-size: 12.5px;
		color: rgba(196,181,253,0.75);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0;
	}

	.job-status { font-size: 10.5px; color: rgba(196,181,253,0.35); flex-shrink: 0; }
	.job-time { font-size: 10.5px; color: rgba(196,181,253,0.28); flex-shrink: 0; min-width: 50px; text-align: right; }

	.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

	.info-card {
		background: rgba(10,10,22,0.5);
		border: 1px solid rgba(139,92,246,0.1);
		border-radius: 8px;
		padding: 12px 14px;
		display: flex;
		flex-direction: column;
		gap: 5px;
	}
	.info-card.wide { grid-column: 1 / -1; }

	.info-label { font-size: 9.5px; font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: rgba(196,181,253,0.3); }
	.info-value { font-size: 12.5px; color: rgba(196,181,253,0.75); word-break: break-all; }
	.info-value.mono { font-family: 'SFMono-Regular', Consolas, monospace; font-size: 11.5px; color: #a78bfa; }

	.ping-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 13px;
		border-radius: 7px;
		background: linear-gradient(135deg, #7c3aed, #6d28d9);
		color: white;
		font-size: 11.5px;
		font-weight: 500;
		border: none;
		cursor: pointer;
		font-family: inherit;
		transition: opacity 0.2s;
	}
	.ping-btn:disabled { opacity: 0.5; cursor: not-allowed; }

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

	.ping-hint { font-size: 12px; color: rgba(196,181,253,0.3); line-height: 1.6; }
	.ping-hint code {
		font-family: 'SFMono-Regular', Consolas, monospace;
		background: rgba(139,92,246,0.1);
		padding: 1px 5px;
		border-radius: 4px;
		color: #a78bfa;
	}

	.ping-result { border-radius: 8px; padding: 14px; display: flex; flex-direction: column; gap: 10px; }
	.ping-result.ok { background: rgba(74,222,128,0.06); border: 1px solid rgba(74,222,128,0.2); }
	.ping-result.fail { background: rgba(248,113,113,0.06); border: 1px solid rgba(248,113,113,0.2); }

	.ping-status { display: flex; align-items: center; gap: 7px; font-size: 13px; font-weight: 600; }
	.ping-ok { color: #4ade80; }
	.ping-fail { color: #f87171; }

	.ping-output, .ping-error {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 11.5px;
		line-height: 1.6;
		white-space: pre-wrap;
		word-break: break-all;
		margin: 0;
	}
	.ping-output { color: rgba(196,181,253,0.65); }
	.ping-error { color: #fca5a5; }

	.modal-header { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; padding-right: 32px; }
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
	.modal-title { font-family: 'Space Grotesk', system-ui, sans-serif; font-size: 17px; font-weight: 700; letter-spacing: -0.01em; color: #f0f0ff; }
	.modal-sub { font-size: 12px; color: rgba(196,181,253,0.4); margin-top: 2px; }
</style>
