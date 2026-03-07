<script lang="ts">
	import Button from '../atoms/Button.svelte';
	import { createEventDispatcher } from 'svelte';

	export let enabled = false;
	export let placeholder = 'Send a message...';
	export let loading = false;
	export let disabledMessage = 'Waiting for worker response…';

	let content = '';
	const dispatch = createEventDispatcher<{ send: string }>();

	function handleSubmit() {
		const text = content.trim();
		if (!text || !enabled || loading) return;
		dispatch('send', text);
		content = '';
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit();
		}
	}
</script>

<div class="composer">
	{#if !enabled}
		<p class="waiting-label">{disabledMessage}</p>
	{/if}
	<div class="composer-row">
		<textarea
			rows="2"
			bind:value={content}
			{placeholder}
			disabled={!enabled || loading}
			on:keydown={handleKeydown}
			class="composer-textarea {!enabled ? 'disabled' : ''}"
		/>
		<Button
			variant="primary"
			disabled={!enabled || loading || content.trim() === ''}
			on:click={handleSubmit}
		>
			<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="22" y1="2" x2="11" y2="13"/>
				<polygon points="22 2 15 22 11 13 2 9 22 2"/>
			</svg>
			Send
		</Button>
	</div>
</div>

<style>
	.composer {
		border-top: 1px solid rgba(139,92,246,0.18);
		padding: 12px 20px;
		background: rgba(10,10,22,0.7);
		backdrop-filter: blur(12px);
	}

	.waiting-label {
		font-size: 11.5px;
		color: rgba(196,181,253,0.35);
		text-align: center;
		padding-bottom: 8px;
	}

	.composer-row {
		display: flex;
		gap: 10px;
		align-items: flex-end;
	}

	.composer-textarea {
		flex: 1;
		resize: none;
		border-radius: 10px;
		background: rgba(139,92,246,0.04);
		border: 1px solid rgba(139,92,246,0.18);
		color: #f0f0ff;
		font-size: 13.5px;
		font-family: inherit;
		padding: 10px 14px;
		outline: none;
		transition: border-color 0.2s, box-shadow 0.2s;
		line-height: 1.55;
	}
	.composer-textarea::placeholder { color: rgba(196,181,253,0.28); }
	.composer-textarea:focus {
		border-color: rgba(139,92,246,0.40);
		box-shadow: 0 0 0 3px rgba(139,92,246,0.08);
	}
	.composer-textarea.disabled {
		opacity: 0.38;
		cursor: not-allowed;
	}
</style>
