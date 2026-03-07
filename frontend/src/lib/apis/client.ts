import type { ApiErrorResponse } from '../../types/api';
import { ApiError } from '../../types/api';

// VITE_BACKEND_URL is set at build time (e.g. http://localhost:8080 for Docker).
// When empty the browser uses relative paths (same-origin / dev proxy).
const BASE = `${import.meta.env.VITE_BACKEND_URL ?? ''}/api/v1`;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, {
		...init,
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...init?.headers
		}
	});

	if (res.status === 204) return null as T;

	const body = await res.json();

	if (!res.ok) {
		const err = body as ApiErrorResponse;
		throw new ApiError(err.error?.code ?? 'UNKNOWN', err.error?.message ?? res.statusText, res.status);
	}

	return body.data as T;
}

export async function GET<T>(path: string): Promise<T> {
	return request<T>(path);
}

export async function POST<T>(path: string, body?: unknown): Promise<T> {
	return request<T>(path, {
		method: 'POST',
		body: body !== undefined ? JSON.stringify(body) : undefined
	});
}

export async function PATCH<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
}

export async function DELETE<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'DELETE' });
}
