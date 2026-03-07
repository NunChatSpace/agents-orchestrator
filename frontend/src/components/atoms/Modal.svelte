<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher();

	function close() {
		dispatch('close');
	}

	function handleBackdrop(e: MouseEvent) {
		if (e.target === e.currentTarget) close();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}

	onMount(() => window.addEventListener('keydown', handleKeydown));
	onDestroy(() => window.removeEventListener('keydown', handleKeydown));
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="backdrop" on:click={handleBackdrop}>
	<div class="modal nx-card" role="dialog" aria-modal="true">
		<button class="close-btn" on:click={close} aria-label="Close">
			<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
			</svg>
		</button>
		<slot />
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.55);
		backdrop-filter: blur(6px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 200;
		padding: 24px;
	}

	.modal {
		position: relative;
		width: 100%;
		max-width: 560px;
		max-height: 90vh;
		overflow-y: auto;
		padding: 28px;
	}

	.close-btn {
		position: absolute;
		top: 14px;
		right: 14px;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		background: transparent;
		border: 1px solid rgba(139,92,246,0.2);
		color: rgba(196,181,253,0.45);
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		transition: background 0.18s, color 0.18s, border-color 0.18s;
	}
	.close-btn:hover {
		background: rgba(139,92,246,0.1);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.4);
	}
</style>
