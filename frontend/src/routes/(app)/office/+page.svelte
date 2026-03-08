<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import Modal from '../../../components/atoms/Modal.svelte';
	import NewJobForm from '../../../components/organisms/NewJobForm.svelte';
	import OfficeInteractionPanel from '../../../components/organisms/OfficeInteractionPanel.svelte';
	import { allWorkers } from '../../../stores/workers';
	import { allJobs, upsertJob } from '../../../stores/jobs';
	import { createJob, listJobs, submitJob } from '../../../lib/apis/jobs';
	import { MAP_CONFIG, resolveDeskPosition } from '../../../lib/office/mapConfig';
	import { OfficeEngine } from '../../../lib/office/OfficeEngine';
	import type { GroupZoneLabel, OfficeNPC } from '../../../lib/office/npcRenderer';
	import type { CreateJobPayload, Job } from '../../../types/job';
	import type { Worker } from '../../../types/worker';

	const ACTIVE_JOB_STATUSES = new Set(['assigned', 'busy', 'pending_user']);

	let viewportEl: HTMLDivElement | null = null;
	let canvasEl: HTMLCanvasElement | null = null;
	let engine: OfficeEngine | null = null;
	let resizeHandler: (() => void) | null = null;
	let keyHandler: ((event: KeyboardEvent) => void) | null = null;

	let nearestWorkerId: string | null = null;
	let panelWorkerId: string | null = null;
	let showJobModal = false;
	let jobLoading = false;
	let jobError = '';
	let createdJobId: string | null = null;

	function byCreatedAtThenName(a: Worker, b: Worker): number {
		const aTime = new Date(a.created_at).getTime();
		const bTime = new Date(b.created_at).getTime();
		if (aTime !== bTime) return aTime - bTime;
		return a.name.localeCompare(b.name);
	}

	function buildOfficeNPCs(workers: Worker[], jobs: Job[]): OfficeNPC[] {
		const groups = new Map<string, Worker[]>();
		for (const worker of workers) {
			const list = groups.get(worker.group_name) ?? [];
			list.push(worker);
			groups.set(worker.group_name, list);
		}

		const jobCountByWorker = new Map<string, number>();
		for (const job of jobs) {
			if (!job.assigned_worker_id) continue;
			if (!ACTIVE_JOB_STATUSES.has(job.status)) continue;
			jobCountByWorker.set(
				job.assigned_worker_id,
				(jobCountByWorker.get(job.assigned_worker_id) ?? 0) + 1
			);
		}

		const npcs: OfficeNPC[] = [];
		for (const workersInGroup of groups.values()) {
			const sorted = [...workersInGroup].sort(byCreatedAtThenName);
			for (const [index, worker] of sorted.entries()) {
				const desk = resolveDeskPosition(worker, index);
				npcs.push({
					workerId: worker.worker_id,
					name: worker.name,
					groupName: worker.group_name,
					status: worker.status,
					tileX: desk.tileX,
					tileY: desk.tileY,
					activeJobCount: jobCountByWorker.get(worker.worker_id) ?? 0
				});
			}
		}

		return npcs;
	}

	function buildGroupLabels(npcs: OfficeNPC[]): GroupZoneLabel[] {
		const groups = new Map<string, OfficeNPC[]>();
		for (const npc of npcs) {
			const items = groups.get(npc.groupName) ?? [];
			items.push(npc);
			groups.set(npc.groupName, items);
		}

		const labels: GroupZoneLabel[] = [];
		for (const [groupName, groupNpcs] of groups.entries()) {
			const minX = Math.min(...groupNpcs.map((npc) => npc.tileX));
			const minY = Math.min(...groupNpcs.map((npc) => npc.tileY));
			labels.push({
				groupName,
				tileX: Math.max(1, minX - 2),
				tileY: Math.max(1, minY - 2)
			});
		}

		return labels.sort((a, b) => a.tileY - b.tileY || a.tileX - b.tileX);
	}

	function jobsForWorker(workerId: string): Job[] {
		return $allJobs
			.filter((job) => job.assigned_worker_id === workerId)
			.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
	}

	function onCanvasClick(event: MouseEvent) {
		if (!engine || !canvasEl) return;
		const rect = canvasEl.getBoundingClientRect();
		engine.handleCanvasClick(event.clientX - rect.left, event.clientY - rect.top);
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

	function setVirtual(direction: 'up' | 'down' | 'left' | 'right', active: boolean) {
		engine?.setVirtualDirection(direction, active);
	}

	function handleDpadPress(direction: 'up' | 'down' | 'left' | 'right') {
		return (event: PointerEvent) => {
			event.preventDefault();
			setVirtual(direction, true);
		};
	}

	function handleDpadRelease(direction: 'up' | 'down' | 'left' | 'right') {
		return (event: PointerEvent) => {
			event.preventDefault();
			setVirtual(direction, false);
		};
	}

	$: officeNpcs = buildOfficeNPCs($allWorkers, $allJobs);
	$: groupLabels = buildGroupLabels(officeNpcs);
	$: nearestWorker = nearestWorkerId ? $allWorkers.find((worker) => worker.worker_id === nearestWorkerId) : null;
	$: panelWorker = panelWorkerId ? $allWorkers.find((worker) => worker.worker_id === panelWorkerId) : null;
	$: panelJobs = panelWorkerId ? jobsForWorker(panelWorkerId) : [];
	$: panelActiveJob = panelJobs.find((job) => ACTIVE_JOB_STATUSES.has(job.status)) ?? null;
	$: panelRecentJobs = panelJobs.slice(0, 3);

	$: if (engine) {
		engine.setWorkers(officeNpcs, groupLabels);
		if (panelWorkerId && !officeNpcs.some((npc) => npc.workerId === panelWorkerId)) {
			panelWorkerId = null;
		}
	}

	onMount(async () => {
		try {
			const jobs = await listJobs({ limit: 500, offset: 0 });
			allJobs.set(jobs);
		} catch {
			// Keep existing store values when refresh fails.
		}

		await tick();
		if (!viewportEl || !canvasEl) return;

		engine = new OfficeEngine(canvasEl, MAP_CONFIG, {
			onNearestWorkerChange: (workerId) => {
				nearestWorkerId = workerId;
			},
			onInteract: (workerId) => {
				panelWorkerId = workerId;
				createdJobId = null;
			}
		});

		resizeHandler = () => {
			if (!viewportEl || !engine) return;
			engine.resize(viewportEl.clientWidth, viewportEl.clientHeight);
		};
		resizeHandler();
		window.addEventListener('resize', resizeHandler);

		keyHandler = (event: KeyboardEvent) => {
			if (event.key === 'Escape' && !showJobModal) {
				panelWorkerId = null;
				return;
			}
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

		engine.setWorkers(officeNpcs, groupLabels);
		engine.start();
	});

	onDestroy(() => {
		if (resizeHandler) window.removeEventListener('resize', resizeHandler);
		if (keyHandler) window.removeEventListener('keydown', keyHandler);
		engine?.destroy();
		engine = null;
	});
</script>

<div class="office-shell">
	<div class="office-viewport" bind:this={viewportEl}>
		<canvas bind:this={canvasEl} class="office-canvas" on:click={onCanvasClick}></canvas>

		{#if nearestWorker}
			<div class="prompt">
				<span>[ Press E to interact — {nearestWorker.name} ]</span>
			</div>
		{/if}

		<div class="legend">
			<div class="legend-title">Office Controls</div>
			<div class="legend-line">Move: WASD / Arrows</div>
			<div class="legend-line">Interact: E or click agent</div>
		</div>

		<div class="dpad">
			<button
				class="dpad-btn up"
				on:pointerdown={handleDpadPress('up')}
				on:pointerup={handleDpadRelease('up')}
				on:pointercancel={handleDpadRelease('up')}
				on:pointerleave={handleDpadRelease('up')}
				aria-label="Move up">↑</button>
			<button
				class="dpad-btn left"
				on:pointerdown={handleDpadPress('left')}
				on:pointerup={handleDpadRelease('left')}
				on:pointercancel={handleDpadRelease('left')}
				on:pointerleave={handleDpadRelease('left')}
				aria-label="Move left">←</button>
			<button
				class="dpad-btn right"
				on:pointerdown={handleDpadPress('right')}
				on:pointerup={handleDpadRelease('right')}
				on:pointercancel={handleDpadRelease('right')}
				on:pointerleave={handleDpadRelease('right')}
				aria-label="Move right">→</button>
			<button
				class="dpad-btn down"
				on:pointerdown={handleDpadPress('down')}
				on:pointerup={handleDpadRelease('down')}
				on:pointercancel={handleDpadRelease('down')}
				on:pointerleave={handleDpadRelease('down')}
				aria-label="Move down">↓</button>
		</div>
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
			<h2 class="modal-title">New Job for {panelWorker.name}</h2>
			<p class="modal-sub">Target group and worker override are pre-filled.</p>
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
	}

	.office-viewport {
		position: relative;
		width: 100%;
		height: 100%;
		background: radial-gradient(circle at 10% 10%, rgba(124,58,237,0.2), rgba(5,5,12,0.8) 42%);
		overflow: hidden;
	}

	.office-canvas {
		width: 100%;
		height: 100%;
		display: block;
		cursor: crosshair;
	}

	.prompt {
		position: absolute;
		bottom: 24px;
		left: 50%;
		transform: translateX(-50%);
		padding: 8px 14px;
		border-radius: 9px;
		background: rgba(10,10,22,0.82);
		border: 1px solid rgba(139,92,246,0.35);
		color: #e9ddff;
		font-size: 12px;
		font-family: 'Space Grotesk', sans-serif;
		z-index: 40;
		pointer-events: none;
	}

	.legend {
		position: absolute;
		top: 14px;
		left: 14px;
		background: rgba(10,10,22,0.75);
		border: 1px solid rgba(139,92,246,0.24);
		border-radius: 10px;
		padding: 8px 10px;
		z-index: 40;
	}

	.legend-title {
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(196,181,253,0.54);
		margin-bottom: 4px;
	}

	.legend-line {
		font-size: 11px;
		color: rgba(233,221,255,0.86);
	}

	.dpad {
		position: absolute;
		left: 14px;
		bottom: 18px;
		width: 120px;
		height: 120px;
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		grid-template-rows: repeat(3, 1fr);
		gap: 4px;
		z-index: 41;
	}

	.dpad-btn {
		border-radius: 10px;
		border: 1px solid rgba(139,92,246,0.34);
		background: rgba(13,13,30,0.72);
		color: #ddd6fe;
		font-size: 19px;
		font-weight: 700;
	}

	.dpad-btn:active {
		background: rgba(124,58,237,0.35);
	}

	.dpad-btn.up {
		grid-column: 2;
		grid-row: 1;
	}

	.dpad-btn.left {
		grid-column: 1;
		grid-row: 2;
	}

	.dpad-btn.right {
		grid-column: 3;
		grid-row: 2;
	}

	.dpad-btn.down {
		grid-column: 2;
		grid-row: 3;
	}

	.modal-header {
		margin-bottom: 12px;
		padding-right: 26px;
	}

	.modal-title {
		font-family: 'Space Grotesk', sans-serif;
		font-size: 18px;
		font-weight: 700;
		color: #f2ebff;
	}

	.modal-sub {
		margin-top: 4px;
		font-size: 12px;
		color: rgba(196,181,253,0.58);
	}

	.job-error {
		margin-bottom: 12px;
		font-size: 12.5px;
		color: #fca5a5;
		background: rgba(248,113,113,0.08);
		border: 1px solid rgba(248,113,113,0.25);
		border-radius: 8px;
		padding: 8px 10px;
	}

	@media (min-width: 960px) {
		.dpad {
			display: none;
		}
	}
</style>
