import { GET, POST, DELETE } from './client';
import type { PreviewSession, CreatePreviewSessionPayload } from '../../types/previewSession';

export const listPreviewSessions = () => GET<PreviewSession[]>('/preview-sessions');

export const getPreviewSession = (id: string) => GET<PreviewSession>(`/preview-sessions/${id}`);

export const createPreviewSession = (payload: CreatePreviewSessionPayload) =>
	POST<PreviewSession>('/preview-sessions', payload);

export const stopPreviewSession = (id: string) =>
	POST<{ status: string }>(`/preview-sessions/${id}/stop`);

export const deletePreviewSession = (id: string) => DELETE<void>(`/preview-sessions/${id}`);

export const getRoleLog = (sessionId: string, roleId: string) =>
	GET<{ log: string }>(`/preview-sessions/${sessionId}/roles/${roleId}/logs`);
