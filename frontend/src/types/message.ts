export type MessageRole = 'user' | 'oagent' | 'worker';
export type MessageKind = 'instruction' | 'question' | 'answer' | 'summary' | 'system' | 'thinking' | 'file_change';

export interface Message {
	message_id: string;
	job_id: string;
	role: MessageRole;
	kind: MessageKind;
	content: string;
	created_at: string;
}
