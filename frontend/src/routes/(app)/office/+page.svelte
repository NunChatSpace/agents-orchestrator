<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import Modal from '../../../components/atoms/Modal.svelte';
	import NewJobForm from '../../../components/organisms/NewJobForm.svelte';
	import OfficeInteractionPanel from '../../../components/organisms/OfficeInteractionPanel.svelte';
	import { allWorkers } from '../../../stores/workers';
	import { allJobs, upsertJob } from '../../../stores/jobs';
	import { createJob, listJobs, submitJob } from '../../../lib/apis/jobs';
	import {
		MAP_CONFIG,
		buildZoneLabel,
		resolveDeskPosition,
		type GroupZoneLabel,
		type OfficeWorkerPlacement
	} from '../../../lib/office/mapConfig';
	import { OfficeScene } from '../../../lib/office/OfficeScene';
	import type { CreateJobPayload, Job } from '../../../types/job';
	import type { Worker } from '../../../types/worker';

	const ACTIVE_JOB_STATUSES = new Set(['assigned', 'busy', 'pending_user']);

	let viewportEl: HTMLDivElement | null = null;
	let stageEl: HTMLDivElement | null = null;
	let officeScene: OfficeScene | null = null;
	let resizeHandler: (() => void) | null = null;
	let keyHandler: ((event: KeyboardEvent) => void) | null = null;
	let clockTimer: ReturnType<typeof setInterval> | null = null;

	let nearestWorkerId: string | null = null;
	let panelWorkerId: string | null = null;
	let showJobModal = false;
	let jobLoading = false;
	let jobError = '';
	let createdJobId: string | null = null;
	let currentTimeText = '--:--:--';

	function updateClock() {
		const now = new Date();
		currentTimeText = [
			now.getHours().toString().padStart(2, '0'),
			now.getMinutes().toString().padStart(2, '0'),
			now.getSeconds().toString().padStart(2, '0')
		].join(':');
	}

	function byCreatedAtThenName(a: Worker, b: Worker): number {
		const aTime = new Date(a.created_at).getTime();
		const bTime = new Date(b.created_at).getTime();
		if (aTime !== bTime) return aTime - bTime;
		return a.name.localeCompare(b.name);
	}

	function buildOfficePlacements(workers: Worker[]): OfficeWorkerPlacement[] {
		const groups = new Map<string, Worker[]>();
		for (const worker of workers) {
			const workersInGroup = groups.get(worker.group_name) ?? [];
			workersInGroup.push(worker);
			groups.set(worker.group_name, workersInGroup);
		}

		const placements: OfficeWorkerPlacement[] = [];
		for (const workersInGroup of groups.values()) {
			const sorted = [...workersInGroup].sort(byCreatedAtThenName);
			for (const [index, worker] of sorted.entries()) {
				const desk = resolveDeskPosition(worker, index);
				placements.push({
					workerId: worker.worker_id,
					name: worker.name,
					groupName: worker.group_name,
					status: worker.status,
					mapX: desk.mapX,
					mapY: desk.mapY,
					worldX: desk.worldX,
					worldZ: desk.worldZ,
					accent: desk.accent
				});
			}
		}

		return placements;
	}

	function buildGroupLabels(workers: Worker[]): GroupZoneLabel[] {
		const groups = [...new Set(workers.map((worker) => worker.group_name))].sort();
		return groups.map((groupName) => buildZoneLabel(groupName));
	}

	function jobsForWorker(workerId: string): Job[] {
		return $allJobs
			.filter((job) => job.assigned_worker_id === workerId)
			.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
	}

	function onPanelNewJob() {
		if (!panelWorker) return;
		showJobModal = true;
		jobError = '';
	}

	function onPanelViewJobs() {
		if (!panelWorker) return;
		goto(`/agents/${panelWorker.worker_id}/jobs`);
	}

	function onPanelSettings() {
		if (!panelWorker) return;
		goto(`/agents/${panelWorker.worker_id}/settings`);
	}

	function isTypingTarget(target: EventTarget | null): boolean {
		return (
			target instanceof HTMLInputElement ||
			target instanceof HTMLTextAreaElement ||
			target instanceof HTMLSelectElement ||
			(target instanceof HTMLElement && target.isContentEditable)
		);
	}

	async function handleJobDraft(event: CustomEvent<CreateJobPayload>) {
		jobLoading = true;
		jobError = '';
		try {
			const created = await createJob(event.detail);
			upsertJob(created);
			createdJobId = created.job_id;
			showJobModal = false;
		} catch (error: unknown) {
			jobError = error instanceof Error ? error.message : 'Failed to create job';
		} finally {
			jobLoading = false;
		}
	}

	async function handleJobSubmit(event: CustomEvent<CreateJobPayload>) {
		if (!panelWorker) return;
		jobLoading = true;
		jobError = '';
		try {
			const created = await createJob(event.detail);
			upsertJob(created);
			const queued = await submitJob(created.job_id, panelWorker.worker_id);
			upsertJob(queued);
			createdJobId = created.job_id;
			showJobModal = false;
		} catch (error: unknown) {
			jobError = error instanceof Error ? error.message : 'Failed to submit job';
		} finally {
			jobLoading = false;
		}
	}

	$: officePlacements = buildOfficePlacements($allWorkers);
	$: groupLabels = buildGroupLabels($allWorkers);
	$: panelWorker = panelWorkerId ? $allWorkers.find((worker) => worker.worker_id === panelWorkerId) : null;
	$: panelJobs = panelWorkerId ? jobsForWorker(panelWorkerId) : [];
	$: panelActiveJob = panelJobs.find((job) => ACTIVE_JOB_STATUSES.has(job.status)) ?? null;
	$: panelRecentJobs = panelJobs.slice(0, 3);
	$: onlineWorkers = $allWorkers.filter((worker) => worker.status !== 'offline').length;

	$: if (officeScene) {
		officeScene.setWorkers(officePlacements, groupLabels);
		officeScene.setSelectedWorker(panelWorkerId);
		officeScene.setMovementEnabled(!showJobModal);
		if (panelWorkerId && !officePlacements.some((worker) => worker.workerId === panelWorkerId)) {
			panelWorkerId = null;
		}
	}

	onMount(async () => {
		updateClock();
		clockTimer = setInterval(updateClock, 1000);

		try {
			const jobs = await listJobs({ limit: 500, offset: 0 });
			allJobs.set(jobs);
		} catch {
			// Keep existing store values when refresh fails.
		}

		await tick();
		if (!viewportEl || !stageEl) return;

		officeScene = new OfficeScene(stageEl, MAP_CONFIG, {
			onNearestWorkerChange: (workerId) => {
				nearestWorkerId = workerId;
			},
			onInteract: (workerId) => {
				panelWorkerId = workerId;
				createdJobId = null;
			}
		});

		resizeHandler = () => {
			if (!viewportEl || !officeScene) return;
			officeScene.resize(viewportEl.clientWidth, viewportEl.clientHeight);
		};
		resizeHandler();
		window.addEventListener('resize', resizeHandler);

		keyHandler = (event: KeyboardEvent) => {
			if (isTypingTarget(event.target)) return;
			if (event.key === 'Escape' && !showJobModal) {
				panelWorkerId = null;
				return;
			}
			if (showJobModal) return;
			if ((event.key === 'e' || event.key === 'E') && nearestWorkerId) {
				event.preventDefault();
				if (panelWorkerId === nearestWorkerId) {
					panelWorkerId = null;
				} else {
					panelWorkerId = nearestWorkerId;
					createdJobId = null;
				}
			}
		};
		window.addEventListener('keydown', keyHandler);

		officeScene.setWorkers(officePlacements, groupLabels);
		officeScene.setSelectedWorker(panelWorkerId);
		officeScene.setMovementEnabled(!showJobModal);
		officeScene.start();
	});

	onDestroy(() => {
		if (clockTimer) clearInterval(clockTimer);
		if (resizeHandler) window.removeEventListener('resize', resizeHandler);
		if (keyHandler) window.removeEventListener('keydown', keyHandler);
		officeScene?.destroy();
		officeScene = null;
	});
