<script lang="ts">
	import { page } from '$app/stores';
	import { onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { activeJobId, activeJob, activeMessages, composerEnabled, appendMessage } from '../../../../stores/activeJob';
	import { upsertJob, removeJob } from '../../../../stores/jobs';
	import { selectedWorker } from '../../../../stores/selectedWorker';
	import { getJob, cancelJob, deleteJob, submitJob, finishJob } from '../../../../lib/apis/jobs';
	import { listMessages, sendMessage } from '../../../../lib/apis/messages';
	import { listPreviewBundles } from '../../../../lib/apis/preview';
	import Button from '../../../../components/atoms/Button.svelte';
	import TopBar from '../../../../components/organisms/TopBar.svelte';
	import MessageFeed from '../../../../components/organisms/MessageFeed.svelte';
	import ChangesPanel from '../../../../components/organisms/ChangesPanel.svelte';
	import Composer from '../../../../components/organisms/Composer.svelte';
	import RequestPreviewModal from '../../../../components/organisms/RequestPreviewModal.svelte';
	import PreviewStatusPanel from '../../../../components/molecules/PreviewStatusPanel.svelte';
	import Spinner from '../../../../components/atoms/Spinner.svelte';
	import type { PreviewBundle } from '../../../../types/preview';

	let loading = true;
	let error = '';
	let sending = false;
	let previewModalOpen = false;
	let activeBundle: PreviewBundle | undefined = undefined;
	let previewLoadToken = 0;

	$: jobId = $page.params.job_id;

	async function load(id: string) {
		loading = true;
		error = '';
		activeJobId.set(id);
		try {
			const [job, msgs] = await Promise.all([getJob(id), listMessages(id)]);
			activeJob.set(job);
			activeMessages.set(msgs);
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Failed to load job';
		} finally {
			loading = false;
		}
	}

	async function loadPreviewBundle(id: string) {
		const token = ++previewLoadToken;
		activeBundle = undefined;
		try {
			const bundles = await listPreviewBundles();
			if (token !== previewLoadToken || currentJobId !== id) return;
			activeBundle = bundles.find((bundle) => bundle.task_id === id && bundle.status !== 'destroyed');
		} catch {
			// ignore — preview panel is best-effort only
		}
	}

	let currentJobId = '';
	$: if (jobId && jobId !== currentJobId) {
		currentJobId = jobId;
		load(jobId);
		loadPreviewBundle(jobId);
	}

	onDestroy(() => {
		activeJobId.set(null);
		activeJob.set(null);
		activeMessages.set([]);
	});

	async function handleCancel() {
		if (!$activeJob) return;
		try {
			const updated = await cancelJob($activeJob.job_id);
			activeJob.set(updated);
			upsertJob(updated);
		} catch {
			// ignore — WS will update state if backend succeeds
		}
	}

	async function handleClose() {
		if (!$activeJob) return;
		try {
			await deleteJob($activeJob.job_id);
			removeJob($activeJob.job_id);
			goto('/');
		} catch {
			// ignore
		}
	}

	async function handleFinish() {
		if (!$activeJob) return;
		try {
			const updated = await finishJob($activeJob.job_id);
			activeJob.set(updated);
			upsertJob(updated);
			goto('/');
		} catch {
			// ignore
		}
	}

	async function handleChoose(e: CustomEvent<string>) {
		await handleSend(new CustomEvent('send', { detail: e.detail }));
	}

	async function handleSend(e: CustomEvent<string>) {
		if (!$activeJob) return;
		sending = true;
		try {
			// If job is still a draft, submit it to the selected worker first
			if ($activeJob.status === 'draft' && $selectedWorker) {
				const submitted = await submitJob($activeJob.job_id, $selectedWorker.worker_id);
				activeJob.set(submitted);
				upsertJob(submitted);
			}
			const msg = await sendMessage($activeJob.job_id, e.detail);
			appendMessage(msg);
			const updated = await getJob($activeJob.job_id);
			activeJob.set(updated);
			upsertJob(updated);
		} catch {
			// ignore
		} finally {
			sending = false;
		}
	}

	let activeTab: 'chat' | 'changes' = 'chat';

	function switchTab(tab: 'chat' | 'changes') {
		activeTab = tab;
	}

	function handleBundleCreated(e: CustomEvent<PreviewBundle>) {
		previewLoadToken += 1;
		activeBundle = e.detail;
		previewModalOpen = false;
	}

	async function handleBundleDestroyed() {
		if (!jobId) return;
		await loadPreviewBundle(jobId);
	}
</script>

<style>
	.job-shell {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.job-inner {
		max-width: 900px;
		width: 100%;
		margin: 0 auto;
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.tab-bar {
		display: flex;
		border-bottom: 1px solid rgba(139,92,246,0.15);
		flex-shrink: 0;
	}
	.tab {
		padding: 8px 20px;
		font-size: 12.5px;
		font-weight: 500;
		color: rgba(196,181,253,0.45);
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: color 0.15s, border-color 0.15s;
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.tab.active {
		color: rgba(167,139,250,0.95);
		border-bottom-color: rgba(139,92,246,0.7);
	}
	.tab:hover:not(.active) { color: rgba(196,181,253,0.75); }
	.tab-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.chat-panel {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.preview-strip {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 14px 20px;
		border-bottom: 1px solid rgba(139,92,246,0.15);
		background: rgba(139,92,246,0.03);
	}

	.preview-strip-copy {
		min-width: 0;
	}

	.preview-strip-title {
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.45);
	}

	.preview-strip-sub {
		margin-top: 4px;
		font-size: 12.5px;
		line-height: 1.5;
		color: rgba(196,181,253,0.62);
	}

	.preview-panel-wrap {
		padding: 16px 20px 0;
	}

	@media (max-width: 640px) {
		.preview-strip {
			flex-direction: column;
			align-items: stretch;
		}
	}
</style>

{#if loading}
	<div class="flex-1 flex items-center justify-center">
		<Spinner />
	</div>
{:else if error}
	<div class="flex-1 flex items-center justify-center text-red-400 text-sm">{error}</div>
{:else if $activeJob}
	<div class="job-shell">
		<div class="job-inner">
			<TopBar job={$activeJob} on:cancel={handleCancel} on:close={handleClose} on:finish={handleFinish} />
			<div class="preview-strip">
				<div class="preview-strip-copy">
					<p class="preview-strip-title">Preview</p>
					<p class="preview-strip-sub">
						{#if activeBundle}
							Showing bundle for stack <strong>{activeBundle.stack_id}</strong>.
						{:else}
							Request a preview bundle from the current job workspace state.
						{/if}
					</p>
				</div>
				<Button variant="secondary" on:click={() => (previewModalOpen = true)}>Request Preview</Button>
			</div>
			{#if activeBundle}
				<div class="preview-panel-wrap">
					<PreviewStatusPanel bundle={activeBundle} on:destroyed={handleBundleDestroyed} />
				</div>
			{/if}
			<div class="tab-bar">
				<button class="tab {activeTab === 'chat' ? 'active' : ''}" on:click={() => switchTab('chat')}>
					Chat
				</button>
				<button class="tab {activeTab === 'changes' ? 'active' : ''}" on:click={() => switchTab('changes')}>
					Changes
				</button>
			</div>
			<div class="tab-content">
				{#if activeTab === 'chat'}
					<div class="chat-panel">
						<MessageFeed messages={$activeMessages} status={$activeJob.status} on:choose={handleChoose} />
						<Composer
							enabled={$composerEnabled}
							loading={sending}
							disabledMessage={
								$activeJob?.status === 'draft' ? 'Select an agent in the sidebar first.' :
								$activeJob?.status === 'failed' ? 'Job failed — see messages above.' :
								$activeJob?.status === 'cancelled' ? 'Job was cancelled.' :
								'Waiting for worker response…'
							}
							on:send={handleSend}
						/>
					</div>
				{:else}
					<ChangesPanel jobId={$activeJob.job_id} />
				{/if}
			</div>
		</div>
	</div>
	<RequestPreviewModal bind:open={previewModalOpen} jobId={jobId ?? ''} on:created={handleBundleCreated} />
{/if}
