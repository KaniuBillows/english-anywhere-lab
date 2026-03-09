import { openDB, type IDBPDatabase } from 'idb';
import type { SyncEventType } from '../api/types';

const DB_NAME = 'english-anywhere-sync';
const DB_VERSION = 1;
const STORE_NAME = 'event_queue';
const SEQ_KEY = 'ea_sync_client_seq';

export interface QueuedEvent {
  client_event_id: string;
  event_type: SyncEventType;
  occurred_at: string;
  client_seq: number;
  payload: Record<string, unknown>;
  sync_status: 'pending' | 'acked' | 'rejected';
  retry_count: number;
}

function getNextSeq(): number {
  const current = parseInt(localStorage.getItem(SEQ_KEY) ?? '0', 10);
  const next = current + 1;
  localStorage.setItem(SEQ_KEY, String(next));
  return next;
}

let dbPromise: Promise<IDBPDatabase> | null = null;

function getDB(): Promise<IDBPDatabase> {
  if (!dbPromise) {
    dbPromise = openDB(DB_NAME, DB_VERSION, {
      upgrade(db) {
        const store = db.createObjectStore(STORE_NAME, { keyPath: 'client_event_id' });
        store.createIndex('by_status', 'sync_status');
        store.createIndex('by_seq', 'client_seq');
      },
    });
  }
  return dbPromise;
}

export async function enqueueEvent(
  event: Pick<QueuedEvent, 'client_event_id' | 'event_type' | 'occurred_at' | 'payload'>,
): Promise<void> {
  const db = await getDB();
  const record: QueuedEvent = {
    ...event,
    client_seq: getNextSeq(),
    sync_status: 'pending',
    retry_count: 0,
  };
  await db.put(STORE_NAME, record);
}

export async function getPendingEvents(limit = 500): Promise<QueuedEvent[]> {
  const db = await getDB();
  const all = await db.getAllFromIndex(STORE_NAME, 'by_status', 'pending');
  all.sort((a, b) => {
    const t = a.occurred_at.localeCompare(b.occurred_at);
    return t !== 0 ? t : a.client_seq - b.client_seq;
  });
  return all.slice(0, limit);
}

export async function markAcked(id: string): Promise<void> {
  const db = await getDB();
  const record = await db.get(STORE_NAME, id);
  if (record) {
    record.sync_status = 'acked';
    await db.put(STORE_NAME, record);
  }
}

export async function markRejected(id: string): Promise<void> {
  const db = await getDB();
  const record = await db.get(STORE_NAME, id);
  if (record) {
    record.sync_status = 'rejected';
    await db.put(STORE_NAME, record);
  }
}

export async function incrementRetry(ids: string[]): Promise<void> {
  const db = await getDB();
  const tx = db.transaction(STORE_NAME, 'readwrite');
  const store = tx.objectStore(STORE_NAME);
  for (const id of ids) {
    const record = await store.get(id);
    if (record) {
      record.retry_count += 1;
      await store.put(record);
    }
  }
  await tx.done;
}

export async function getPendingCount(): Promise<number> {
  const db = await getDB();
  return db.countFromIndex(STORE_NAME, 'by_status', 'pending');
}

export async function pruneAckedEvents(beforeDate: Date): Promise<void> {
  const db = await getDB();
  const acked = await db.getAllFromIndex(STORE_NAME, 'by_status', 'acked');
  const tx = db.transaction(STORE_NAME, 'readwrite');
  const store = tx.objectStore(STORE_NAME);
  for (const event of acked) {
    if (new Date(event.occurred_at) < beforeDate) {
      await store.delete(event.client_event_id);
    }
  }
  await tx.done;
}