</script>

<div class="office-shell">
	<div class="office-viewport" bind:this={viewportEl}>
		<div bind:this={stageEl} class="office-stage"></div>

		<div class="hud">
			<div class="corner tl"></div>
			<div class="corner tr"></div>
			<div class="corner bl"></div>
			<div class="corner br"></div>

			<div class="topbar">
				<div class="logo">◈ NEXUS</div>
				<div class="sep">|</div>
				<div>
					<div class="label">LOCATION</div>
					<div class="val">NIGHT OFFICE · SECTOR 7</div>
				</div>
				<div class="sep">|</div>
				<div>
					<div class="label">AGENTS ONLINE</div>
					<div class="val">{onlineWorkers} / {$allWorkers.length}</div>
				</div>
				<div class="sep">|</div>
				<div>
					<div class="label">SYSTEM</div>
					<div class="val">{currentTimeText}</div>
				</div>
			</div>

			<div class="controls">
				<span>WASD / ↑↓←→</span> — MOVE<br />
				<span>E / CLICK</span> — INTERACT<br />
				<span>ESC</span> — CLOSE PANEL
			</div>

			<div class="bottombar">
				◈ NETRUNNER INTERFACE v2.077 | SONGBIRD PROTOCOL ACTIVE | ENCRYPTED
			</div>
		</div>

		<div class="scanlines"></div>
		<div class="vignette"></div>
	</div>
