<script lang="ts">
	import { goto } from '$app/navigation';
	import NewAgentForm from '../../../../components/organisms/NewAgentForm.svelte';
</script>

<div class="page">
	<div class="page-inner">
		<div class="page-header">
			<div class="page-icon">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="8" r="4"/>
					<path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
				</svg>
			</div>
			<div>
				<h1 class="page-title">New Agent</h1>
				<p class="page-sub">Register a worker agent with a git repository</p>
			</div>
		</div>
		<div class="form-card nx-card">
			<NewAgentForm on:created={() => goto('/')} on:cancel={() => goto('/')} />
		</div>
	</div>
</div>

<style>
	.page {
		flex: 1;
		overflow-y: auto;
		padding: 32px;
	}

	.page-inner {
		max-width: 600px;
		margin: 0 auto;
	}

	.page-header {
		display: flex;
		align-items: center;
		gap: 14px;
		margin-bottom: 24px;
	}

	.page-icon {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		background: rgba(139,92,246,0.1);
		border: 1px solid rgba(139,92,246,0.22);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a78bfa;
		flex-shrink: 0;
	}

	.page-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0f0ff;
	}

	.page-sub {
		font-size: 12.5px;
		color: rgba(196,181,253,0.4);
		margin-top: 2px;
	}

	.form-card {
		padding: 28px;
	}
</style>
