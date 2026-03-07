<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Button from '../atoms/Button.svelte';
	import { createWorker } from '../../lib/apis/workers';
	import { TARGET_GROUPS } from '../../types/job';
	import type { CLICommand } from '../../types/worker';

	const dispatch = createEventDispatcher();

	let name = '';
	let gitRepoURL = '';
	let groupName = '';
	let cliCommand: CLICommand = 'codex';
	let loading = false;
	let error = '';

	const WORKSPACES_ROOT = '/workspaces';

	$: workspace = name.trim() ? `${WORKSPACES_ROOT}/${name.trim()}` : `${WORKSPACES_ROOT}/…`;
	$: cloneCmd = `git clone ${gitRepoURL.trim() || '<repo-url>'} ${workspace}`;
	$: runCmd = `${cliCommand} <prompt> --output-format json`;

	async function handleSubmit() {
		if (!name.trim() || !gitRepoURL.trim() || !groupName.trim()) return;
		loading = true;
		error = '';
		try {
			const worker = await createWorker({
				name: name.trim(),
				git_repo_url: gitRepoURL.trim(),
				group_name: groupName.trim(),
				cli_command: cliCommand
			});
			dispatch('created', worker);
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Failed to create agent';
		} finally {
			loading = false;
		}
	}
</script>

<form class="form" on:submit|preventDefault={handleSubmit}>
	{#if error}
		<p class="error-msg">{error}</p>
	{/if}

	<div class="field">
		<label class="field-label">Agent name <span class="required">*</span></label>
		<input type="text" bind:value={name} placeholder="e.g. my-backend-agent" class="nx-input" />
		<p class="field-hint">Workspace: <code>/workspaces/{name || '…'}</code></p>
	</div>

	<div class="field">
		<label class="field-label">Git repo URL <span class="required">*</span></label>
		<input type="text" bind:value={gitRepoURL} placeholder="https://github.com/org/repo.git" class="nx-input" />
		<p class="field-hint">Use a plain HTTPS URL — GIT_TOKEN is injected automatically.</p>
	</div>

	<div class="field">
		<label class="field-label">Group <span class="required">*</span></label>
		<input
			type="text"
			bind:value={groupName}
			placeholder="fi-backend"
			list="group-suggestions"
			class="nx-input"
		/>
		<datalist id="group-suggestions">
			{#each TARGET_GROUPS as g}
				<option value={g} />
			{/each}
		</datalist>
		<p class="field-hint">Pick an existing group or type a new name.</p>
	</div>

	<div class="field">
		<label class="field-label">CLI command <span class="required">*</span></label>
		<select bind:value={cliCommand} class="nx-input">
			<option value="codex">codex</option>
			<option value="claude">claude</option>
		</select>
	</div>

	<div class="preview-block">
		<p class="preview-label">Commands that will run</p>
		<div class="preview-lines">
			<div class="preview-line">
				<span class="cmd-step">1</span>
				<code class="cmd-text">{cloneCmd}</code>
			</div>
			<div class="preview-line">
				<span class="cmd-step">2</span>
				<code class="cmd-text">{runCmd}</code>
			</div>
		</div>
	</div>

	<div class="form-actions">
		<Button variant="secondary" type="button" disabled={loading} on:click={() => dispatch('cancel')}>
			Cancel
		</Button>
		<Button
			variant="primary"
			type="submit"
			disabled={loading || !name.trim() || !gitRepoURL.trim() || !groupName.trim()}
		>
			{loading ? 'Creating…' : 'Create Agent'}
		</Button>
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

	.required { color: rgba(239,68,68,0.7); font-weight: 400; }

	.field-hint {
		font-size: 11.5px;
		color: rgba(196,181,253,0.3);
		margin-top: 2px;
	}

	.field-hint code {
		font-family: 'SFMono-Regular', Consolas, monospace;
		background: rgba(139,92,246,0.1);
		padding: 1px 5px;
		border-radius: 4px;
		color: #a78bfa;
	}

	.error-msg {
		font-size: 13px;
		color: #fca5a5;
		background: rgba(239,68,68,0.08);
		border: 1px solid rgba(239,68,68,0.2);
		border-radius: 7px;
		padding: 8px 12px;
	}

	.preview-block {
		background: rgba(5,5,12,0.6);
		border: 1px solid rgba(139,92,246,0.18);
		border-radius: 10px;
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.preview-label {
		font-size: 9.5px;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.35);
	}

	.preview-lines {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.preview-line {
		display: flex;
		align-items: flex-start;
		gap: 10px;
	}

	.cmd-step {
		font-size: 10px;
		font-weight: 600;
		color: rgba(139,92,246,0.5);
		background: rgba(139,92,246,0.1);
		border-radius: 4px;
		padding: 2px 6px;
		flex-shrink: 0;
		margin-top: 1px;
		font-family: 'SFMono-Regular', Consolas, monospace;
	}

	.cmd-text {
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 12px;
		color: #a78bfa;
		word-break: break-all;
		line-height: 1.55;
	}

	.form-actions {
		display: flex;
		gap: 10px;
		justify-content: flex-end;
		padding-top: 4px;
	}
</style>
