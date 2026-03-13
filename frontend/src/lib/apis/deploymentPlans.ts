import { DELETE, GET, POST } from './client';
import type { CreateDeploymentPlanRequest, DeploymentPlan } from '../../types/deploymentPlan';

export const listDeploymentPlans = () => GET<DeploymentPlan[]>('/deployment-plans');

export const createDeploymentPlan = (body: CreateDeploymentPlanRequest) =>
	POST<DeploymentPlan>('/deployment-plans', body);

export const getDeploymentPlan = (id: string) => GET<DeploymentPlan>(`/deployment-plans/${id}`);

export const stopDeploymentPlan = (id: string) =>
	POST<DeploymentPlan>(`/deployment-plans/${id}/stop`, {});

export const retryDeploymentPlan = (id: string) =>
	POST<DeploymentPlan>(`/deployment-plans/${id}/retry`, {});

export const deleteDeploymentPlan = (id: string) => DELETE<void>(`/deployment-plans/${id}`);
