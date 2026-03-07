<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';
	import Badge from '../atoms/Badge.svelte';
	import type { Message } from '../../types/message';

	function renderMd(content: string): string {
		const html = marked.parse(content, { async: false }) as string;
		return DOMPurify.sanitize(html);
	}

	export let message: Message;

	const dispatch = createEventDispatcher<{ choose: string }>();

	const isUser = message.role === 'user';
	const isOAgent = message.role === 'oagent';
	const isThinking = message.kind === 'thinking';
	const isFileChange = message.kind === 'file_change';
	const isQuestion = message.kind === 'question';
	const isSystemNote = message.kind === 'system';
	const isInstruction = message.kind === 'instruction';

	$: fileChange = isFileChange
		? (() => { try { return JSON.parse(message.content) as { op: string; path: string }; } catch { return null; } })()
		: null;

	const kindLabel: Record<string, string> = {
		instruction: 'instruction',
		question: 'question',
		answer: 'answer',
		summary: 'summary',
		system: 'system',
		thinking: 'thinking'
	};

	const kindColor: Record<string, 'gray' | 'blue' | 'purple' | 'orange'> = {
		instruction: 'blue',
		question: 'orange',
		answer: 'gray',
		summary: 'purple',
		system: 'gray',
		thinking: 'purple'
	};

	// Parse numbered choices from question messages:
	// "Which approach?\n1. Option A\n2. Option B"
	function parseChoices(content: string): { preamble: string; choices: string[] } | null {
		const lines = content.split('\n');
		const choices: string[] = [];
		const textLines: string[] = [];
		for (const line of lines) {
			const m = line.match(/^\s*(\d+)[.)]\s+(.+)$/);
			if (m) {
				choices.push(m[2].trim());
			} else {
				textLines.push(line);
			}
		}
		if (choices.length >= 2) {
			return { preamble: textLines.join('\n').trim(), choices };
		}
		return null;
	}

	$: parsed = isQuestion ? parseChoices(message.content) : null;

	function thinkingLabel(content: string): string {
		const idx = content.indexOf(':');
		if (idx > 0) {
			const tool = content.slice(0, idx).trim();
			const rest = content.slice(idx + 1).trim();
			const truncated = rest.length > 120 ? rest.slice(0, 120) + '…' : rest;
			return `${tool}: ${truncated}`;
		}
		return content.length > 140 ? content.slice(0, 140) + '…' : content;
	}
</script>

