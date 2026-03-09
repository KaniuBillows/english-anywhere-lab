import { useSyncStatus } from '../sync/useSyncStatus';

export default function SyncIndicator() {
  const { status, pendingCount, syncNow } = useSyncStatus();

  if (status === 'idle' && pendingCount === 0) return null;

  return (
    <button
      onClick={syncNow}
      className="fixed top-3 right-3 z-50 flex items-center gap-1.5 rounded-full bg-white/90 px-3 py-1.5 text-xs font-medium shadow-md backdrop-blur transition-colors hover:bg-white"
    >
      {status === 'syncing' && (
        <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
      )}
      {status === 'offline' && (
        <span className="text-gray-500">Offline</span>
      )}
      {status === 'error' && (
        <span className="text-red-500">Sync failed</span>
      )}
      {status === 'idle' && pendingCount > 0 && (
        <span className="text-yellow-600">Pending</span>
      )}
      {pendingCount > 0 && (
        <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-blue-500 px-1 text-[10px] text-white">
          {pendingCount}
        </span>
      )}
    </button>
  );
}
