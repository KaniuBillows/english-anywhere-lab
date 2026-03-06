import { useEffect, useState } from 'react';
import { getWeeklyReport } from '../api/progress';
import type { WeeklyReportResponse } from '../api/types';
import { getWeekStart, addDays, formatDate } from '../lib/date';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

export default function WeeklyReport() {
  const [weekStart, setWeekStart] = useState(() => getWeekStart());
  const [report, setReport] = useState<WeeklyReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function load(ws: string) {
    setLoading(true);
    setError('');
    try {
      const data = await getWeeklyReport(ws);
      setReport(data);
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to load report';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load(weekStart);
  }, [weekStart]);

  function prevWeek() {
    setWeekStart((ws) => addDays(ws, -7));
  }
  function nextWeek() {
    setWeekStart((ws) => addDays(ws, 7));
  }

  return (
    <div>
      {/* Week picker */}
      <div className="flex items-center justify-between mb-4">
        <button onClick={prevWeek} className="p-2 hover:bg-gray-100 rounded-lg">&larr;</button>
        <h1 className="text-lg font-bold">Week of {formatDate(weekStart)}</h1>
        <button onClick={nextWeek} className="p-2 hover:bg-gray-100 rounded-lg">&rarr;</button>
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : error ? (
        <ErrorMessage message={error} onRetry={() => load(weekStart)} />
      ) : report ? (
        <div className="space-y-4">
          {/* Summary row */}
          <div className="grid grid-cols-3 gap-2 text-center">
            <Stat label="Minutes" value={report.total_minutes} />
            <Stat label="Active Days" value={`${report.active_days}/${report.weekly_goal_days}`} />
            <Stat label="Cards" value={report.cards_reviewed} />
          </div>

          {/* Goal progress */}
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span>Weekly Goal</span>
              <span className={report.goal_achieved ? 'text-success-600' : 'text-gray-500'}>
                {report.goal_achieved ? 'Achieved!' : `${report.active_days}/${report.weekly_goal_days} days`}
              </span>
            </div>
            <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${report.goal_achieved ? 'bg-success-500' : 'bg-primary-500'}`}
                style={{ width: `${Math.min((report.active_days / report.weekly_goal_days) * 100, 100)}%` }}
              />
            </div>
          </div>

          {/* Review health */}
          <div>
            <p className="text-sm font-medium mb-1">Review Health</p>
            <div className="flex h-4 rounded-full overflow-hidden">
              {report.review_health.total > 0 ? (
                <>
                  <Bar color="bg-danger-500" value={report.review_health.again} total={report.review_health.total} />
                  <Bar color="bg-warning-500" value={report.review_health.hard} total={report.review_health.total} />
                  <Bar color="bg-success-500" value={report.review_health.good} total={report.review_health.total} />
                  <Bar color="bg-primary-500" value={report.review_health.easy} total={report.review_health.total} />
                </>
              ) : (
                <div className="w-full bg-gray-200" />
              )}
            </div>
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Again: {report.review_health.again}</span>
              <span>Hard: {report.review_health.hard}</span>
              <span>Good: {report.review_health.good}</span>
              <span>Easy: {report.review_health.easy}</span>
            </div>
          </div>

          {/* Daily breakdown */}
          <div>
            <p className="text-sm font-medium mb-2">Daily Breakdown</p>
            <div className="space-y-1">
              {report.daily_breakdown.map((d) => (
                <div key={d.date} className="flex justify-between text-sm py-1 border-b border-gray-100">
                  <span className="text-gray-600">{formatDate(d.date)}</span>
                  <span>{d.minutes_learned} min</span>
                  <span>{d.cards_reviewed} cards</span>
                </div>
              ))}
            </div>
          </div>

          {/* Prev week comparison */}
          {report.previous_week_comparison && (
            <div>
              <p className="text-sm font-medium mb-2">vs. Previous Week</p>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <Delta label="Minutes" value={report.previous_week_comparison.minutes_delta} />
                <Delta label="Active Days" value={report.previous_week_comparison.active_days_delta} />
                <Delta label="Cards Reviewed" value={report.previous_week_comparison.cards_reviewed_delta} />
                <Delta label="Lessons" value={report.previous_week_comparison.lessons_delta} />
              </div>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-2">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-lg font-bold">{value}</p>
    </div>
  );
}

function Bar({ color, value, total }: { color: string; value: number; total: number }) {
  if (value === 0) return null;
  return <div className={color} style={{ width: `${(value / total) * 100}%` }} />;
}

function Delta({ label, value }: { label: string; value: number }) {
  const positive = value > 0;
  const zero = value === 0;
  return (
    <div className="flex justify-between">
      <span className="text-gray-500">{label}</span>
      <span className={zero ? 'text-gray-400' : positive ? 'text-success-600' : 'text-danger-600'}>
        {positive ? '↑' : value < 0 ? '↓' : '–'} {Math.abs(value)}
      </span>
    </div>
  );
}
