export type PlanSessionStatus = 'pending' | 'completed' | 'discarded';

export interface PlanMessage {
	id: string;
	role: 'user' | 'agent';
	content: string;
	created_at: string;
}

export interface PlanSession {
	id: string;
	worker_id: string;
	title: string;
	status: PlanSessionStatus;
	generated_prompt?: string;
	created_at: string;
	updated_at: string;
	messages?: PlanMessage[];
}
