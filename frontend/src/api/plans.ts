import { apiFetch } from './client';
import type {
  BootstrapPlanRequest,
  CompleteTaskRequest,
  DailyPlanResponse,
  TaskCompletionResponse,
} from './types';

export async function bootstrapPlan(req: BootstrapPlanRequest): Promise<DailyPlanResponse> {
  return apiFetch<DailyPlanResponse>('/api/v1/plans/bootstrap', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function getTodayPlan(timezone: string): Promise<DailyPlanResponse> {
  return apiFetch<DailyPlanResponse>(
    `/api/v1/plans/today?timezone=${encodeURIComponent(timezone)}`,
  );
}

export async function completeTask(
  planId: string,
  taskId: string,
  req: CompleteTaskRequest,
): Promise<TaskCompletionResponse> {
  return apiFetch<TaskCompletionResponse>(
    `/api/v1/plans/${planId}/tasks/${taskId}/complete`,
    {
      method: 'POST',
      body: JSON.stringify(req),
    },
  );
}
