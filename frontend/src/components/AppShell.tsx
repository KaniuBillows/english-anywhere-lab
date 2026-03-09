import { Outlet } from 'react-router';
import BottomNav from './BottomNav';
import SyncProvider from '../sync/SyncProvider';
import SyncIndicator from './SyncIndicator';

export default function AppShell() {
  return (
    <SyncProvider>
      <div className="min-h-screen pb-16">
        <SyncIndicator />
        <div className="max-w-lg mx-auto px-4 py-4">
          <Outlet />
        </div>
        <BottomNav />
      </div>
    </SyncProvider>
  );
}
