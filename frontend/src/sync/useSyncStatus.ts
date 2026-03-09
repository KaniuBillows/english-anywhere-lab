import { useContext } from 'react';
import { SyncContext } from './syncContext';

export function useSyncStatus() {
  const ctx = useContext(SyncContext);
  if (!ctx) throw new Error('useSyncStatus must be used within SyncProvider');
  return ctx;
}
