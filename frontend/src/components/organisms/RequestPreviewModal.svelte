<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from '../atoms/Modal.svelte';
	import Button from '../atoms/Button.svelte';
	import { allWorkers } from '../../stores/workers';
	import { createPreviewBundle, listPreviewStacks } from '../../lib/apis/preview';
	import type {
		CreatePreviewBundleRequest,
		PreviewBundle,
		PreviewStack,
		PreviewStackRole
	} from '../../types/preview';
	import type { Worker } from '../../types/worker';

	export let jobId: string;
	export let open = false;

	const dispatch = createEventDispatcher<{
		close: void;
		created: PreviewBundle;
	}>();

	let loadingStacks = false;
	let submitting = false;
	let loadError = '';
	let submitError = '';
	let stacks: PreviewStack[] = [];
	let selectedStackId = '';
	let selectedStack: PreviewStack | undefined = undefined;
	let roleOverrides: Record<string, string> = {};
	let wasOpen = false;
	let requestToken = 0;

	function resetState() {
		loadingStacks = false;
		submitting = false;
		loadError = '';
		submitError = '';
		stacks = [];
		selectedStackId = '';
		selectedStack = undefined;
		roleOverrides = {};
	}

	function normalizeSelectedStack(stackId: string) {
		selectedStackId = stackId;
		selectedStack = stacks.find((stack) => stack.stack_id === stackId);
		const previous = roleOverrides;
		const next: Record<string, string> = {};
		for (const role of selectedStack?.roles ?? []) {
			next[role.role] = previous[role.role] ?? '';
		}
		roleOverrides = next;
	}

	function formatError(error: unknown, fallback: string): string {
		return error instanceof Error ? error.message : fallback;
	}

	function formatRoleLabel(role: string): string {
		return `${role.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())} Builder`;
	}

	function workersForRole(role: PreviewStackRole): Worker[] {
		return $allWorkers
			.filter((worker) => worker.group_name === role.worker_group)
			.sort((a, b) => a.name.localeCompare(b.name));
	}

	function handleStackChange(event: Event) {
		const value = (event.currentTarget as HTMLSelectElement).value;
		normalizeSelectedStack(value);
	}

	function handleRoleOverrideChange(role: string, event: Event) {
		const value = (event.currentTarget as HTMLSelectElement).value;
		roleOverrides = { ...roleOverrides, [role]: value };
	}

	function closeModal() {
		open = false;
		dispatch('close');
	}

	async function initializeModal() {
		const token = ++requestToken;
		loadingStacks = true;
		submitting = false;
		loadError = '';
		submitError = '';
		stacks = [];
		selectedStackId = '';
		selectedStack = undefined;
		roleOverrides = {};

		try {
			const loadedStacks = await listPreviewStacks();
			if (token !== requestToken) return;
			stacks = loadedStacks;
			if (loadedStacks.length === 0) {
				loadError = 'No preview stacks are configured.';
				return;
			}
			normalizeSelectedStack(loadedStacks[0].stack_id);
		} catch (error: unknown) {
			if (token !== requestToken) return;
			loadError = formatError(error, 'Failed to load preview stacks.');
		} finally {
			if (token === requestToken) {
				loadingStacks = false;
			}
		}
	}

	async function handleSubmit() {
		if (!selectedStack) return;

		submitting = true;
		submitError = '';

		const roleOverridesPayload = Object.entries(roleOverrides)
			.filter(([, workerId]) => workerId)
			.map(([role, worker_id]) => ({ role, worker_id }));

		const body: CreatePreviewBundleRequest = {
			stack_id: selectedStack.stack_id,
			task_id: jobId,
			role_overrides: roleOverridesPayload.length > 0 ? roleOverridesPayload : undefined
		};

		try {
			const bundle = await createPreviewBundle(body);
			dispatch('created', bundle);
			closeModal();
		} catch (error: unknown) {
			submitError = formatError(error, 'Failed to create preview bundle.');
		} finally {
			submitting = false;
		}
	}

	$: if (open && !wasOpen) {
		wasOpen = true;
		void initializeModal();
	} else if (!open && wasOpen) {
		wasOpen = false;
		requestToken += 1;
		resetState();
	}
</script>

{#if open}
	<Modal on:close={closeModal}>
		<div class="modal-header">
			<div>
				<p class="nx-label">Preview Bundle</p>
				<h2 class="modal-title">Request Preview</h2>
				<p class="modal-sub">Create a preview bundle from this job without committing or pushing code.</p>
			</div>
		</div>

		{#if loadError}
			<div class="error-box">{loadError}</div>
		{:else}
			<div class="form-shell">
				<div class="field">
					<label class="nx-label" for="preview-stack">Stack</label>
					<select
						id="preview-stack"
						class="nx-input"
						value={selectedStackId}
						on:change={handleStackChange}
						disabled={loadingStacks || stacks.length === 0}
					>
						{#each stacks as stack}
							<option value={stack.stack_id}>{stack.display_name} ({stack.stack_id})</option>
						{/each}
					</select>
				</div>

				{#if selectedStack}
					<div class="roles">
						{#each selectedStack.roles as role}
							{@const roleWorkers = workersForRole(role)}
							<div class="role-card nx-card">
								<div class="role-header">
									<div>
										<p class="nx-label">{formatRoleLabel(role.role)}</p>
										<p class="role-meta">
											Group: <code>{role.worker_group}</code>
										</p>
									</div>
								</div>

								<select
									class="nx-input"
									value={roleOverrides[role.role] ?? ''}
									on:change={(event) => handleRoleOverrideChange(role.role, event)}
									disabled={loadingStacks || submitting}
								>
									<option value="">Auto-pick</option>
									{#each roleWorkers as worker}
										<option value={worker.worker_id}>{worker.name} ({worker.worker_id})</option>
									{/each}
								</select>

								{#if roleWorkers.length === 0}
									<p class="role-note">No workers available - auto-pick will be used.</p>
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				{#if submitError}
					<div class="error-box">{submitError}</div>
				{/if}

				<div class="actions">
					<Button variant="ghost" on:click={closeModal} disabled={submitting}>Cancel</Button>
					<Button
						variant="primary"
						on:click={handleSubmit}
						disabled={loadingStacks || submitting || !selectedStack}
					>
						{submitting ? 'Requesting…' : 'Request Preview'}
					</Button>
				</div>
			</div>
		{/if}
	</Modal>
{/if}

<style>
	.modal-header {
		padding-right: 32px;
		margin-bottom: 18px;
	}

	.modal-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: #f0f0ff;
	}

	.modal-sub {
		margin-top: 6px;
		font-size: 13px;
		line-height: 1.5;
		color: rgba(196,181,253,0.45);
	}

	.form-shell {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.roles {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.role-card {
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.role-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 12px;
	}

	.role-meta {
		font-size: 12px;
		color: rgba(196,181,253,0.45);
	}

	.role-meta code {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		color: rgba(240,240,255,0.8);
	}

	.role-note {
		font-size: 12px;
		line-height: 1.5;
		color: rgba(196,181,253,0.52);
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

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding-top: 4px;
	}
</style>