<!-- Instruction: prominent task brief block -->
{#if isInstruction}
	<div class="instruction-block">
		<div class="instruction-header">
			<span class="instruction-icon">⬡</span>
			<span class="instruction-label">Instruction sent to agent</span>
		</div>
		<div class="instruction-text md-body">{@html renderMd(message.content)}</div>
	</div>

<!-- Thinking step: compact timeline row -->
{:else if isThinking}
	<div class="thinking-row">
		<span class="thinking-dot"></span>
		<span class="thinking-text">{thinkingLabel(message.content)}</span>
	</div>

<!-- File change: compact inline badge (full diff is in the Changes panel) -->
{:else if isFileChange && fileChange}
	<div class="file-change-row">
		<span class="file-op-badge {fileChange.op === 'write' ? 'badge-write' : 'badge-edit'}">
			{fileChange.op === 'write' ? '+' : '~'}
		</span>
		<span class="file-path">{fileChange.path}</span>
	</div>

<!-- System note: centred divider -->
{:else if isSystemNote}
	<div class="system-row">
		<span class="system-line"></span>
		<span class="system-text">{message.content}</span>
		<span class="system-line"></span>
	</div>

<!-- Regular chat bubble -->
{:else}
	<div class="bubble-row {isUser ? 'user' : 'agent'}">
		<div class="bubble-wrap">
			<div class="bubble {isUser ? 'bubble-user' : isOAgent ? 'bubble-oagent' : 'bubble-worker'}">
				{#if parsed}
					{#if parsed.preamble}<p class="choice-preamble">{parsed.preamble}</p>{/if}
					<div class="choice-list">
						{#each parsed.choices as choice, i}
							<button class="choice-btn" on:click={() => dispatch('choose', choice)}>
								<span class="choice-num">{i + 1}</span>
								{choice}
							</button>
						{/each}
					</div>
				{:else}
					<div class="md-body">{@html renderMd(message.content)}</div>
				{/if}
			</div>
			<div class="bubble-meta {isUser ? 'meta-right' : ''}">
				<span class="role-label">{message.role}</span>
				<Badge color={kindColor[message.kind] ?? 'gray'}>{kindLabel[message.kind] ?? message.kind}</Badge>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Markdown body styles — applied to both bubble content and instruction block */
	:global(.md-body) { line-height: 1.65; }
	:global(.md-body p) { margin: 0 0 0.6em; }
	:global(.md-body p:last-child) { margin-bottom: 0; }
	:global(.md-body h1, .md-body h2, .md-body h3, .md-body h4) {
		font-weight: 700;
		color: #f0f0ff;
		margin: 0.8em 0 0.35em;
		line-height: 1.3;
	}
	:global(.md-body h1) { font-size: 1.15em; }
	:global(.md-body h2) { font-size: 1.05em; }
	:global(.md-body h3, .md-body h4) { font-size: 0.95em; }
	:global(.md-body ul, .md-body ol) {
		margin: 0.4em 0 0.6em;
		padding-left: 1.4em;
	}
	:global(.md-body li) { margin-bottom: 0.25em; }
	:global(.md-body code) {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 0.87em;
		background: rgba(139,92,246,0.12);
		border: 1px solid rgba(139,92,246,0.2);
		border-radius: 4px;
		padding: 1px 5px;
		color: #c4b5fd;
	}
	:global(.md-body pre) {
		background: rgba(10,10,25,0.7);
		border: 1px solid rgba(139,92,246,0.18);
		border-radius: 7px;
		padding: 12px 14px;
		overflow-x: auto;
		margin: 0.5em 0;
	}
	:global(.md-body pre code) {
		background: none;
		border: none;
		padding: 0;
		font-size: 0.85em;
		color: rgba(220,210,255,0.9);
	}
	:global(.md-body blockquote) {
		border-left: 3px solid rgba(139,92,246,0.4);
		margin: 0.5em 0;
		padding: 4px 12px;
		color: rgba(196,181,253,0.65);
		font-style: italic;
	}
	:global(.md-body hr) {
		border: none;
		border-top: 1px solid rgba(139,92,246,0.15);
		margin: 0.75em 0;
	}
	:global(.md-body a) {
		color: #a78bfa;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	:global(.md-body strong) { color: #f0f0ff; font-weight: 700; }
	:global(.md-body em) { color: rgba(196,181,253,0.85); }
	:global(.md-body table) {
		border-collapse: collapse;
		width: 100%;
		margin: 0.5em 0;
		font-size: 0.88em;
	}
	:global(.md-body th, .md-body td) {
		border: 1px solid rgba(139,92,246,0.18);
		padding: 5px 10px;
		text-align: left;
	}
	:global(.md-body th) {
		background: rgba(139,92,246,0.1);
		color: rgba(196,181,253,0.85);
		font-weight: 600;
	}

	/* Instruction block */
	.instruction-block {
		margin: 8px 16px 12px;
		padding: 12px 16px;
		border-radius: 10px;
		background: rgba(139,92,246,0.07);
		border: 1px solid rgba(139,92,246,0.28);
		border-left: 3px solid rgba(139,92,246,0.7);
	}
	.instruction-header {
		display: flex;
		align-items: center;
		gap: 7px;
		margin-bottom: 8px;
	}
	.instruction-icon {
		font-size: 12px;
		color: rgba(167,139,250,0.7);
		line-height: 1;
	}
	.instruction-label {
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(139,92,246,0.75);
	}
	.instruction-text {
		margin: 0;
		font-size: 13px;
		line-height: 1.65;
		color: rgba(220,210,255,0.9);
		white-space: pre-wrap;
	}

	/* Thinking step */
	.thinking-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 3px 4px 3px 16px;
		margin-bottom: 2px;
		opacity: 0.65;
	}
	.thinking-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: rgba(139,92,246,0.5);
		flex-shrink: 0;
		border: 1px solid rgba(139,92,246,0.35);
	}
	.thinking-text {
		font-size: 11.5px;
		font-family: 'SFMono-Regular', Consolas, monospace;
		color: rgba(167,139,250,0.65);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* File change row */
	.file-change-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 3px 4px 3px 16px;
		margin-bottom: 2px;
	}

	.file-op-badge {
		width: 16px;
		height: 16px;
		border-radius: 3px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 11px;
		font-weight: 700;
		flex-shrink: 0;
	}
	.badge-write {
		background: rgba(34, 197, 94, 0.15);
		border: 1px solid rgba(34, 197, 94, 0.35);
		color: rgba(134, 239, 172, 0.9);
	}
	.badge-edit {
		background: rgba(234, 179, 8, 0.12);
		border: 1px solid rgba(234, 179, 8, 0.3);
		color: rgba(253, 224, 71, 0.85);
	}
	.file-path {
		font-size: 11.5px;
		font-family: 'SFMono-Regular', Consolas, monospace;
		color: rgba(196, 181, 253, 0.7);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* System note */
	.system-row {
		display: flex;
		align-items: center;
		gap: 10px;
		margin: 8px 0;
		padding: 0 8px;
	}
	.system-line {
		flex: 1;
		height: 1px;
		background: rgba(139,92,246,0.12);
	}
	.system-text {
		font-size: 11px;
		color: rgba(196,181,253,0.38);
		font-style: italic;
		white-space: nowrap;
		flex-shrink: 0;
	}

	/* Chat bubbles */
	.bubble-row {
		display: flex;
		margin-bottom: 12px;
	}
	.bubble-row.user  { justify-content: flex-end; }
	.bubble-row.agent { justify-content: flex-start; }

	.bubble-wrap {
		max-width: 75%;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.bubble {
		border-radius: 12px;
		padding: 10px 14px;
		font-size: 13.5px;
		line-height: 1.6;
		border: 1px solid transparent;
	}

	.bubble-user {
		background: linear-gradient(135deg, rgba(124,58,237,0.7), rgba(109,40,217,0.6));
		color: #f0f0ff;
		border-color: rgba(139,92,246,0.4);
		border-bottom-right-radius: 4px;
		box-shadow: 0 0 16px rgba(124,58,237,0.2);
	}

	.bubble-oagent {
		background: rgba(13,13,30,0.8);
		backdrop-filter: blur(12px);
		color: #c4b5fd;
		border-color: rgba(139,92,246,0.22);
		border-bottom-left-radius: 4px;
	}

	.bubble-worker {
		background: rgba(13,13,30,0.7);
		color: rgba(240,240,255,0.88);
		border-color: rgba(139,92,246,0.18);
		border-bottom-left-radius: 4px;
	}

	.bubble-meta {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.meta-right { justify-content: flex-end; }

	.role-label {
		font-size: 10.5px;
		color: rgba(196,181,253,0.35);
		font-weight: 500;
	}

	/* Choice buttons */
	.choice-preamble {
		margin: 0 0 10px;
		white-space: pre-wrap;
	}

	.choice-list {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.choice-btn {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 12px;
		border-radius: 8px;
		background: rgba(139,92,246,0.08);
		border: 1px solid rgba(139,92,246,0.25);
		color: #c4b5fd;
		font-size: 13px;
		font-family: inherit;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s, border-color 0.15s;
	}
	.choice-btn:hover {
		background: rgba(139,92,246,0.18);
		border-color: rgba(139,92,246,0.45);
		color: #f0f0ff;
	}

	.choice-num {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: rgba(139,92,246,0.2);
		border: 1px solid rgba(139,92,246,0.35);
		color: #a78bfa;
		font-size: 11px;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
</style>
