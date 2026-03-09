import { apiFetch } from './client';
import type { SyncEventsRequest, SyncEventsResponse, SyncChangesResponse } from './types';

export async function pushSyncEvents(req: SyncEventsRequest): Promise<SyncEventsResponse> {
  return apiFetch<SyncEventsResponse>('/api/v1/sync/events', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function pullSyncChanges(
  cursor: string,
  limit = 100,
): Promise<SyncChangesResponse> {
  return apiFetch<SyncChangesResponse>(
    `/api/v1/sync/changes?cursor=${encodeURIComponent(cursor)}&limit=${limit}`,
  );
}
