<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { allWorkers } from '../../../stores/workers';
	import { selectedGroup } from '../../../stores/selectedGroup';
	import {
		listPreviewSessions,
		createPreviewSession,
		stopPreviewSession,
		deletePreviewSession,
		getRoleLog
	} from '../../../lib/apis/previewSessions';
	import type { PreviewSession, CreatePreviewSessionRolePayload } from '../../../types/previewSession';

	$: groupWorkers = $allWorkers.filter(
		(w) => w.group_name === $selectedGroup && !!w.preview_command
	);

	let sessions: PreviewSession[] = [];
	let loading = false;
	let error = '';

	// New session form state
	let showForm = false;
	let formRoles: Array<{ worker_id: string; role: string }> = [];
	let creating = false;
	let createError = '';

	// Log viewer state
	let logSessionId: string | null = null;
	let logRoleId: string | null = null;
	let logRoleName = '';
	let logContent = '';
	let logLoading = false;
	let logPollTimer: ReturnType<typeof setInterval> | null = null;

	onMount(async () => {
		await loadSessions();
	});

	onDestroy(() => {
		stopLogPoll();
	});

	async function loadSessions() {
		loading = true;
		error = '';
		try {
			sessions = await listPreviewSessions();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load preview sessions';
		} finally {
			loading = false;
		}
	}

	function openForm() {
		formRoles = groupWorkers.map((w) => ({ worker_id: w.worker_id, role: 'app' }));
		if (formRoles.length === 0) {
			formRoles = [{ worker_id: '', role: 'app' }];
		}
		createError = '';
		showForm = true;
	}

	function addRole() {
		formRoles = [...formRoles, { worker_id: '', role: 'app' }];
	}

	function removeRole(i: number) {
		formRoles = formRoles.filter((_, idx) => idx !== i);
	}

	async function submitCreate() {
		const roles: CreatePreviewSessionRolePayload[] = formRoles.filter((r) => r.worker_id !== '');
		if (roles.length === 0) {
			createError = 'Select at least one agent';
			return;
		}
		creating = true;
		createError = '';
		try {
			const sess = await createPreviewSession({ roles });
			sessions = [sess, ...sessions];
			showForm = false;
		} catch (e) {
			createError = e instanceof Error ? e.message : 'Failed to start preview';
		} finally {
			creating = false;
		}
	}

	async function handleStop(id: string) {
		try {
			await stopPreviewSession(id);
			sessions = sessions.map((s) => (s.id === id ? { ...s, status: 'stopped' } : s));
			if (logSessionId === id) stopLogPoll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to stop session';
		}
	}

	async function handleDelete(id: string) {
		try {
			await deletePreviewSession(id);
			sessions = sessions.filter((s) => s.id !== id);
			if (logSessionId === id) closeLog();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete session';
		}
	}

	async function openLog(sessId: string, roleId: string, label: string) {
		stopLogPoll();
		logSessionId = sessId;
		logRoleId = roleId;
		logRoleName = label;
		logContent = '';
		logLoading = true;
		await fetchLog();
		logLoading = false;

		const sess = sessions.find((s) => s.id === sessId);
		const role = sess?.roles.find((r) => r.id === roleId);
		if (role && (role.status === 'running' || role.status === 'starting')) {
			logPollTimer = setInterval(fetchLog, 2000);
		}
	}

	async function fetchLog() {
		if (!logSessionId || !logRoleId) return;
		try {
			const res = await getRoleLog(logSessionId, logRoleId);
			logContent = res.log;
		} catch {}
	}

	function stopLogPoll() {
		if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null; }
	}

	function closeLog() {
		stopLogPoll();
		logSessionId = null;
		logRoleId = null;
		logContent = '';
	}

	function workerName(id: string): string {
		return $allWorkers.find((w) => w.worker_id === id)?.name ?? id.slice(0, 8);
	}

	function statusColor(status: string): string {
		if (status === 'running') return '#4ade80';
		if (status === 'starting') return '#facc15';
		if (status === 'failed') return '#f87171';
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
	<div class="page-header">
		<h1 class="page-title">Preview Deploys</h1>
		<button class="btn-primary" on:click={openForm}>
			<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
			New Preview
		</button>
	</div>

	{#if error}
		<p class="err-banner">{error}</p>
	{/if}

	<!-- New preview form -->
	{#if showForm}
		<div class="card form-card">
			<div class="card-header">
				<span class="card-title">Start Preview Session</span>
				<button class="icon-btn" on:click={() => (showForm = false)}>✕</button>
			</div>

			{#if groupWorkers.length === 0}
				<p class="hint">No agents in the <strong>{$selectedGroup}</strong> group have a preview command configured. Set a <em>Preview Command</em> in each agent's settings first.</p>
			{:else}
				<div class="roles-list">
					{#each formRoles as row, i}
						<div class="role-row">
							<select class="nx-select" bind:value={row.worker_id}>
								<option value="">— select agent —</option>
								{#each groupWorkers as w}
									<option value={w.worker_id}>{w.name}</option>
								{/each}
							</select>
							<select class="nx-select role-select" bind:value={row.role}>
								<option value="app">app</option>
								<option value="frontend">frontend</option>
								<option value="backend">backend</option>
							</select>
							<button class="icon-btn danger" on:click={() => removeRole(i)} disabled={formRoles.length === 1}>✕</button>
						</div>
					{/each}
				</div>
				<button class="btn-ghost small" on:click={addRole}>+ Add agent</button>

				{#if createError}
					<p class="err-inline">{createError}</p>
				{/if}

				<div class="form-actions">
					<button class="btn-ghost" on:click={() => (showForm = false)}>Cancel</button>
					<button class="btn-primary" on:click={submitCreate} disabled={creating}>
						{creating ? 'Starting…' : 'Start Preview'}
					</button>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Log viewer -->
	{#if logRoleId}
		<div class="log-panel">
			<div class="log-header">
				<span class="log-title">Logs — {logRoleName}</span>
				<button class="icon-btn" on:click={closeLog}>✕</button>
			</div>
			{#if logLoading}
				<p class="hint log-hint">Loading…</p>
			{:else if logContent === ''}
				<p class="hint log-hint">No output yet.</p>
			{:else}
				<pre class="log-output">{logContent}</pre>
			{/if}
		</div>
	{/if}

	<!-- Session list -->
	{#if loading}
		<p class="hint">Loading…</p>
	{:else if sessions.length === 0}
		<div class="empty">
			<p>No preview sessions yet.</p>
			<p class="sub">Start a new preview to see your agents' apps live.</p>
		</div>
	{:else}
		<div class="sessions">
			{#each sessions as sess}
				<div class="session-card" class:stopped={sess.status === 'stopped'}>
					<div class="session-header">
						<div class="session-meta">
							<span class="status-dot" style="background:{statusColor(sess.status)}"></span>
							<span class="status-label">{sess.status}</span>
							<span class="ts">{relativeTime(sess.created_at)}</span>
						</div>
						<div class="session-actions">
							{#if sess.status === 'running' || sess.status === 'starting'}
								<button class="btn-ghost small danger" on:click={() => handleStop(sess.id)}>Stop</button>
							{/if}
							<button class="btn-ghost small" on:click={() => handleDelete(sess.id)}>Delete</button>
						</div>
					</div>

					{#if sess.error}
						<p class="err-inline">{sess.error}</p>
					{/if}

					<div class="roles">
						{#each sess.roles as role}
							<div class="role-item">
								<span class="role-badge">{role.role}</span>
								<span class="role-worker">{role.worker_name || workerName(role.worker_id)}</span>
								<span class="status-dot small" style="background:{statusColor(role.status)}"></span>
								{#if role.preview_url && role.status === 'running'}
									<a class="preview-link" href={role.preview_url} target="_blank" rel="noreferrer">
										{role.preview_url}
									</a>
								{:else}
									<span class="role-status">{role.status}</span>
								{/if}
								{#if role.error_message}
									<span class="role-error">{role.error_message}</span>
								{/if}
								<button
									class="btn-ghost small log-btn"
									class:active={logRoleId === role.id}
									on:click={() => openLog(sess.id, role.id, `${role.role} / ${role.worker_name || workerName(role.worker_id)}`)}
								>Logs</button>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page {
		padding: 24px;
		max-width: 860px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.page-title {
		font-size: 14px;
		font-weight: 600;
		color: var(--nx-text-primary, #e2e8f0);
		margin: 0;
	}

	.btn-primary {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: var(--nx-accent, #818cf8);
		color: #fff;
		border: none;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-ghost {
		padding: 5px 10px;
		background: transparent;
		color: var(--nx-text-secondary, #94a3b8);
		border: 1px solid var(--nx-border, #334155);
		border-radius: 6px;
		font-size: 12px;
		cursor: pointer;
	}

	.btn-ghost.small { padding: 3px 8px; font-size: 11px; }
	.btn-ghost.danger { color: #f87171; border-color: #f87171; }
	.btn-ghost:hover:not(:disabled) { background: rgba(255,255,255,0.05); }
	.btn-ghost:disabled { opacity: 0.4; cursor: not-allowed; }
	.btn-ghost.active { background: rgba(129, 140, 248, 0.15); color: #818cf8; border-color: #818cf8; }

	.log-btn { margin-left: auto; }

	.icon-btn {
		background: transparent;
		border: none;
		color: var(--nx-text-muted, #64748b);
		cursor: pointer;
		font-size: 13px;
		padding: 2px 6px;
		line-height: 1;
	}

	.icon-btn.danger { color: #f87171; }

	.card {
		background: var(--nx-surface, #0f172a);
		border: 1px solid var(--nx-border, #1e293b);
		border-radius: 8px;
		padding: 16px;
	}

	.form-card {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.card-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--nx-text-primary, #e2e8f0);
	}

	.roles-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.role-row {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.nx-select {
		flex: 1;
		background: var(--nx-input-bg, #1e293b);
		color: var(--nx-text-primary, #e2e8f0);
		border: 1px solid var(--nx-border, #334155);
		border-radius: 6px;
		padding: 6px 10px;
		font-size: 12px;
	}

	.role-select { flex: 0 0 110px; }

	.form-actions {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
		margin-top: 4px;
	}

	/* Log panel */
	.log-panel {
		background: #0a0f1a;
		border: 1px solid #1e293b;
		border-radius: 8px;
		display: flex;
		flex-direction: column;
		max-height: 320px;
		overflow: hidden;
	}

	.log-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 14px;
		border-bottom: 1px solid #1e293b;
		flex-shrink: 0;
	}

	.log-title {
		font-size: 11px;
		font-weight: 600;
		color: #4ade80;
		font-family: monospace;
	}

	.log-hint {
		padding: 12px 14px;
	}

	.log-output {
		font-family: 'SF Mono', 'Fira Code', monospace;
		font-size: 11px;
		line-height: 1.5;
		color: #a3e635;
		white-space: pre-wrap;
		word-break: break-all;
		padding: 12px 14px;
		margin: 0;
		overflow-y: auto;
		flex: 1;
	}

	.sessions {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.session-card {
		background: var(--nx-surface, #0f172a);
		border: 1px solid var(--nx-border, #1e293b);
		border-radius: 8px;
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.session-card.stopped {
		opacity: 0.6;
	}

	.session-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.session-meta {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.session-actions {
		display: flex;
		gap: 6px;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.status-dot.small {
		width: 6px;
		height: 6px;
	}

	.status-label {
		font-size: 12px;
		font-weight: 500;
		color: var(--nx-text-primary, #e2e8f0);
		text-transform: capitalize;
	}

	.ts {
		font-size: 11px;
		color: var(--nx-text-muted, #64748b);
	}

	.roles {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.role-item {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 12px;
	}

	.role-badge {
		background: rgba(129, 140, 248, 0.15);
		color: #818cf8;
		border-radius: 4px;
		padding: 1px 6px;
		font-size: 11px;
		font-weight: 500;
	}

	.role-worker {
		color: var(--nx-text-primary, #e2e8f0);
		min-width: 80px;
	}

	.role-status {
		color: var(--nx-text-muted, #64748b);
		font-size: 11px;
		text-transform: capitalize;
	}

	.role-error {
		color: #f87171;
		font-size: 11px;
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 300px;
	}

	.preview-link {
		color: #38bdf8;
		font-size: 11px;
		text-decoration: none;
	}

	.preview-link:hover { text-decoration: underline; }

	.empty {
		text-align: center;
		padding: 48px 0;
		color: var(--nx-text-muted, #64748b);
	}

	.empty p { margin: 4px 0; font-size: 13px; }
	.empty .sub { font-size: 12px; }

	.hint {
		font-size: 12px;
		color: var(--nx-text-muted, #64748b);
		margin: 0;
	}

	.err-banner {
		background: rgba(248, 113, 113, 0.1);
		border: 1px solid rgba(248, 113, 113, 0.3);
		color: #f87171;
		border-radius: 6px;
		padding: 8px 12px;
		font-size: 12px;
		margin: 0;
	}

	.err-inline {
		color: #f87171;
		font-size: 12px;
		margin: 0;
	}
</style>
