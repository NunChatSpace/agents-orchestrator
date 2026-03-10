<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Badge from '../atoms/Badge.svelte';
	import Button from '../atoms/Button.svelte';
	import { destroyPreviewBundle } from '../../lib/apis/preview';
	import type {
		PreviewBundle,
		PreviewBundleRoleState,
		PreviewBundleStatus,
		PreviewRoleBuildStatus
	} from '../../types/preview';

	type BadgeColor = 'gray' | 'blue' | 'yellow' | 'green' | 'red' | 'purple' | 'orange';

	const bundleStatusColors: Record<PreviewBundleStatus, BadgeColor> = {
		pending_build: 'gray',
		building: 'yellow',
		ready_to_deploy: 'blue',
		deploying: 'purple',
		healthy: 'green',
		failed: 'red',
		destroyed: 'gray'
	};

	const roleStatusColors: Record<PreviewRoleBuildStatus, BadgeColor> = {
		requested: 'gray',
		ready: 'green',
		failed: 'red'
	};

	export let bundle: PreviewBundle;

	const dispatch = createEventDispatcher<{ destroyed: void }>();

	let destroying = false;
	let destroyError = '';

	function formatLabel(value: string): string {
		return value.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
	}

	function formatAssignedWorker(role: PreviewBundleRoleState): string {
		return role.assigned_worker_id ?? 'auto';
	}

	function truncateMiddle(value?: string, left = 12, right = 10): string {
		if (!value) return 'n/a';
		if (value.length <= left + right + 3) return value;
		return `${value.slice(0, left)}...${value.slice(-right)}`;
	}

	function formatError(error: unknown, fallback: string): string {
		return error instanceof Error ? error.message : fallback;
	}

	async function handleDestroy() {
		destroying = true;
		destroyError = '';
		try {
			await destroyPreviewBundle(bundle.id);
			dispatch('destroyed');
		} catch (error: unknown) {
			destroyError = formatError(error, 'Failed to destroy preview bundle.');
		} finally {
			destroying = false;
		}
	}
</script>

<section class="preview-panel nx-card">
	<div class="panel-header">
		<div>
			<p class="nx-label">Preview Bundle</p>
			<h3 class="panel-title">{bundle.stack_id}</h3>
			<p class="panel-sub">Bundle ID: <code>{bundle.id}</code></p>
		</div>
		<Badge color={bundleStatusColors[bundle.status]}>{formatLabel(bundle.status)}</Badge>
	</div>

	{#if bundle.status === 'healthy' && bundle.preview_url}
		<div class="preview-link-block">
			<p class="nx-label">Preview URL</p>
			<a href={bundle.preview_url} target="_blank" rel="noreferrer" class="preview-link">
				{bundle.preview_url}
			</a>
		</div>
	{/if}

	<div class="roles">
		{#each bundle.roles as role}
			<div class="role-row">
				<div class="role-main">
					<div class="role-heading">
						<span class="role-name">{formatLabel(role.role)}</span>
						<Badge color={roleStatusColors[role.build_status]}>{formatLabel(role.build_status)}</Badge>
					</div>
					<div class="role-meta">
						<span>Worker: <code>{formatAssignedWorker(role)}</code></span>
						<span>Digest: <code>{truncateMiddle(role.image_digest)}</code></span>
					</div>
					{#if role.error_message}
						<p class="role-error">{role.error_message}</p>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if destroyError}
		<div class="error-box">{destroyError}</div>
	{/if}

	{#if bundle.status !== 'destroyed'}
		<div class="panel-actions">
			<Button variant="danger" on:click={handleDestroy} disabled={destroying}>
				{destroying ? 'Destroying…' : 'Destroy'}
			</Button>
		</div>
	{/if}
</section>

<style>
	.preview-panel {
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.panel-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
	}

	.panel-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: #f0f0ff;
	}

	.panel-sub {
		margin-top: 4px;
		font-size: 12px;
		line-height: 1.5;
		color: rgba(196,181,253,0.45);
	}

	.panel-sub code,
	.role-meta code {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		color: rgba(240,240,255,0.82);
	}

	.preview-link-block {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.preview-link {
		color: #c4b5fd;
		font-size: 13px;
		word-break: break-all;
		text-decoration: none;
	}

	.preview-link:hover {
		text-decoration: underline;
	}

	.roles {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.role-row {
		border: 1px solid rgba(139,92,246,0.12);
		border-radius: 12px;
		background: rgba(139,92,246,0.03);
		padding: 14px;
	}

	.role-main {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.role-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}

	.role-name {
		font-size: 14px;
		font-weight: 600;
		color: rgba(240,240,255,0.92);
	}

	.role-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
		font-size: 12px;
		line-height: 1.5;
		color: rgba(196,181,253,0.48);
	}

	.role-error {
		font-size: 12px;
		line-height: 1.5;
		color: #fecaca;
	}

	.error-box {
		border-radius: 12px;
		border: 1px solid rgba(239,68,68,0.28);
		background: rgba(239,68,68,0.09);
		padding: 12px 14px;
		font-size: 13px;
		line-height: 1.5;
		color: #fecaca;
	}

	.panel-actions {
		display: flex;
		justify-content: flex-end;
	}
</style>
