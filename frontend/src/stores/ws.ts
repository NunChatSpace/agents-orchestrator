import { get } from 'svelte/store';
import type { Job } from '../types/job';
import type { Message } from '../types/message';
import type { Worker } from '../types/worker';
import { upsertJob } from './jobs';
import { activeJobId, activeJob, appendMessage } from './activeJob';
import { upsertWorker } from './workers';

let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

interface WSEvent {
	type: 'job_updated' | 'message_added' | 'worker_updated';
	data: unknown;
}

export function connectWS() {
	if (socket?.readyState === WebSocket.OPEN) return;

	// Use VITE_BACKEND_URL when non-empty, otherwise same-origin WS.
	const backendOrigin = import.meta.env.VITE_BACKEND_URL || `${location.protocol}//${location.host}`;
	const protocol = backendOrigin.startsWith('https') ? 'wss' : 'ws';
	const host = backendOrigin.replace(/^https?:\/\//, '');
	const url = `${protocol}://${host}/ws`;
	socket = new WebSocket(url);

	socket.onmessage = (e) => {
		try {
			const evt = JSON.parse(e.data) as WSEvent;
			handleEvent(evt);
		} catch {
			// ignore malformed events
		}
	};

	socket.onclose = () => {
		socket = null;
		// Reconnect after 3 seconds
		reconnectTimer = setTimeout(() => connectWS(), 3000);
	};

	socket.onerror = () => {
		socket?.close();
	};
}

export function disconnectWS() {
	if (reconnectTimer) clearTimeout(reconnectTimer);
	socket?.close();
	socket = null;
}

function handleEvent(evt: WSEvent) {
	switch (evt.type) {
		case 'job_updated': {
			const job = evt.data as Job;
			upsertJob(job);
			// If this is the currently viewed job, update activeJob too
			if (job.job_id === get(activeJobId)) {
				activeJob.set(job);
			}
			break;
		}
		case 'message_added': {
			const msg = evt.data as Message;
			if (msg.job_id === get(activeJobId)) {
				appendMessage(msg);
			}
			break;
		}
		case 'worker_updated': {
			const worker = evt.data as Worker;
			upsertWorker(worker);
			break;
		}
	}
}
