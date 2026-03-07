<script lang="ts">
	import { afterUpdate, onMount, createEventDispatcher } from 'svelte';
	import MessageBubble from '../molecules/MessageBubble.svelte';
	import type { Message } from '../../types/message';

	export let messages: Message[];
	export let status: string = '';

	$: isWorking = status === 'busy' || status === 'assigned';

	let feedEl: HTMLDivElement;
	const dispatch = createEventDispatcher<{ choose: string }>();

	function scrollToBottom() {
		if (feedEl) feedEl.scrollTop = feedEl.scrollHeight;
	}

	onMount(scrollToBottom);
	afterUpdate(scrollToBottom);
</script>

<div bind:this={feedEl} class="feed">
	{#each messages as msg (msg.message_id)}
		<MessageBubble message={msg} on:choose={(e) => dispatch('choose', e.detail)} />
	{:else}
		<p class="empty">No messages yet.</p>
	{/each}
	{#if isWorking}
		<div class="working-row">
			<span class="working-dot"></span>
			<span class="working-dot"></span>
			<span class="working-dot"></span>
			<span class="working-label">Agent is working…</span>
		</div>
	{/if}
</div>

<style>
	.feed {
		flex: 1;
		overflow-y: auto;
		padding: 20px 20px;
	}

	.empty {
		font-size: 12px;
		color: rgba(196,181,253,0.3);
		text-align: center;
		padding-top: 32px;
	}

	.working-row {
		display: flex;
		align-items: center;
		gap: 5px;
		padding: 8px 4px 8px 16px;
		margin-top: 4px;
	}

	.working-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: rgba(139, 92, 246, 0.6);
		animation: bounce 1.2s infinite ease-in-out;
	}
	.working-dot:nth-child(2) { animation-delay: 0.2s; }
	.working-dot:nth-child(3) { animation-delay: 0.4s; }

	@keyframes bounce {
		0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
		40% { transform: translateY(-5px); opacity: 1; }
	}

	.working-label {
		font-size: 11.5px;
		color: rgba(167, 139, 250, 0.55);
		font-style: italic;
		margin-left: 4px;
	}
</style>
