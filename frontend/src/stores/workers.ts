import { writable } from 'svelte/store';
import type { Worker } from '../types/worker';

export const allWorkers = writable<Worker[]>([]);

export function upsertWorker(worker: Worker) {
	allWorkers.update((workers) => {
		const idx = workers.findIndex((w) => w.worker_id === worker.worker_id);
		if (idx >= 0) {
			const copy = [...workers];
			copy[idx] = { ...copy[idx], ...worker };
			return copy;
		}
		return [...workers, worker];
	});
}

export function removeWorker(workerId: string) {
	allWorkers.update((workers) => workers.filter((w) => w.worker_id !== workerId));
}
