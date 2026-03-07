export interface ApiResponse<T> {
	data: T;
	meta: {
		request_id?: string;
	};
}

export interface ApiListResponse<T> {
	data: T[];
	meta: {
		request_id?: string;
		total_items: number;
	};
}

export interface ApiErrorResponse {
	error: {
		code: string;
		message: string;
	};
	meta?: {
		request_id?: string;
	};
}

export class ApiError extends Error {
	constructor(
		public code: string,
		message: string,
		public status: number
	) {
		super(message);
		this.name = 'ApiError';
	}
}
