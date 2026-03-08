<script lang="ts">
	import { pingWorker, updateWorker } from '../../../../../lib/apis/workers';
	import { selectedWorker } from '../../../../../stores/selectedWorker';
	import { upsertWorker } from '../../../../../stores/workers';
	import type { PingResult } from '../../../../../types/worker';

	let pinging = false;
	let pingResult: PingResult | null = null;
	let savingPosition = false;
	let mapX = '0';
	let mapY = '0';
	let positionError = '';
	let positionMessage = '';
	let positionWorkerId = '';

	$: if ($selectedWorker && $selectedWorker.worker_id !== positionWorkerId) {
		positionWorkerId = $selectedWorker.worker_id;
		mapX = String($selectedWorker.map_x ?? 0);
		mapY = String($selectedWorker.map_y ?? 0);
		positionError = '';
		positionMessage = '';
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

	async function runPing() {
		if (!$selectedWorker) return;
		pinging = true;
		pingResult = null;
		try {
			pingResult = await pingWorker($selectedWorker.worker_id);
		} catch {
			pingResult = { ok: false, output: '', error: 'Request failed' };
		} finally {
			pinging = false;
		}
	}

	async function savePosition() {
		if (!$selectedWorker) return;
		const nextX = Number.parseInt(mapX, 10);
		const nextY = Number.parseInt(mapY, 10);
		positionError = '';
		positionMessage = '';

		if (!Number.isInteger(nextX) || !Number.isInteger(nextY)) {
			positionError = 'X and Y must be integers';
			return;
		}
		if (nextX < 0 || nextY < 0) {
			positionError = 'X and Y must be 0 or greater';
			return;
		}

		savingPosition = true;
		try {
			const updated = await updateWorker($selectedWorker.worker_id, { map_x: nextX, map_y: nextY });
			selectedWorker.set(updated);
			upsertWorker(updated);
			mapX = String(updated.map_x);
			mapY = String(updated.map_y);
			positionMessage = 'Office position saved';
		} catch (e: unknown) {
			positionError = e instanceof Error ? e.message : 'Failed to save office position';
		} finally {
			savingPosition = false;
		}
	}
</script>

{#if $selectedWorker}
	<div class="page">
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
					<div class="info-value mono">{$selectedWorker.cli_command}</div>
				</div>
				<div class="info-card">
					<div class="info-label">Last Active</div>
					<div class="info-value">{$selectedWorker.last_active_at ? relativeTime($selectedWorker.last_active_at) : 'Never'}</div>
				</div>
				<div class="info-card wide">
					<div class="info-label">Workspace</div>
					<div class="info-value mono">{$selectedWorker.workspace}</div>
				</div>
				<div class="info-card wide">
					<div class="info-label">Git Repository</div>
					<div class="info-value mono">{$selectedWorker.git_repo_url || '—'}</div>
				</div>
			</div>
		</div>

		<!-- Office Position -->
		<div class="section">
			<div class="section-header">
				<div class="section-title">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 2l3 6h6l-4.5 4.5L18 20l-6-3-6 3 1.5-7.5L3 8h6z"/>
					</svg>
					Office Position
				</div>
				<button class="ping-btn" on:click={savePosition} disabled={savingPosition}>
					{#if savingPosition}
						<span class="spinner"></span> Saving…
					{:else}
						Save Position
					{/if}
				</button>
			</div>

			<div class="position-grid">
				<div class="position-field">
					<label class="position-label" for="map-x">X tile</label>
					<input id="map-x" class="nx-input" type="number" min="0" step="1" bind:value={mapX} />
				</div>
				<div class="position-field">
					<label class="position-label" for="map-y">Y tile</label>
					<input id="map-y" class="nx-input" type="number" min="0" step="1" bind:value={mapY} />
				</div>
			</div>

			{#if positionError}
				<p class="position-error">{positionError}</p>
			{:else if positionMessage}
				<p class="position-ok">{positionMessage}</p>
			{:else}
				<p class="position-hint">Default fallback is auto-placement by group when both values are 0.</p>
			{/if}
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
				<p class="ping-hint">Runs <code>{$selectedWorker.cli_command} --version</code> inside the agent's workspace to verify the CLI is installed and reachable.</p>
			{/if}
		</div>
	</div>
{/if}

<style>
	.page {
		padding: 24px 32px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		max-width: 760px;
		margin: 0 auto;
	}

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

	.position-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}

	.position-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.position-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.38);
	}

	.position-hint {
		font-size: 12px;
		color: rgba(196,181,253,0.34);
	}

	.position-error {
		font-size: 12.5px;
		color: #fca5a5;
	}

	.position-ok {
		font-size: 12.5px;
		color: #4ade80;
	}

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
</style>
