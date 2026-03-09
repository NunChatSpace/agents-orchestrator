import { GET, POST, PATCH, DELETE } from './client';
import type { Worker, CreateWorkerPayload, PingResult, UpdateWorkerPayload } from '../../types/worker';

export const listWorkers = () => GET<Worker[]>('/workers');

export const getWorker = (workerId: string) => GET<Worker>(`/workers/${workerId}`);

export const createWorker = (payload: CreateWorkerPayload) =>
	POST<Worker>('/workers', payload);

export const updateWorker = (workerId: string, payload: UpdateWorkerPayload) =>
	PATCH<Worker>(`/workers/${workerId}`, payload);

export const resetWorkerInstruction = (
	workerId: string,
	field: 'job' | 'plan' | 'discuss'
) => POST<Worker>(`/workers/${workerId}/instructions/reset/${field}`, {});

export const pingWorker = (workerId: string) =>
	POST<PingResult>(`/workers/${workerId}/ping`, {});

export const deleteWorker = (workerId: string) =>
	DELETE<null>(`/workers/${workerId}`);

export const planWorker = (workerId: string, task: string) =>
	POST<{ prompt: string }>(`/workers/${workerId}/plan`, { task });
