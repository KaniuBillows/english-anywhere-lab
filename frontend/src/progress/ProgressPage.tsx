import { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { getSummary, getDaily } from '../api/progress';
import type { ProgressSummary, DailyPoint } from '../api/types';
import SummaryCards from './SummaryCards';
import DailyChart from './DailyChart';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

const RANGES = ['7d', '30d', '90d'];

export default function ProgressPage() {
  const [range, setRange] = useState('7d');
  const [summary, setSummary] = useState<ProgressSummary | null>(null);
  const [points, setPoints] = useState<DailyPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function load(r: string) {
    setLoading(true);
    setError('');
    try {
      const [s, d] = await Promise.all([getSummary(r), getDaily(r)]);
      setSummary(s);
      setPoints(d.points);
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to load progress';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load(range);
  }, [range]);

  return (
    <div>
      <h1 className="text-xl font-bold mb-4">Progress</h1>

      {/* Range pills */}
      <div className="flex gap-2 mb-4">
        {RANGES.map((r) => (
          <button
            key={r}
            onClick={() => setRange(r)}
            className={`rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
              range === r
                ? 'bg-primary-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            {r}
          </button>
        ))}
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : error ? (
        <ErrorMessage message={error} onRetry={() => load(range)} />
      ) : (
        <>
          {summary && <SummaryCards summary={summary} />}
          {points.length > 0 && <DailyChart points={points} />}
        </>
      )}

      {/* Report links */}
      <div className="mt-6 space-y-2">
        <Link
          to="/progress/weekly"
          className="block rounded-lg border border-gray-200 px-4 py-3 hover:bg-gray-50"
        >
          <span className="font-medium">Weekly Report</span>
          <span className="text-gray-400 ml-2">&rarr;</span>
        </Link>
        <Link
          to="/progress/monthly"
          className="block rounded-lg border border-gray-200 px-4 py-3 hover:bg-gray-50"
        >
          <span className="font-medium">Monthly Report</span>
          <span className="text-gray-400 ml-2">&rarr;</span>
        </Link>
      </div>
    </div>
  );
}
