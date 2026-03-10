import { GET, POST } from './client';
import type { CreatePreviewBundleRequest, PreviewBundle, PreviewStack } from '../../types/preview';

export const listPreviewStacks = () => GET<PreviewStack[]>('/preview-stacks');

export const listPreviewBundles = () => GET<PreviewBundle[]>('/preview-bundles');

export const createPreviewBundle = (body: CreatePreviewBundleRequest) =>
	POST<PreviewBundle>('/preview-bundles', body);

export const destroyPreviewBundle = (bundleId: string) =>
	POST<PreviewBundle>(`/preview-bundles/${bundleId}/destroy`, {});
