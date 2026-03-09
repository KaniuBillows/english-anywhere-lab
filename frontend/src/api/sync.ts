import { apiFetch } from './client';
import type { SyncEventsRequest, SyncEventsResponse, SyncChangesResponse } from './types';

export class SyncApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function pushSyncEvents(req: SyncEventsRequest): Promise<SyncEventsResponse> {
  try {
    return await apiFetch<SyncEventsResponse>('/api/v1/sync/events', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  } catch (err: unknown) {
    // apiFetch throws ApiError objects with a `code` field.
    // Re-throw as SyncApiError with numeric status when identifiable.
    if (err && typeof err === 'object' && 'code' in err) {
      const apiErr = err as { code: string; message: string };
      // apiFetch sets code = 'UNKNOWN' with statusText for non-JSON errors
      // For 5xx the backend may return structured errors or just statusText
      throw new SyncApiError(0, apiErr.message);
    }
    throw err;
  }
}

/**
 * Low-level push that preserves HTTP status for retry logic.
 * Bypasses apiFetch so the engine can distinguish 4xx vs 5xx.
 */
export async function pushSyncEventsRaw(
  req: SyncEventsRequest,
  accessToken: string | null,
): Promise<{ status: number; data?: SyncEventsResponse }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;

  const res = await fetch('/api/v1/sync/events', {
    method: 'POST',
    headers,
    body: JSON.stringify(req),
  });

  if (res.ok) {
    const data = (await res.json()) as SyncEventsResponse;
    return { status: res.status, data };
  }

  return { status: res.status };
}

export async function pullSyncChanges(
  cursor: string,
  limit = 100,
): Promise<SyncChangesResponse> {
  return apiFetch<SyncChangesResponse>(
    `/api/v1/sync/changes?cursor=${encodeURIComponent(cursor)}&limit=${limit}`,
  );
}
