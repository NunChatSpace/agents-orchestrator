export type WorkerStatus = 'busy' | 'pending_user' | 'idle' | 'offline';
export type CLICommand = 'claude' | 'codex';

export interface Worker {
	worker_id: string;
	group_name: string;
	name: string;
	cli_command: string;
	build_command?: string;
	workspace: string;
	git_repo_url: string;
	instruction_job: string;
	instruction_plan: string;
	instruction_discuss: string;
	map_x: number;
	map_y: number;
	status: WorkerStatus;
	last_active_at?: string;
	created_at: string;
}

export interface CreateWorkerPayload {
	name: string;
	git_repo_url: string;
	group_name: string;
	cli_command: CLICommand;
}

export interface PingResult {
	ok: boolean;
	output: string;
	error?: string;
}

export interface UpdateWorkerPayload {
	cli_command?: CLICommand;
	build_command?: string;
	map_x?: number;
	map_y?: number;
	instruction_job?: string;
	instruction_plan?: string;
	instruction_discuss?: string;
}
