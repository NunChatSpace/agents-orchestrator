<script lang="ts">
	import { page } from '$app/stores';
	import { onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { activeJobId, activeJob, activeMessages, composerEnabled, appendMessage } from '../../../../stores/activeJob';
	import { upsertJob, removeJob } from '../../../../stores/jobs';
	import { selectedWorker } from '../../../../stores/selectedWorker';
	import { getJob, cancelJob, deleteJob, submitJob, finishJob } from '../../../../lib/apis/jobs';
	import { listMessages, sendMessage } from '../../../../lib/apis/messages';
	import TopBar from '../../../../components/organisms/TopBar.svelte';
	import MessageFeed from '../../../../components/organisms/MessageFeed.svelte';
	import ChangesPanel from '../../../../components/organisms/ChangesPanel.svelte';
	import Composer from '../../../../components/organisms/Composer.svelte';
	import Spinner from '../../../../components/atoms/Spinner.svelte';

	let loading = true;
	let error = '';
	let sending = false;

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

	let currentJobId = '';
	$: if (jobId && jobId !== currentJobId) {
		currentJobId = jobId;
		load(jobId);
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
	.tab-count {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 16px;
		height: 16px;
		padding: 0 4px;
		border-radius: 8px;
		background: rgba(139,92,246,0.2);
		font-size: 10px;
		font-weight: 700;
		color: rgba(167,139,250,0.9);
	}
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
{/if}
