<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Badge from '../atoms/Badge.svelte';
	import Button from '../atoms/Button.svelte';
	import Modal from '../atoms/Modal.svelte';
	import { allWorkers } from '../../stores/workers';
	import { listPreviewStacks } from '../../lib/apis/preview';
	import { createDeploymentPlan } from '../../lib/apis/deploymentPlans';
	import type { PreviewStack, PreviewStackRole } from '../../types/preview';
	import type {
		CreateDeploymentPlanRequest,
		DeploymentPlan,
		DeploymentPlanBuildMode
	} from '../../types/deploymentPlan';
	import type { Worker, WorkerStatus } from '../../types/worker';

	type BadgeColor = 'gray' | 'blue' | 'yellow' | 'green' | 'red' | 'purple' | 'orange';

	const planNamePattern = /^[a-z0-9-]{1,40}$/;

	const workerStatusColors: Record<WorkerStatus, BadgeColor> = {
		busy: 'yellow',
		pending_user: 'orange',
		idle: 'green',
		offline: 'gray'
	};

	const workerStatusLabels: Record<WorkerStatus, string> = {
		busy: 'Busy',
		pending_user: 'Waiting',
		idle: 'Idle',
		offline: 'Offline'
	};

	export let open = false;

	const dispatch = createEventDispatcher<{
		close: void;
		created: DeploymentPlan;
	}>();

	let wasOpen = false;
	let requestToken = 0;
	let loadingStacks = false;
	let submitting = false;
	let loadError = '';
	let submitError = '';
	let planName = '';
	let selectedStackId = '';
	let selectedStack: PreviewStack | undefined;
	let buildMode: DeploymentPlanBuildMode = 'fresh';
	let stacks: PreviewStack[] = [];
	let roleAssignments: Record<string, string> = {};

	function formatError(error: unknown, fallback: string): string {
		return error instanceof Error ? error.message : fallback;
	}

	function formatRoleLabel(value: string): string {
		return value.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
	}

	function workerLabel(worker: Worker): string {
		return `${worker.name} · ${workerStatusLabels[worker.status]}`;
	}

	function workersForRole(role: PreviewStackRole): Worker[] {
		return $allWorkers
			.filter((worker) => worker.group_name === role.worker_group)
			.sort((a, b) => a.name.localeCompare(b.name));
	}

	function selectedWorkerForRole(role: PreviewStackRole): Worker | undefined {
		return workersForRole(role).find((worker) => worker.worker_id === roleAssignments[role.role]);
	}

	function resetState() {
		loadingStacks = false;
		submitting = false;
		loadError = '';
		submitError = '';
		planName = '';
		selectedStackId = '';
		selectedStack = undefined;
		buildMode = 'fresh';
		stacks = [];
		roleAssignments = {};
	}

	function closeModal() {
		open = false;
		dispatch('close');
	}

	function normalizeSelectedStack(stackId: string) {
		selectedStackId = stackId;
		selectedStack = stacks.find((stack) => stack.stack_id === stackId);
		const previous = roleAssignments;
		const next: Record<string, string> = {};
		for (const role of selectedStack?.roles ?? []) {
			next[role.role] = previous[role.role] ?? '';
		}
		roleAssignments = next;
	}

	function handleStackChange(event: Event) {
		normalizeSelectedStack((event.currentTarget as HTMLSelectElement).value);
	}

	function handleRoleAssignmentChange(role: string, event: Event) {
		roleAssignments = {
			...roleAssignments,
			[role]: (event.currentTarget as HTMLSelectElement).value
		};
	}

	async function initializeModal() {
		const token = ++requestToken;
		loadingStacks = true;
		submitting = false;
		loadError = '';
		submitError = '';
		planName = '';
		buildMode = 'fresh';
		stacks = [];
		selectedStackId = '';
		selectedStack = undefined;
		roleAssignments = {};

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
		if (!selectedStack || !canSubmit) return;

		const token = requestToken;
		submitting = true;
		submitError = '';

		const body: CreateDeploymentPlanRequest = {
			name: planName.trim(),
			stack_id: selectedStack.stack_id,
			build_mode: buildMode,
			roles: selectedStack.roles.map((role) => ({
				role: role.role,
				worker_id: roleAssignments[role.role]
			}))
		};

		try {
			const plan = await createDeploymentPlan(body);
			if (token !== requestToken) return;
			dispatch('created', plan);
			closeModal();
		} catch (error: unknown) {
			if (token !== requestToken) return;
			submitError = formatError(error, 'Failed to create deployment plan.');
		} finally {
			if (token === requestToken) {
				submitting = false;
			}
		}
	}

	$: trimmedPlanName = planName.trim();
	$: nameError =
		trimmedPlanName.length > 0 && !planNamePattern.test(trimmedPlanName)
			? 'Plan name must match [a-z0-9-]{1,40}.'
			: '';
	$: allRolesAssigned =
		(selectedStack?.roles ?? []).length > 0 &&
		(selectedStack?.roles ?? []).every((role) => Boolean(roleAssignments[role.role]));
	$: canSubmit =
		!loadingStacks &&
		!submitting &&
		Boolean(selectedStack) &&
		trimmedPlanName.length > 0 &&
		nameError === '' &&
		allRolesAssigned;

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
		<form class="form-shell" on:submit|preventDefault={handleSubmit}>
			<div class="modal-header">
				<div>
					<p class="nx-label">Deployment Plan</p>
					<h2 class="modal-title">New Plan</h2>
					<p class="modal-sub">Create a named preview deployment by assigning one agent to each required role.</p>
				</div>
			</div>

			{#if loadError}
				<div class="error-box">{loadError}</div>
			{:else}
				<div class="field">
					<label class="nx-label" for="deploy-plan-name">Plan Name</label>
					<input
						id="deploy-plan-name"
						class="nx-input"
						type="text"
						bind:value={planName}
						placeholder="e.g. feature-cart"
						autocomplete="off"
						disabled={loadingStacks || submitting}
					/>
					{#if nameError}
						<p class="field-error">{nameError}</p>
					{/if}
				</div>

				<div class="field">
					<label class="nx-label" for="deploy-stack">Stack</label>
					<select
						id="deploy-stack"
						class="nx-input"
						value={selectedStackId}
						on:change={handleStackChange}
						disabled={loadingStacks || submitting || stacks.length === 0}
					>
						{#each stacks as stack}
							<option value={stack.stack_id}>{stack.display_name} ({stack.stack_id})</option>
						{/each}
					</select>
				</div>

				<fieldset class="field fieldset-reset">
					<legend class="nx-label">Build Mode</legend>
					<div class="mode-toggle">
						<Button
							type="button"
							variant={buildMode === 'fresh' ? 'primary' : 'secondary'}
							on:click={() => (buildMode = 'fresh')}
							disabled={submitting}
						>
							Fresh
						</Button>
						<Button
							type="button"
							variant={buildMode === 'latest' ? 'primary' : 'secondary'}
							on:click={() => (buildMode = 'latest')}
							disabled={submitting}
						>
							Latest
						</Button>
					</div>
					<p class="field-note">
						{#if buildMode === 'fresh'}
							Force each assigned agent to build before deploy starts.
						{:else}
							Reuse each agent&apos;s latest ready build, with fresh fallback when none exists.
						{/if}
					</p>
				</fieldset>

				{#if selectedStack}
					<div class="roles">
						<div class="roles-head">
							<div>
								<p class="nx-label">Roles</p>
								<p class="roles-sub">Each role must be assigned to one agent from the required worker group.</p>
							</div>
						</div>

						{#each selectedStack.roles as role}
							{@const roleWorkers = workersForRole(role)}
							{@const selectedWorker = selectedWorkerForRole(role)}
							<div class="role-card nx-card">
								<div class="role-header">
									<div>
										<h3 class="role-title">{formatRoleLabel(role.role)}</h3>
										<p class="role-meta">
											Group: <code>{role.worker_group}</code>
										</p>
									</div>
									{#if selectedWorker}
										<Badge color={workerStatusColors[selectedWorker.status]}>
											{workerStatusLabels[selectedWorker.status]}
										</Badge>
									{/if}
								</div>

								<select
									class="nx-input"
									value={roleAssignments[role.role] ?? ''}
									on:change={(event) => handleRoleAssignmentChange(role.role, event)}
									disabled={submitting}
								>
									<option value="">Select agent</option>
									{#each roleWorkers as worker}
										<option value={worker.worker_id}>{workerLabel(worker)}</option>
									{/each}
								</select>

								{#if roleWorkers.length === 0}
									<p class="field-error">No agents are available in <code>{role.worker_group}</code>.</p>
								{:else}
									<div class="worker-list">
										{#each roleWorkers as worker}
											<div
												class="worker-row"
												class:selected={worker.worker_id === roleAssignments[role.role]}
											>
												<span>{worker.name}</span>
												<Badge color={workerStatusColors[worker.status]}>
													{workerStatusLabels[worker.status]}
												</Badge>
											</div>
										{/each}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				{#if submitError}
					<div class="error-box">{submitError}</div>
				{/if}
			{/if}

			<div class="actions">
				<Button type="button" variant="ghost" on:click={closeModal} disabled={submitting}>Cancel</Button>
				<Button type="submit" variant="primary" disabled={!canSubmit}>
					{submitting ? 'Deploying…' : 'Deploy'}
				</Button>
			</div>
		</form>
	</Modal>
{/if}

<style>
	.form-shell {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}

	.modal-header {
		padding-right: 32px;
	}

	.modal-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--nx-white);
	}

	.modal-sub {
		margin-top: 6px;
		font-size: 13px;
		line-height: 1.5;
		color: var(--nx-muted);
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.fieldset-reset {
		margin: 0;
		padding: 0;
		border: none;
	}

	.field-note,
	.roles-sub,
	.role-meta {
		font-size: 12px;
		line-height: 1.5;
		color: var(--nx-muted);
	}

	.field-error {
		font-size: 12px;
		line-height: 1.5;
		color: #fca5a5;
	}

	.mode-toggle {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
	}

	.roles {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.role-card {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 16px;
	}

	.role-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 12px;
	}

	.role-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--nx-white);
	}

	.role-meta code {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		color: rgba(240,240,255,0.82);
	}

	.worker-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.worker-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 10px 12px;
		border-radius: 10px;
		border: 1px solid rgba(139,92,246,0.12);
		background: rgba(139,92,246,0.04);
		font-size: 13px;
		color: rgba(240,240,255,0.92);
	}

	.worker-row.selected {
		border-color: var(--nx-borderb);
		background: rgba(139,92,246,0.08);
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
	}

	@media (max-width: 720px) {
		.role-header,
		.worker-row,
		.actions {
			flex-direction: column;
			align-items: stretch;
		}

		.actions :global(.btn) {
			width: 100%;
			justify-content: center;
		}
	}
</style>
