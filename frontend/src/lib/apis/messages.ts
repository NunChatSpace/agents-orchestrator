import { GET, POST } from './client';
import type { Message } from '../../types/message';

export const listMessages = (jobId: string) => GET<Message[]>(`/jobs/${jobId}/messages`);

export const sendMessage = (jobId: string, content: string) =>
	POST<Message>(`/jobs/${jobId}/messages`, { content });
