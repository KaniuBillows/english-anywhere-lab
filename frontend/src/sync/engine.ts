import { pushSyncEventsRaw, pullSyncChanges } from '../api/sync';
import type { SyncChange, SyncEventDTO } from '../api/types';
import {
  getPendingEvents,
  markAcked,
  markRejected,
  incrementRetry,
  getPendingCount,
  pruneAckedEvents,
} from './db';

export type SyncStatus = 'idle' | 'syncing' | 'error' | 'offline';

const CURSOR_KEY = 'ea_sync_cursor';
const SYNC_INTERVAL_MS = 30_000;
const MAX_RETRY = 5;
const BACKOFF_BASE_MS = 1_000;
const BACKOFF_CAP_MS = 16_000;
const PRUNE_AGE_DAYS = 7;

export interface SyncEngineCallbacks {
  onStatusChange: (status: SyncStatus, pendingCount: number) => void;
  onChangesReceived: (changes: SyncChange[]) => void;
  getAccessToken: () => string | null;
}

export class SyncEngine {
  private callbacks: SyncEngineCallbacks;
  private intervalId: ReturnType<typeof setInterval> | null = null;
  private running = false;
  private status: SyncStatus = 'idle';

  constructor(callbacks: SyncEngineCallbacks) {
    this.callbacks = callbacks;
  }

  start(): void {
    window.addEventListener('online', this.handleOnline);
    window.addEventListener('offline', this.handleOffline);
    document.addEventListener('visibilitychange', this.handleVisibility);
    this.intervalId = setInterval(() => void this.sync(), SYNC_INTERVAL_MS);

    if (navigator.onLine) {
      void this.sync();
    } else {
      this.setStatus('offline');
    }
  }

  stop(): void {
    window.removeEventListener('online', this.handleOnline);
    window.removeEventListener('offline', this.handleOffline);
    document.removeEventListener('visibilitychange', this.handleVisibility);
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  async sync(): Promise<void> {
    if (this.running) return;
    if (!navigator.onLine) {
      this.setStatus('offline');
      return;
    }

    this.running = true;
    this.setStatus('syncing');

    try {
      await this.push();
      await this.pull();
      await this.prune();
      this.setStatus('idle');
    } catch {
      this.setStatus('error');
    } finally {
      this.running = false;
    }
  }

  private async push(): Promise<void> {
    const pending = await getPendingEvents();
    const eligible = pending.filter((e) => e.retry_count < MAX_RETRY);
    if (eligible.length === 0) return;

    const events: SyncEventDTO[] = eligible.map((e) => ({
      client_event_id: e.client_event_id,
      event_type: e.event_type,
      occurred_at: e.occurred_at,
      payload: e.payload,
    }));

    const token = this.callbacks.getAccessToken();

    let res: { status: number; data?: Awaited<ReturnType<typeof pushSyncEventsRaw>>['data'] };
    try {
      res = await pushSyncEventsRaw({ events }, token);
    } catch {
      // Network error (offline, DNS, etc.) — treat as transient
      const ids = eligible.map((e) => e.client_event_id);
      await incrementRetry(ids);
      await this.backoff(eligible);
      throw new Error('Network error during push');
    }

    // 2xx — process per-event acks
    if (res.status >= 200 && res.status < 300 && res.data) {
      for (const ack of res.data.acks) {
        if (ack.status === 'accepted' || ack.status === 'duplicate') {
          await markAcked(ack.client_event_id);
        } else {
          await markRejected(ack.client_event_id);
        }
      }
      return;
    }

    // 401 — auth expired; don't burn retries, let next cycle try after token refresh
    if (res.status === 401) {
      return;
    }

    // 5xx — transient server failure, increment retry + exponential backoff
    if (res.status >= 500) {
      const ids = eligible.map((e) => e.client_event_id);
      await incrementRetry(ids);
      await this.backoff(eligible);
      throw new Error(`Server error ${res.status} during push`);
    }

    // Other 4xx — client error, don't retry (would never succeed)
    // Log and move on without incrementing retry or throwing
    console.warn(`[sync] push got ${res.status}, skipping batch`);
  }

  private async backoff(
    events: { retry_count: number }[],
  ): Promise<void> {
    const maxRetry = Math.max(...events.map((e) => e.retry_count));
    const delay = Math.min(BACKOFF_BASE_MS * 2 ** maxRetry, BACKOFF_CAP_MS);
    await new Promise((r) => setTimeout(r, delay));
  }

  private async pull(): Promise<void> {
    const cursor = localStorage.getItem(CURSOR_KEY) ?? '';
    const res = await pullSyncChanges(cursor);

    if (res.changes.length > 0) {
      this.callbacks.onChangesReceived(res.changes);
    }

    if (res.next_cursor) {
      localStorage.setItem(CURSOR_KEY, res.next_cursor);
    }
  }

  private async prune(): Promise<void> {
    const cutoff = new Date();
    cutoff.setDate(cutoff.getDate() - PRUNE_AGE_DAYS);
    await pruneAckedEvents(cutoff);
  }

  private setStatus(status: SyncStatus): void {
    this.status = status;
    void getPendingCount().then((count) => {
      this.callbacks.onStatusChange(this.status, count);
    });
  }

  private handleOnline = (): void => {
    void this.sync();
  };

  private handleOffline = (): void => {
    this.setStatus('offline');
  };

  private handleVisibility = (): void => {
    if (document.visibilityState === 'visible' && navigator.onLine) {
      void this.sync();
    }
  };
}
