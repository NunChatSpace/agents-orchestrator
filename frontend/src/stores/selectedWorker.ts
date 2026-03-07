import { writable } from 'svelte/store';
import type { Worker } from '../types/worker';

export const selectedWorker = writable<Worker | null>(null);
