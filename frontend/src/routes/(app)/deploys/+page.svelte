<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import Button from '../../../components/atoms/Button.svelte';
	import DeployPlanCard from '../../../components/molecules/DeployPlanCard.svelte';
	import CreateDeployPlanModal from '../../../components/organisms/CreateDeployPlanModal.svelte';
	import {
		deleteDeploymentPlan,
		listDeploymentPlans,
		retryDeploymentPlan,
		stopDeploymentPlan
	} from '../../../lib/apis/deploymentPlans';
	import type { DeploymentPlan, DeploymentPlanStatus } from '../../../types/deploymentPlan';

	type PlanAction = 'stop' | 'retry' | 'delete' | null;

	const activeStatuses = new Set<DeploymentPlanStatus>(['pending', 'deploying']);

	let plans: DeploymentPlan[] = [];
	let createModalOpen = false;
	let loading = false;
	let error = '';
	let loadToken = 0;
	let pollHandle: ReturnType<typeof setInterval> | null = null;
	let pendingActions: Record<string, Exclude<PlanAction, null>> = {};

	function formatError(error: unknown, fallback: string): string {
		return error instanceof Error ? error.message : fallback;
	}

	function setPendingAction(id: string, action: Exclude<PlanAction, null> | null) {
		const next = { ...pendingActions };
		if (action) {
			next[id] = action;
		} else {
			delete next[id];
		}
		pendingActions = next;
	}

	function planRank(status: DeploymentPlanStatus): number {
		switch (status) {
			case 'running':
				return 0;
			case 'deploying':
				return 1;
			case 'pending':
				return 2;
			case 'failed':
				return 3;
			case 'stopped':
				return 4;
		}
	}

	function needsPolling(nextPlans: DeploymentPlan[] = plans): boolean {
		return nextPlans.some((plan) => activeStatuses.has(plan.status));
	}

	function stopPolling() {
		if (!pollHandle) return;
		clearInterval(pollHandle);
		pollHandle = null;
	}

	function startPolling() {
		if (pollHandle || typeof window === 'undefined') return;
		pollHandle = window.setInterval(() => {
			void loadPlans({ silent: true });
		}, 5000);
	}

	function syncPolling() {
		if (needsPolling()) {
			startPolling();
		} else {
			stopPolling();
		}
	}

	async function loadPlans(options: { silent?: boolean } = {}) {
		const token = ++loadToken;
		if (!options.silent) {
			loading = true;
		}

		try {
			const nextPlans = await listDeploymentPlans();
			if (token !== loadToken) return;
			plans = nextPlans;
			error = '';
		} catch (err: unknown) {
			if (token !== loadToken) return;
			error = formatError(err, 'Failed to load deployment plans.');
		} finally {
			if (token === loadToken && !options.silent) {
				loading = false;
			}
		}
	}

	function handleCreated(event: CustomEvent<DeploymentPlan>) {
		const created = event.detail;
		plans = [created, ...plans.filter((plan) => plan.id !== created.id)];
		createModalOpen = false;
		error = '';
	}

	async function handleStop(event: CustomEvent<string>) {
		const id = event.detail;
		setPendingAction(id, 'stop');
		error = '';

		try {
			const updated = await stopDeploymentPlan(id);
			plans = plans.map((plan) => (plan.id === id ? updated : plan));
		} catch (err: unknown) {
			error = formatError(err, 'Failed to stop deployment plan.');
		} finally {
			setPendingAction(id, null);
		}
	}

	async function handleRetry(event: CustomEvent<string>) {
		const id = event.detail;
		setPendingAction(id, 'retry');
		error = '';

		try {
			const updated = await retryDeploymentPlan(id);
			plans = plans.map((plan) => (plan.id === id ? updated : plan));
		} catch (err: unknown) {
			error = formatError(err, 'Failed to retry deployment plan.');
		} finally {
			setPendingAction(id, null);
		}
	}

	async function handleDelete(event: CustomEvent<string>) {
		const id = event.detail;
		setPendingAction(id, 'delete');
		error = '';

		try {
			await deleteDeploymentPlan(id);
			plans = plans.filter((plan) => plan.id !== id);
		} catch (err: unknown) {
			error = formatError(err, 'Failed to delete deployment plan.');
		} finally {
			setPendingAction(id, null);
		}
	}

	$: sortedPlans = [...plans].sort((a, b) => {
		const rankDiff = planRank(a.status) - planRank(b.status);
		if (rankDiff !== 0) return rankDiff;
		return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
	});

	$: syncPolling();

	onMount(() => {
		void loadPlans();
		return () => stopPolling();
	});

	onDestroy(() => {
		stopPolling();
	});
</script>

<div class="page">
	<div class="page-inner">
		<div class="page-header">
			<div>
				<p class="nx-label">Preview Runtime</p>
				<h1 class="page-title">Deployments</h1>
				<p class="page-sub">Create, monitor, stop, and remove named deployment plans.</p>
			</div>
			<Button variant="primary" on:click={() => (createModalOpen = true)}>New Plan</Button>
		</div>

		{#if error}
			<div class="error-box">{error}</div>
		{/if}

		{#if loading}
			<div class="empty-card nx-card">
				<p class="empty-copy">Loading deployment plans…</p>
			</div>
		{:else if sortedPlans.length === 0}
			<div class="empty-card nx-card">
				<p class="empty-copy">No deployment plans yet.</p>
			</div>
		{:else}
			<div class="plans-list">
				{#each sortedPlans as plan (plan.id)}
					<DeployPlanCard
						{plan}
						pendingAction={pendingActions[plan.id] ?? null}
						on:stop={handleStop}
						on:retry={handleRetry}
						on:delete={handleDelete}
					/>
				{/each}
			</div>
		{/if}
	</div>
</div>

<CreateDeployPlanModal bind:open={createModalOpen} on:created={handleCreated} />

<style>
	.page {
		flex: 1;
		overflow-y: auto;
		padding: 32px;
	}

	.page-inner {
		max-width: 1080px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.page-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
	}

	.page-title {
		font-family: 'Space Grotesk', system-ui, sans-serif;
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		color: var(--nx-white);
	}

	.page-sub {
		margin-top: 6px;
		font-size: 13px;
		line-height: 1.5;
		color: var(--nx-muted);
	}

	.plans-list {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.empty-card {
		padding: 24px;
	}

	.empty-copy {
		font-size: 14px;
		color: rgba(240,240,255,0.8);
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

	@media (max-width: 720px) {
		.page {
			padding: 20px;
		}

		.page-header {
			flex-direction: column;
			align-items: stretch;
		}

		.page-header :global(.btn) {
			width: 100%;
			justify-content: center;
		}
	}
</style>
