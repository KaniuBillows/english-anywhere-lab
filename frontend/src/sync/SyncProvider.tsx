import { useEffect, useState, useCallback, useRef, type ReactNode } from 'react';
import { SyncContext } from './syncContext';
import { SyncEngine, type SyncStatus } from './engine';
import { enqueueEvent } from './db';
import { generateEventId } from './uuid';
import type { SyncEventType } from '../api/types';

export default function SyncProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SyncStatus>('idle');
  const [pendingCount, setPendingCount] = useState(0);
  const engineRef = useRef<SyncEngine | null>(null);

  useEffect(() => {
    const engine = new SyncEngine({
      onStatusChange(s, count) {
        setStatus(s);
        setPendingCount(count);
      },
      onChangesReceived(changes) {
        // Future: update local cache / trigger React Query invalidation
        console.debug('[sync] received changes:', changes.length);
      },
    });
    engineRef.current = engine;
    engine.start();
    return () => engine.stop();
  }, []);

  const syncNow = useCallback(() => {
    void engineRef.current?.sync();
  }, []);

  const enqueue = useCallback(
    async (eventType: SyncEventType, payload: Record<string, unknown>) => {
      await enqueueEvent({
        client_event_id: generateEventId(),
        event_type: eventType,
        occurred_at: new Date().toISOString(),
        payload,
      });
      void engineRef.current?.sync();
    },
    [],
  );

  return (
    <SyncContext.Provider value={{ status, pendingCount, syncNow, enqueue }}>
      {children}
    </SyncContext.Provider>
  );
}
