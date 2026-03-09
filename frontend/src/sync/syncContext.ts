import { createContext } from 'react';
import type { SyncStatus } from './engine';
import type { SyncEventType } from '../api/types';

export interface SyncContextValue {
  status: SyncStatus;
  pendingCount: number;
  syncNow: () => void;
  enqueue: (
    eventType: SyncEventType,
    payload: Record<string, unknown>,
  ) => Promise<void>;
}

export const SyncContext = createContext<SyncContextValue | null>(null);
