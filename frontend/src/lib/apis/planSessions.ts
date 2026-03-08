import { GET, POST, PATCH } from './client';
import type { PlanSession } from '../../types/plan_session';

const BASE = `${import.meta.env.VITE_BACKEND_URL ?? ''}/api/v1`;

export const listPlanSessions = () => GET<PlanSession[]>('/plan-sessions');

export const createPlanSession = (workerID: string) =>
	POST<PlanSession>('/plan-sessions', { worker_id: workerID });

export const getPlanSession = (id: string) => GET<PlanSession>(`/plan-sessions/${id}`);

export const updatePlanSessionTitle = (id: string, title: string) =>
	PATCH<PlanSession>(`/plan-sessions/${id}`, { title });

export const completePlanSession = (id: string) =>
	POST<PlanSession>(`/plan-sessions/${id}/complete`);

export const discardPlanSession = (id: string) =>
	POST<void>(`/plan-sessions/${id}/discard`);

// SSE: send a user message and stream the agent reply.
// onThinking is called for each thinking step; onDone is called with the final agent reply.
export function sendPlanMessage(
	id: string,
	content: string,
	onThinking: (text: string) => void,
	onDone: () => void,
	onError: (msg: string) => void
): () => void {
	const ctrl = new AbortController();

	fetch(`${BASE}/plan-sessions/${id}/message`, {
		method: 'POST',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ content }),
		signal: ctrl.signal
	}).then(async (res) => {
		if (!res.ok || !res.body) {
			onError('Request failed');
			return;
		}
		await readSSEStream(res.body, onThinking, onDone, onError);
	}).catch((err) => {
		if (err?.name !== 'AbortError') onError(err?.message ?? 'Unknown error');
	});

	return () => ctrl.abort();
}

// SSE: trigger plan generation and stream the agent reply.
// onDone is called with the generated_prompt string.
export function generatePlan(
	id: string,
	onThinking: (text: string) => void,
	onDone: (prompt: string) => void,
	onError: (msg: string) => void
): () => void {
	const ctrl = new AbortController();

	fetch(`${BASE}/plan-sessions/${id}/generate`, {
		method: 'POST',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		signal: ctrl.signal
	}).then(async (res) => {
		if (!res.ok || !res.body) {
			onError('Request failed');
			return;
		}
		await readSSEStream(
			res.body,
			onThinking,
			(data) => onDone(typeof data === 'string' ? data : ''),
			onError
		);
	}).catch((err) => {
		if (err?.name !== 'AbortError') onError(err?.message ?? 'Unknown error');
	});

	return () => ctrl.abort();
}

// Reads an SSE stream, calling callbacks for each event type.
async function readSSEStream(
	body: ReadableStream<Uint8Array>,
	onThinking: (text: string) => void,
	onDone: (data?: string) => void,
	onError: (msg: string) => void
) {
	const reader = body.getReader();
	const decoder = new TextDecoder();
	let buffer = '';

	try {
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			buffer += decoder.decode(value, { stream: true });

			const blocks = buffer.split('\n\n');
			buffer = blocks.pop() ?? '';

			for (const block of blocks) {
				const lines = block.split('\n');
				let event = 'message';
				let data = '';
				for (const line of lines) {
					if (line.startsWith('event: ')) event = line.slice(7).trim();
					else if (line.startsWith('data: ')) data = line.slice(6).trim();
				}
				if (!data) continue;

				const parsed = JSON.parse(data);
				if (event === 'thinking') onThinking(parsed as string);
				else if (event === 'done') onDone(parsed as string);
				else if (event === 'error') onError(parsed as string);
			}
		}
	} finally {
		reader.releaseLock();
	}
}
