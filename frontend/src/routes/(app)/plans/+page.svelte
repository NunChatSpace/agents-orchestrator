<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';

	function renderMd(content: string): string {
		const html = marked.parse(content, { async: false }) as string;
		return DOMPurify.sanitize(html);
	}
	import { goto } from '$app/navigation';
	import { allWorkers } from '../../../stores/workers';
	import { selectedGroup } from '../../../stores/selectedGroup';
	import { upsertJob } from '../../../stores/jobs';
	import { createJob, submitJob } from '../../../lib/apis/jobs';
	import {
		listPlanSessions,
		createPlanSession,
		getPlanSession,
		sendPlanMessage,
		generatePlan,
		completePlanSession,
		discardPlanSession
	} from '../../../lib/apis/planSessions';
	import type { PlanSession, PlanMessage } from '../../../types/plan_session';

	// Workers for the selected group (for new session worker selector)
	$: groupWorkers = $allWorkers.filter((w) => w.group_name === $selectedGroup);

	// Plan sessions filtered to those whose worker belongs to the selected group
	$: groupWorkerIDs = new Set(groupWorkers.map((w) => w.worker_id));
	$: filteredSessions = planSessions.filter((s) => groupWorkerIDs.has(s.worker_id));

	// ── State ────────────────────────────────────────────────────────────
	let planSessions: PlanSession[] = [];
	let planSessionsLoading = false;
	let planLoaded = false;
	let activeSession: PlanSession | null = null;
	let chatMessages: PlanMessage[] = [];
	let composerText = '';
	let isSending = false;
	let thinkingText = '';
	let isGenerating = false;
	let generateElapsed = 0;
	let generateTimer: ReturnType<typeof setInterval> | null = null;
	let generatedPrompt = '';
	let showConfirm = false;
	let newSessionWorkerID = '';
	let jobLoading = false;
	let planError = '';
	let optimisticMessageCounter = 0;
	let chatFeed: HTMLElement;
	let orphanedMessage = ''; // last user msg with no agent reply (interrupted stream)

	async function scrollToBottom() {
		await tick();
		chatFeed?.scrollTo({ top: chatFeed.scrollHeight, behavior: 'smooth' });
	}

	// Scroll when thinking indicator appears
	$: if (thinkingText || isSending || isGenerating) scrollToBottom();

	function createClientMessageId(): string {
		const maybeCrypto = globalThis.crypto;
		if (maybeCrypto && typeof maybeCrypto.randomUUID === 'function') {
			return maybeCrypto.randomUUID();
		}
		optimisticMessageCounter += 1;
		const rand = Math.random().toString(36).slice(2, 10);
		return `tmp-${Date.now()}-${optimisticMessageCounter}-${rand}`;
	}

	// Auto-select first worker for new session
	$: if (
		groupWorkers.length > 0 &&
		(!newSessionWorkerID || !groupWorkers.find((w) => w.worker_id === newSessionWorkerID))
	) {
		newSessionWorkerID = groupWorkers[0].worker_id;
	}

	onMount(() => {
		if (planLoaded) return;
		planLoaded = true;
		loadSessions();
	});

	function relativeTime(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		return `${Math.floor(hrs / 24)}d ago`;
	}

	async function loadSessions() {
		planSessionsLoading = true;
		try {
			planSessions = await listPlanSessions();
		} finally {
			planSessionsLoading = false;
		}
	}

	async function openSession(session: PlanSession) {
		planError = '';
		const full = await getPlanSession(session.id);
		activeSession = full;
		chatMessages = full.messages ?? [];
		generatedPrompt = full.generated_prompt ?? '';
		showConfirm = !!full.generated_prompt;
		composerText = '';
		thinkingText = '';
		// Detect interrupted stream: last message is from user with no agent reply
		const last = chatMessages.at(-1);
		orphanedMessage = last?.role === 'user' ? last.content : '';
		scrollToBottom();
	}

	function backToList() {
		activeSession = null;
		chatMessages = [];
		generatedPrompt = '';
		showConfirm = false;
		thinkingText = '';
		planError = '';
		orphanedMessage = '';
		loadSessions();
	}

	async function startNewSession() {
		if (!newSessionWorkerID) return;
		try {
			const session = await createPlanSession(newSessionWorkerID);
			await openSession(session);
		} catch (e: unknown) {
			planError = e instanceof Error ? e.message : 'Failed to create session';
		}
	}

	async function handleSendMessage() {
		if (!activeSession || !composerText.trim() || isSending) return;
		const content = composerText.trim();
		composerText = '';
		isSending = true;
		thinkingText = '';
		planError = '';

		chatMessages = [
			...chatMessages,
			{ id: createClientMessageId(), role: 'user', content, created_at: new Date().toISOString() }
		];
		scrollToBottom();

		sendPlanMessage(
			activeSession.id,
			content,
			(thinking) => { thinkingText = thinking; },
			async () => {
				thinkingText = '';
				isSending = false;
				const updated = await getPlanSession(activeSession!.id);
				activeSession = updated;
				chatMessages = updated.messages ?? [];
				scrollToBottom();
			},
			(err) => {
				thinkingText = '';
				isSending = false;
				planError = err;
			}
		);
	}

	function handleGenerate() {
		if (!activeSession || isGenerating) return;
		isGenerating = true;
		generateElapsed = 0;
		generateTimer = setInterval(() => generateElapsed++, 1000);
		thinkingText = '';
		planError = '';

		generatePlan(
			activeSession.id,
			(thinking) => { thinkingText = thinking; },
			async (prompt) => {
				thinkingText = '';
				isGenerating = false;
				if (generateTimer) { clearInterval(generateTimer); generateTimer = null; }
				generatedPrompt = prompt;
				showConfirm = true;
				const updated = await getPlanSession(activeSession!.id);
				activeSession = updated;
				chatMessages = updated.messages ?? [];
			},
			(err) => {
				thinkingText = '';
				isGenerating = false;
				if (generateTimer) { clearInterval(generateTimer); generateTimer = null; }
				planError = err;
			}
		);
	}

	async function handleConfirm() {
		if (!activeSession || jobLoading) return;
		jobLoading = true;
		try {
			await completePlanSession(activeSession.id);
			const workerID = activeSession.worker_id;
			const job = await createJob({
				title: activeSession.title || generatedPrompt.slice(0, 80),
				prompt: generatedPrompt,
				target_group: $selectedGroup,
				manual_worker_override: workerID
			});
			upsertJob(job);
			const submitted = await submitJob(job.job_id, workerID);
			upsertJob(submitted);
			goto(`/jobs/${job.job_id}`);
		} catch {
			// ignore
		} finally {
			jobLoading = false;
		}
	}

	async function handleDiscard() {
		if (!activeSession) return;
		await discardPlanSession(activeSession.id);
		backToList();
	}
