import { writable, derived } from 'svelte/store';
import type { Job } from '../types/job';
import type { Message } from '../types/message';
import { selectedWorker } from './selectedWorker';

export const activeJobId = writable<string | null>(null);
export const activeJob = writable<Job | null>(null);
export const activeMessages = writable<Message[]>([]);

/** Composer is active when:
 *  - job is draft and a worker is selected (first send will submit + start)
 *  - job is pending_user (worker waiting for reply)
 *  - job is done (worker finished its turn, user can follow up)
 */
export const composerEnabled = derived(
	[activeJob, selectedWorker],
	([$job, $worker]) => {
		if (!$job) return false;
		if ($job.status === 'draft') return $worker !== null;
		return $job.status === 'pending_user' || $job.status === 'done';
	}
);

/** Append a message to the active message list, keeping created_at order. */
export function appendMessage(msg: Message) {
	activeMessages.update((msgs) => {
		if (msgs.find((m) => m.message_id === msg.message_id)) return msgs;
		return [...msgs, msg].sort(
			(a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
		);
	});
}