</div>

{#if panelWorker}
	<OfficeInteractionPanel
		worker={panelWorker}
		activeJob={panelActiveJob}
		recentJobs={panelRecentJobs}
		{createdJobId}
		on:close={() => (panelWorkerId = null)}
		on:newjob={onPanelNewJob}
		on:viewjobs={onPanelViewJobs}
		on:settings={onPanelSettings}
	/>
{/if}

{#if showJobModal && panelWorker}
	<Modal on:close={() => (showJobModal = false)}>
		<div class="modal-header">
			<h2 class="modal-title">NEW JOB FOR {panelWorker.name.toUpperCase()}</h2>
			<p class="modal-sub">TARGET GROUP AND WORKER OVERRIDE ARE PRE-FILLED.</p>
		</div>
		{#if jobError}
			<p class="job-error">{jobError}</p>
		{/if}
		<NewJobForm
			workers={$allWorkers}
			loading={jobLoading}
			initialTargetGroup={panelWorker.group_name}
			initialWorkerOverride={panelWorker.worker_id}
			on:draft={handleJobDraft}
			on:submit={handleJobSubmit}
		/>
	</Modal>
{/if}

<style>
	.office-shell {
		flex: 1;
		min-height: 0;
		font-family: 'Courier New', monospace;
	}

	.office-viewport {
		position: relative;
		width: 100%;
		height: 100%;
		overflow: hidden;
		background: #03020a;
		cursor: crosshair;
	}

	.office-stage {
		display: block;
		width: 100%;
		height: 100%;
	}

	:global(.office-webgl) {
		display: block;
		width: 100%;
		height: 100%;
	}

	.hud {
		position: absolute;
		inset: 0;
		pointer-events: none;
		z-index: 20;
	}

	.corner {
		position: absolute;
		width: 28px;
		height: 28px;
		border-style: solid;
		border-color: rgba(0, 240, 255, 0.6);
	}

	.corner.tl {
		top: 16px;
		left: 16px;
		border-width: 2px 0 0 2px;
		box-shadow: -3px -3px 8px rgba(0, 240, 255, 0.3) inset;
	}

	.corner.tr {
		top: 16px;
		right: 16px;
		border-width: 2px 2px 0 0;
		box-shadow: 3px -3px 8px rgba(0, 240, 255, 0.3) inset;
	}

	.corner.bl {
		bottom: 16px;
		left: 16px;
		border-width: 0 0 2px 2px;
		box-shadow: -3px 3px 8px rgba(0, 240, 255, 0.3) inset;
	}

	.corner.br {
		right: 16px;
		bottom: 16px;
		border-width: 0 2px 2px 0;
		box-shadow: 3px 3px 8px rgba(0, 240, 255, 0.3) inset;
	}

	.topbar {
		position: absolute;
		top: 16px;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: 24px;
		padding: 6px 24px;
		border: 1px solid rgba(0, 240, 255, 0.2);
		background: rgba(3, 2, 10, 0.85);
		backdrop-filter: blur(8px);
		clip-path:
			polygon(
				12px 0%,
				calc(100% - 12px) 0%,
				100% 12px,
				100% 100%,
				calc(100% - 12px) 100%,
				12px 100%,
				0% calc(100% - 12px),
				0% 12px
			);
	}

	.logo {
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.2em;
		color: #00f0ff;
		text-shadow: 0 0 12px #00f0ff, 0 0 24px rgba(0, 240, 255, 0.4);
	}

	.sep {
		font-size: 10px;
		color: rgba(0, 240, 255, 0.2);
	}

	.label {
		font-size: 10px;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: rgba(0, 240, 255, 0.5);
	}

	.val {
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: #ff2a6d;
		text-shadow: 0 0 8px #ff2a6d;
	}

	.controls {
		position: absolute;
		left: 20px;
		bottom: 48px;
		padding: 8px 14px;
		border: 1px solid rgba(0, 240, 255, 0.12);
		background: rgba(3, 2, 10, 0.7);
		font-size: 10px;
		letter-spacing: 0.08em;
		line-height: 1.8;
		color: rgba(0, 240, 255, 0.35);
	}

	.controls span {
		color: rgba(0, 240, 255, 0.7);
	}

	.bottombar {
		position: absolute;
		left: 50%;
		bottom: 16px;
		transform: translateX(-50%);
		padding: 5px 20px;
		border: 1px solid rgba(255, 42, 109, 0.2);
		background: rgba(3, 2, 10, 0.8);
		font-size: 10px;
		letter-spacing: 0.15em;
		text-transform: uppercase;
		color: rgba(255, 42, 109, 0.6);
		clip-path:
			polygon(
				8px 0%,
				calc(100% - 8px) 0%,
				100% 8px,
				100% 100%,
				calc(100% - 8px) 100%,
				8px 100%,
				0% calc(100% - 8px),
				0% 8px
			);
	}

	.scanlines {
		position: absolute;
		inset: 0;
		background: repeating-linear-gradient(
			0deg,
			transparent,
			transparent 2px,
			rgba(0, 0, 0, 0.08) 2px,
			rgba(0, 0, 0, 0.08) 4px
		);
		pointer-events: none;
		z-index: 21;
	}

	.vignette {
		position: absolute;
		inset: 0;
		background: radial-gradient(ellipse at center, transparent 40%, rgba(0, 0, 0, 0.75) 100%);
		pointer-events: none;
		z-index: 22;
	}

	.modal-header {
		margin-bottom: 12px;
		padding-right: 26px;
		font-family: 'Courier New', monospace;
	}

	.modal-title {
		font-size: 16px;
		font-weight: 700;
		letter-spacing: 0.12em;
		color: #00f0ff;
	}

	.modal-sub {
		margin-top: 4px;
		font-size: 11px;
		letter-spacing: 0.08em;
		color: rgba(255, 42, 109, 0.8);
	}

	.job-error {
		margin-bottom: 12px;
		padding: 8px 10px;
		border: 1px solid rgba(255, 42, 109, 0.35);
		background: rgba(255, 42, 109, 0.08);
		font-size: 12px;
		color: #ff6a96;
	}

	@media (max-width: 900px) {
		.topbar {
			gap: 14px;
			padding: 6px 14px;
		}

		.controls {
			left: 12px;
			bottom: 58px;
			font-size: 9px;
		}

		.bottombar {
			width: calc(100% - 32px);
			text-align: center;
			line-height: 1.5;
		}
	}

	@media (max-width: 700px) {
		.topbar {
			width: calc(100% - 24px);
			justify-content: space-between;
			gap: 10px;
			font-size: 10px;
		}

		.topbar > :not(.logo):not(.sep) {
			min-width: 0;
		}

		.val {
			font-size: 10px;
		}

		.controls {
			display: none;
		}

		.corner {
			width: 22px;
			height: 22px;
		}
	}
</style>
