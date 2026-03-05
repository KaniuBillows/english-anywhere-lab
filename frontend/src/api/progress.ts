import { apiFetch } from './client';
import type {
  MonthlyReportResponse,
  ProgressDailyResponse,
  ProgressSummary,
  WeeklyReportResponse,
} from './types';

export async function getSummary(range: string): Promise<ProgressSummary> {
  return apiFetch<ProgressSummary>(
    `/api/v1/progress/summary?range=${encodeURIComponent(range)}`,
  );
}

export async function getDaily(range: string): Promise<ProgressDailyResponse> {
  return apiFetch<ProgressDailyResponse>(
    `/api/v1/progress/daily?range=${encodeURIComponent(range)}`,
  );
}

export async function getWeeklyReport(weekStart?: string): Promise<WeeklyReportResponse> {
  const qs = weekStart ? `?week_start=${encodeURIComponent(weekStart)}` : '';
  return apiFetch<WeeklyReportResponse>(`/api/v1/progress/weekly-report${qs}`);
}

export async function getMonthlyReport(month?: string): Promise<MonthlyReportResponse> {
  const qs = month ? `?month=${encodeURIComponent(month)}` : '';
  return apiFetch<MonthlyReportResponse>(`/api/v1/progress/monthly-report${qs}`);
}
