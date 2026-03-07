<script lang="ts">
	export let variant: 'primary' | 'secondary' | 'danger' | 'ghost' = 'primary';
	export let disabled = false;
	export let type: 'button' | 'submit' = 'button';
</script>

<button
	{type}
	{disabled}
	class="btn btn-{variant} {disabled ? 'disabled' : ''}"
	on:click
>
	<slot />
</button>

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 7px 14px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		border: none;
		transition: box-shadow 0.22s, transform 0.22s, background 0.18s, opacity 0.18s;
		letter-spacing: 0.01em;
		white-space: nowrap;
	}

	.btn-primary {
		background: linear-gradient(135deg, #7c3aed, #6d28d9);
		color: #fff;
		box-shadow: 0 0 18px rgba(124,58,237,0.32);
	}
	.btn-primary:hover:not(.disabled) {
		box-shadow: 0 0 28px rgba(124,58,237,0.55);
		transform: translateY(-1px);
	}

	.btn-secondary {
		background: transparent;
		color: #a78bfa;
		border: 1px solid rgba(139,92,246,0.40);
	}
	.btn-secondary:hover:not(.disabled) {
		background: rgba(139,92,246,0.07);
		box-shadow: 0 0 12px rgba(124,58,237,0.12);
	}

	.btn-danger {
		background: transparent;
		color: #fca5a5;
		border: 1px solid rgba(239,68,68,0.35);
	}
	.btn-danger:hover:not(.disabled) {
		background: rgba(239,68,68,0.08);
	}

	.btn-ghost {
		background: transparent;
		color: rgba(196,181,253,0.5);
		border: 1px solid transparent;
	}
	.btn-ghost:hover:not(.disabled) {
		background: rgba(139,92,246,0.06);
		color: #a78bfa;
	}

	.disabled {
		opacity: 0.45;
		cursor: not-allowed;
		transform: none !important;
		box-shadow: none !important;
	}
</style>
