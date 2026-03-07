import { GET, POST, PATCH, DELETE } from './client';
import type { Worker, CreateWorkerPayload, PingResult } from '../../types/worker';

export const listWorkers = () => GET<Worker[]>('/workers');

export const getWorker = (workerId: string) => GET<Worker>(`/workers/${workerId}`);

export const createWorker = (payload: CreateWorkerPayload) =>
	POST<Worker>('/workers', payload);

export const updateWorkerCLI = (workerId: string, cliCommand: string) =>
	PATCH<Worker>(`/workers/${workerId}`, { cli_command: cliCommand });

export const pingWorker = (workerId: string) =>
	POST<PingResult>(`/workers/${workerId}/ping`, {});

export const deleteWorker = (workerId: string) =>
	DELETE<null>(`/workers/${workerId}`);

export const planWorker = (workerId: string, task: string) =>
	POST<{ prompt: string }>(`/workers/${workerId}/plan`, { task });
