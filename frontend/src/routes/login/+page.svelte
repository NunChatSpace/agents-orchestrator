<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '../../lib/apis/auth';
	import { user } from '../../stores/auth';
	import Button from '../../components/atoms/Button.svelte';

	let username = '';
	let password = '';
	let error = '';
	let loading = false;

	async function handleLogin() {
		error = '';
		loading = true;
		try {
			const u = await login(username, password);
			user.set(u);
			goto('/');
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login-page">
	<form class="login-card" on:submit|preventDefault={handleLogin}>
		<!-- Brand -->
		<div class="brand">
			<div class="brand-icon">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.2">
					<polygon points="3 11 22 2 13 21 11 13 3 11"/>
				</svg>
			</div>
			<div>
				<div class="brand-name">NEX<span>US</span></div>
				<div class="brand-sub">Agent Orchestrator</div>
			</div>
		</div>

		<div class="card-divider"></div>

		{#if error}
			<p class="error-msg">{error}</p>
		{/if}

		<div class="field">
			<label class="field-label">Username</label>
			<input
				type="text"
				bind:value={username}
				autocomplete="username"
				class="nx-input"
			/>
		</div>

		<div class="field">
			<label class="field-label">Password</label>
			<input
				type="password"
				bind:value={password}
				autocomplete="current-password"
				class="nx-input"
			/>
		</div>

		<Button variant="primary" type="submit" disabled={loading}>
			{loading ? 'Signing in…' : 'Sign in'}
		</Button>
	</form>
</div>

<style>
	.login-page {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		z-index: 10;
	}

	.login-card {
		width: 100%;
		max-width: 360px;
		display: flex;
		flex-direction: column;
		gap: 18px;
		padding: 32px;
		background: rgba(13,13,30,0.75);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(139,92,246,0.22);
		border-radius: 16px;
		position: relative;
		overflow: hidden;
		box-shadow: 0 0 60px rgba(124,58,237,0.08), 0 24px 48px rgba(0,0,0,0.4);
	}

	/* Top glow line */
	.login-card::before {
		content: '';
		position: absolute;
		top: 0; left: 0; right: 0;
		height: 1px;
		background: linear-gradient(90deg, transparent, rgba(139,92,246,0.5), transparent);
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.brand-icon {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		background: linear-gradient(135deg, #7c3aed, #4f46e5);
		box-shadow: 0 0 22px rgba(124,58,237,0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.brand-name {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 17px;
		font-weight: 700;
		letter-spacing: 0.14em;
		color: #f0f0ff;
	}
	.brand-name :global(span) { color: #a78bfa; }

	.brand-sub {
		font-size: 10px;
		color: rgba(196,181,253,0.38);
		letter-spacing: 0.1em;
		text-transform: uppercase;
		margin-top: 1px;
	}

	.card-divider {
		height: 1px;
		background: rgba(139,92,246,0.15);
		margin: 0 -4px;
	}

	.error-msg {
		font-size: 13px;
		color: #fca5a5;
		background: rgba(239,68,68,0.08);
		border: 1px solid rgba(239,68,68,0.2);
		border-radius: 7px;
		padding: 8px 12px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.field-label {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.09em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.5);
	}
</style>