</script>

<div class="plans-wrap">
<div class="page-inner">
	{#if !activeSession}
		<!-- Session list view -->
		<div class="ps-list-view">
			<div class="ps-list-header">
				<div class="ps-list-top">
					<span class="ps-list-title">Plan Sessions</span>
					{#if filteredSessions.length > 0}
						<span class="count-badge">{filteredSessions.length}</span>
					{/if}
				</div>
				{#if groupWorkers.length === 0}
				<div class="no-agents-hint">
					No agents in <strong>{$selectedGroup || 'this workspace'}</strong>.
					<a href="/agents/new" class="no-agents-link">Add an agent</a> to start planning.
				</div>
			{:else}
				<div class="ps-new-row">
					<select class="field-select ps-worker-select" bind:value={newSessionWorkerID}>
						{#each groupWorkers as w}
							<option value={w.worker_id}>{w.name}</option>
						{/each}
					</select>
					<button class="discuss-btn" on:click={startNewSession} disabled={!newSessionWorkerID}>
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
						</svg>
						Let's discuss plan
					</button>
				</div>
			{/if}
				{#if planError}
					<div class="plan-error">
						<span class="plan-error-title">Error</span>
						<p>{planError}</p>
					</div>
				{/if}
			</div>

			{#if planSessionsLoading}
				<div class="ps-empty">Loading…</div>
			{:else if filteredSessions.length === 0}
				<div class="ps-empty">No plan sessions yet. Start one above.</div>
			{:else}
				<div class="ps-cards">
					{#each filteredSessions as session (session.id)}
						<button class="ps-card" on:click={() => openSession(session)}>
							<div class="ps-card-title">{session.title || 'Untitled session'}</div>
							<div class="ps-card-meta">
								<span class="ps-status-badge" data-status={session.status}>{session.status}</span>
								<span class="ps-card-date">{relativeTime(session.updated_at)}</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}
		</div>

	{:else}
		<!-- Session detail: chat view -->
		<div class="ps-detail">
			<div class="ps-detail-header">
				<button class="ps-back" on:click={backToList}>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="15 18 9 12 15 6"/>
					</svg>
					Sessions
				</button>
				<span class="ps-detail-title">{activeSession.title || 'New session'}</span>
				<div class="ps-detail-actions">
					{#if !showConfirm}
						<button
							class="generate-btn"
							on:click={handleGenerate}
							disabled={chatMessages.length < 2 || isGenerating || isSending}
						>
							{#if isGenerating}
								<span class="btn-spinner"></span>
								Generating… {generateElapsed}s
							{:else}
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
								</svg>
								Generate Plan
							{/if}
						</button>
					{/if}
					<button class="discard-btn" on:click={handleDiscard} disabled={isSending || isGenerating}>
						Discard
					</button>
				</div>
			</div>

			<div class="chat-feed" bind:this={chatFeed}>
				{#each chatMessages as msg (msg.id)}
					<div class="chat-bubble" class:user={msg.role === 'user'} class:agent={msg.role === 'agent'}>
						<span class="bubble-role">{msg.role === 'user' ? 'You' : 'Agent'}</span>
						{#if msg.role === 'user'}
							<div class="bubble-content">{msg.content}</div>
						{:else}
							<div class="bubble-content md-body">{@html renderMd(msg.content)}</div>
						{/if}
					</div>
				{/each}

				{#if thinkingText || (isSending && !thinkingText) || (isGenerating && !thinkingText)}
					<div class="thinking-indicator">
						<span class="thinking-dot"></span>
						<span class="thinking-text">{thinkingText || 'Agent is thinking…'}</span>
					</div>
				{/if}
			</div>

			{#if planError}
				<div class="plan-error">
					<span class="plan-error-title">Error</span>
					<p>{planError}</p>
				</div>
			{/if}

			{#if showConfirm}
				<div class="confirm-section">
					<div class="confirm-header">
						<span class="result-label">Generated Prompt</span>
						<span class="result-hint">Review and edit before confirming</span>
					</div>
					<textarea class="field-textarea result-textarea" bind:value={generatedPrompt} rows="8"></textarea>
					<button
						class="submit-btn"
						on:click={handleConfirm}
						disabled={!generatedPrompt.trim() || jobLoading}
					>
						{#if jobLoading}
							<span class="btn-spinner"></span>
							Creating job…
						{:else}
							<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
								<line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
							</svg>
							Confirm & Create Job
						{/if}
					</button>
				</div>
			{:else if activeSession.status === 'pending'}
				{#if orphanedMessage}
					<div class="orphan-notice">
						<span class="orphan-icon">⚠</span>
						<span class="orphan-text">Agent didn't respond to your last message (connection was interrupted).</span>
						<button class="orphan-resend" on:click={() => { composerText = orphanedMessage; orphanedMessage = ''; }}>
							Resend
						</button>
					</div>
				{/if}
				<div class="ps-composer">
					<textarea
						class="composer-input"
						bind:value={composerText}
						placeholder={isSending || isGenerating ? 'Agent is responding…' : 'Type a message… (Shift+Enter for newline)'}
						disabled={isSending || isGenerating}
						rows="3"
						on:keydown={(e) => {
							if (e.key === 'Enter' && !e.shiftKey) {
								e.preventDefault();
								handleSendMessage();
							}
						}}
					></textarea>
					<button
						class="send-btn"
						on:click={handleSendMessage}
						disabled={!composerText.trim() || isSending || isGenerating}
					>
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
						</svg>
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
</div>

<style>
	.plans-wrap {
		flex: 1;
		overflow-y: auto;
		padding: 28px 32px;
	}

	.page-inner {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 20px;
		height: 100%;
	}

	/* Session list view */
	.ps-list-view {
		display: flex;
		flex-direction: column;
		gap: 16px;
		max-width: 700px;
	}
	.ps-list-header {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.ps-list-top {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.ps-list-title {
		font-size: 11px;
		font-weight: 600;
		color: rgba(196,181,253,0.7);
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	.count-badge {
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
	.ps-new-row {
		display: flex;
		gap: 10px;
		align-items: center;
	}
	.ps-worker-select { flex: 1; max-width: 220px; }
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
	.discuss-btn {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 10px 18px;
		border-radius: 8px;
		background: linear-gradient(135deg, rgba(124,58,237,0.75), rgba(109,40,217,0.65));
		border: 1px solid rgba(139,92,246,0.4);
		color: #f0f0ff;
		font-size: 13px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		transition: opacity 0.15s, box-shadow 0.15s;
		box-shadow: 0 0 14px rgba(124,58,237,0.2);
		white-space: nowrap;
	}
	.discuss-btn:hover:not(:disabled) { box-shadow: 0 0 22px rgba(124,58,237,0.4); }
	.discuss-btn:disabled { opacity: 0.45; cursor: not-allowed; }

	.no-agents-hint {
		font-size: 13px;
		color: rgba(196,181,253,0.45);
		padding: 8px 0;
	}
	.no-agents-hint strong { color: rgba(196,181,253,0.7); font-weight: 600; }
	.no-agents-link {
		color: #a78bfa;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	.no-agents-link:hover { color: #c4b5fd; }

	.ps-empty {
		font-size: 13px;
		color: rgba(196,181,253,0.3);
		font-style: italic;
		padding: 24px 0;
	}
	.ps-cards {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.ps-card {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 14px 16px;
		background: rgba(13,13,30,0.6);
		border: 1px solid rgba(139,92,246,0.15);
		border-radius: 10px;
		text-align: left;
		cursor: pointer;
		font-family: inherit;
		transition: border-color 0.15s, background 0.15s;
	}
	.ps-card:hover {
		border-color: rgba(139,92,246,0.35);
		background: rgba(13,13,30,0.8);
	}
	.ps-card-title {
		font-size: 13.5px;
		font-weight: 500;
		color: #f0f0ff;
	}
	.ps-card-meta {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.ps-status-badge {
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		padding: 2px 7px;
		border-radius: 5px;
		background: rgba(139,92,246,0.12);
		color: rgba(167,139,250,0.7);
		border: 1px solid rgba(139,92,246,0.18);
	}
	.ps-status-badge[data-status="completed"] {
		background: rgba(74,222,128,0.08);
		color: rgba(74,222,128,0.75);
		border-color: rgba(74,222,128,0.2);
	}
	.ps-card-date {
		font-size: 11px;
		color: rgba(196,181,253,0.3);
	}

	/* Session detail / chat view */
	.ps-detail {
		display: flex;
		flex-direction: column;
		gap: 0;
		flex: 1;
		min-height: 0;
		max-width: 760px;
		margin: 0 auto;
		width: 100%;
	}
	.ps-detail-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding-bottom: 14px;
		border-bottom: 1px solid rgba(139,92,246,0.12);
		flex-shrink: 0;
	}
	.ps-back {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 5px 10px;
		border-radius: 7px;
		background: transparent;
		border: 1px solid rgba(139,92,246,0.18);
		color: rgba(167,139,250,0.6);
		font-size: 12px;
		font-family: inherit;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
		flex-shrink: 0;
	}
	.ps-back:hover { background: rgba(139,92,246,0.08); color: #a78bfa; }
	.ps-detail-title {
		flex: 1;
		font-size: 14px;
		font-weight: 600;
		color: #f0f0ff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.ps-detail-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.chat-feed {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 20px 0;
		flex: 1;
		overflow-y: auto;
		min-height: 180px;
	}
	.chat-bubble {
		display: flex;
		flex-direction: column;
		gap: 4px;
		max-width: 82%;
	}
	.chat-bubble.user { align-self: flex-end; align-items: flex-end; }
	.chat-bubble.agent { align-self: flex-start; align-items: flex-start; }
	.bubble-role {
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.35);
	}
	.bubble-content {
		padding: 10px 14px;
		border-radius: 10px;
		font-size: 13px;
		line-height: 1.6;
	}
	.chat-bubble.user .bubble-content { white-space: pre-wrap; }
	.chat-bubble.user .bubble-content {
		background: linear-gradient(135deg, rgba(124,58,237,0.5), rgba(109,40,217,0.4));
		border: 1px solid rgba(139,92,246,0.35);
		color: #f0f0ff;
	}
	.chat-bubble.agent .bubble-content {
		background: rgba(13,13,30,0.7);
		border: 1px solid rgba(139,92,246,0.15);
		color: rgba(230,225,255,0.92);
	}

	/* ── Markdown body styles (md-body) ─────────────────────────────── */
	/* MessageBubble.svelte is not mounted here, so we define these locally. */
	:global(.chat-bubble.agent .md-body) { line-height: 1.7; color: rgba(230,225,255,0.92); }
	:global(.chat-bubble.agent .md-body p) { margin: 0 0 0.65em; }
	:global(.chat-bubble.agent .md-body p:last-child) { margin-bottom: 0; }
	:global(.chat-bubble.agent .md-body h1,
	        .chat-bubble.agent .md-body h2,
	        .chat-bubble.agent .md-body h3,
	        .chat-bubble.agent .md-body h4) {
		font-weight: 700;
		color: #f0f0ff;
		margin: 0.9em 0 0.4em;
		line-height: 1.3;
	}
	:global(.chat-bubble.agent .md-body h1) { font-size: 1.1em; }
	:global(.chat-bubble.agent .md-body h2) { font-size: 1.02em; }
	:global(.chat-bubble.agent .md-body h3,
	        .chat-bubble.agent .md-body h4) { font-size: 0.95em; }
	:global(.chat-bubble.agent .md-body ul,
	        .chat-bubble.agent .md-body ol) {
		margin: 0.4em 0 0.65em;
		padding-left: 1.5em;
	}
	:global(.chat-bubble.agent .md-body li) { margin-bottom: 0.3em; }
	:global(.chat-bubble.agent .md-body code) {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 0.85em;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.25);
		border-radius: 4px;
		padding: 1px 5px;
		color: #c4b5fd;
		word-break: break-all;
	}
	:global(.chat-bubble.agent .md-body pre) {
		background: rgba(8,8,20,0.85);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 8px;
		padding: 12px 14px;
		overflow-x: auto;
		margin: 0.55em 0;
	}
	:global(.chat-bubble.agent .md-body pre code) {
		background: none;
		border: none;
		padding: 0;
		font-size: 0.84em;
		color: rgba(220,210,255,0.92);
		word-break: normal;
	}
	:global(.chat-bubble.agent .md-body blockquote) {
		border-left: 3px solid rgba(139,92,246,0.45);
		margin: 0.5em 0;
		padding: 4px 12px;
		color: rgba(196,181,253,0.65);
		font-style: italic;
	}
	:global(.chat-bubble.agent .md-body strong) { color: #f0f0ff; font-weight: 700; }
	:global(.chat-bubble.agent .md-body em) { color: rgba(196,181,253,0.85); }
	:global(.chat-bubble.agent .md-body hr) {
		border: none;
		border-top: 1px solid rgba(139,92,246,0.15);
		margin: 0.75em 0;
	}
	:global(.chat-bubble.agent .md-body a) {
		color: #a78bfa;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	:global(.chat-bubble.agent .md-body table) {
		border-collapse: collapse;
		width: 100%;
		margin: 0.5em 0;
		font-size: 0.88em;
	}
	:global(.chat-bubble.agent .md-body th,
	        .chat-bubble.agent .md-body td) {
		border: 1px solid rgba(139,92,246,0.2);
		padding: 5px 10px;
		text-align: left;
	}
	:global(.chat-bubble.agent .md-body th) {
		background: rgba(139,92,246,0.1);
		color: rgba(196,181,253,0.9);
		font-weight: 600;
	}

	.thinking-indicator {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 8px 0;
	}
	.thinking-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: rgba(139,92,246,0.5);
		flex-shrink: 0;
		margin-top: 4px;
		animation: pulse 1.2s ease-in-out infinite;
	}
	@keyframes pulse {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 1; }
	}
	.thinking-text {
		font-size: 11.5px;
		color: rgba(167,139,250,0.5);
		font-style: italic;
		line-height: 1.5;
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
	}

	/* Orphaned message notice */
	.orphan-notice {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: rgba(234,179,8,0.07);
		border: 1px solid rgba(234,179,8,0.22);
		border-radius: 8px;
		flex-shrink: 0;
		margin-bottom: 4px;
	}
	.orphan-icon {
		font-size: 12px;
		color: rgba(253,224,71,0.7);
		flex-shrink: 0;
	}
	.orphan-text {
		flex: 1;
		font-size: 12px;
		color: rgba(253,224,71,0.65);
	}
	.orphan-resend {
		padding: 4px 10px;
		border-radius: 6px;
		background: rgba(234,179,8,0.12);
		border: 1px solid rgba(234,179,8,0.3);
		color: rgba(253,224,71,0.85);
		font-size: 12px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		flex-shrink: 0;
		transition: background 0.15s;
	}
	.orphan-resend:hover { background: rgba(234,179,8,0.2); }

	/* Composer */
	.ps-composer {
		display: flex;
		gap: 8px;
		align-items: flex-end;
		padding-top: 12px;
		border-top: 1px solid rgba(139,92,246,0.12);
		flex-shrink: 0;
	}
	.composer-input {
		flex: 1;
		background: rgba(13,13,30,0.85);
		border: 1px solid rgba(139,92,246,0.22);
		border-radius: 8px;
		padding: 10px 14px;
		color: #f0f0ff;
		font-size: 13px;
		font-family: inherit;
		line-height: 1.6;
		resize: none;
	}
	.composer-input::placeholder { color: rgba(196,181,253,0.25); }
	.composer-input:focus { outline: none; border-color: rgba(139,92,246,0.5); }
	.composer-input:disabled { opacity: 0.5; }
	.send-btn {
		width: 38px;
		height: 38px;
		border-radius: 8px;
		background: linear-gradient(135deg, rgba(124,58,237,0.75), rgba(109,40,217,0.65));
		border: 1px solid rgba(139,92,246,0.4);
		color: #f0f0ff;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		transition: box-shadow 0.15s, opacity 0.15s;
	}
	.send-btn:hover:not(:disabled) { box-shadow: 0 0 14px rgba(124,58,237,0.4); }
	.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* Generate + discard buttons */
	.generate-btn {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 7px 14px;
		border-radius: 7px;
		background: linear-gradient(135deg, rgba(124,58,237,0.75), rgba(109,40,217,0.65));
		border: 1px solid rgba(139,92,246,0.4);
		color: #f0f0ff;
		font-size: 12px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		transition: opacity 0.15s, box-shadow 0.15s;
		box-shadow: 0 0 10px rgba(124,58,237,0.15);
	}
	.generate-btn:hover:not(:disabled) { box-shadow: 0 0 18px rgba(124,58,237,0.35); }
	.generate-btn:disabled { opacity: 0.45; cursor: not-allowed; }
	.discard-btn {
		padding: 7px 12px;
		border-radius: 7px;
		background: transparent;
		border: 1px solid rgba(239,68,68,0.2);
		color: rgba(252,165,165,0.55);
		font-size: 12px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}
	.discard-btn:hover:not(:disabled) {
		background: rgba(239,68,68,0.08);
		color: rgba(252,165,165,0.85);
		border-color: rgba(239,68,68,0.35);
	}
	.discard-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* Confirm / generated prompt section */
	.confirm-section {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding-top: 14px;
		border-top: 1px solid rgba(139,92,246,0.12);
		flex-shrink: 0;
	}
	.confirm-header {
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
	.result-textarea { min-height: 160px; }

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

	/* Error block */
	.plan-error {
		font-size: 12.5px;
		color: rgba(252,165,165,0.85);
		background: rgba(239,68,68,0.08);
		border: 1px solid rgba(239,68,68,0.2);
		border-radius: 7px;
		padding: 8px 12px;
		flex-shrink: 0;
	}
	.plan-error-title {
		display: block;
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(252,165,165,0.6);
		margin-bottom: 4px;
	}
	.plan-error p { margin: 0; }

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
