<script lang="ts">
	import Button from '../atoms/Button.svelte';
	import { createEventDispatcher } from 'svelte';
	import type { CreateJobPayload, TargetGroup } from '../../types/job';
	import { TARGET_GROUPS } from '../../types/job';
	import type { Worker } from '../../types/worker';

	export let workers: Worker[] = [];
	export let loading = false;
	export let initialTargetGroup: TargetGroup = 'fi-backend';
	export let initialWorkerOverride: string = '';

	let title = '';
	let prompt = '';
	let targetGroup: TargetGroup = initialTargetGroup;
	let manualWorkerOverride = initialWorkerOverride;

	const dispatch = createEventDispatcher<{
		draft: CreateJobPayload;
		submit: CreateJobPayload;
	}>();

	function buildPayload(): CreateJobPayload {
		return {
			title: title.trim() || undefined,
			prompt: prompt.trim(),
			target_group: targetGroup,
			manual_worker_override: manualWorkerOverride || undefined
		};
	}

	function handleDraft() {
		if (!prompt.trim()) return;
		dispatch('draft', buildPayload());
	}

	function handleSubmit() {
		if (!prompt.trim()) return;
		dispatch('submit', buildPayload());
	}
</script>

<form class="form" on:submit|preventDefault={handleSubmit}>
	<div class="field">
		<label class="field-label">Title <span class="optional">(optional)</span></label>
		<input
			type="text"
			bind:value={title}
			placeholder="Leave blank to auto-generate from prompt"
			class="nx-input"
		/>
	</div>

	<div class="field">
		<label class="field-label">Prompt <span class="required">*</span></label>
		<textarea
			rows="6"
			bind:value={prompt}
			placeholder="Describe the task for the worker agent…"
			class="nx-input nx-textarea"
		/>
	</div>

	<div class="field">
		<label class="field-label">Target group <span class="required">*</span></label>
		<select bind:value={targetGroup} class="nx-input">
			{#each TARGET_GROUPS as g}
				<option value={g}>{g}</option>
			{/each}
		</select>
	</div>

	{#if workers.length > 0}
	<div class="field">
		<label class="field-label">Worker override <span class="optional">(optional)</span></label>
		<select bind:value={manualWorkerOverride} class="nx-input">
			<option value="">Auto (LRU)</option>
			{#each workers.filter(w => w.group_name === targetGroup) as w}
				<option value={w.worker_id}>{w.name} ({w.status})</option>
			{/each}
		</select>
	</div>
	{/if}

	<div class="form-actions">
		<Button variant="secondary" disabled={loading} on:click={handleDraft}>Save Draft</Button>
		<Button variant="primary" type="submit" disabled={loading || !prompt.trim()}>Save & Submit</Button>
	</div>
</form>

<style>
	.form {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.field-label {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.5);
	}

	.optional { color: rgba(196,181,253,0.3); font-weight: 400; text-transform: none; letter-spacing: 0; }
	.required { color: rgba(239,68,68,0.7); font-weight: 400; }

	:global(.nx-textarea) {
		resize: vertical;
		line-height: 1.6;
	}

	.form-actions {
		display: flex;
		gap: 10px;
		justify-content: flex-end;
		padding-top: 4px;
	}
</style>
