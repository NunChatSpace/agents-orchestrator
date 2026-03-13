import { GET, POST } from './client';
import type { WorkerBuild, TriggerBuildRequest } from '../../types/workerBuild';

export const triggerBuild = (workerId: string, body: TriggerBuildRequest) =>
	POST<WorkerBuild>(`/workers/${workerId}/builds`, body);

export const listWorkerBuilds = (workerId: string) =>
	GET<WorkerBuild[]>(`/workers/${workerId}/builds`);

export const getWorkerBuild = (buildId: string) =>
	GET<WorkerBuild>(`/worker-builds/${buildId}`);
