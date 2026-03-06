import { useEffect, useState } from 'react';
import { getMonthlyReport } from '../api/progress';
import type { MonthlyReportResponse } from '../api/types';
import { getMonth, addMonths } from '../lib/date';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

export default function MonthlyReport() {
  const [month, setMonth] = useState(() => getMonth());
  const [report, setReport] = useState<MonthlyReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function load(m: string) {
    setLoading(true);
    setError('');
    try {
      const data = await getMonthlyReport(m);
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
    load(month);
  }, [month]);

  function prevMonth() {
    setMonth((m) => addMonths(m, -1));
  }
  function nextMonth() {
    setMonth((m) => addMonths(m, 1));
  }

  const monthLabel = new Date(month + '-01').toLocaleDateString(undefined, {
    month: 'long',
    year: 'numeric',
  });

  return (
    <div>
      {/* Month picker */}
      <div className="flex items-center justify-between mb-4">
        <button onClick={prevMonth} className="p-2 hover:bg-gray-100 rounded-lg">&larr;</button>
        <h1 className="text-lg font-bold">{monthLabel}</h1>
        <button onClick={nextMonth} className="p-2 hover:bg-gray-100 rounded-lg">&rarr;</button>
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : error ? (
        <ErrorMessage message={error} onRetry={() => load(month)} />
      ) : report ? (
        <div className="space-y-4">
          {/* Summary row */}
          <div className="grid grid-cols-3 gap-2 text-center">
            <Stat label="Minutes" value={report.total_minutes} />
            <Stat label="Active Days" value={`${report.active_days}/${report.days_in_month}`} />
            <Stat label="Cards" value={report.cards_reviewed} />
          </div>

          {/* Skill breakdown */}
          <div>
            <p className="text-sm font-medium mb-2">Skill Breakdown</p>
            <div className="space-y-2">
              {(['listening', 'speaking', 'writing', 'reading'] as const).map((skill) => {
                const metric = report.skill_breakdown[skill];
                return (
                  <div key={skill}>
                    <div className="flex justify-between text-sm mb-0.5">
                      <span className="capitalize">{skill}</span>
                      <span className="text-gray-500">{Math.round(metric.percentage * 100)}%</span>
                    </div>
                    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-primary-500 rounded-full"
                        style={{ width: `${metric.percentage * 100}%` }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Weaknesses */}
          {report.weaknesses.length > 0 && (
            <div>
              <p className="text-sm font-medium mb-2">Areas to Improve</p>
              <div className="space-y-2">
                {report.weaknesses.map((w, i) => (
                  <div key={i} className="rounded-lg border border-warning-500/30 bg-warning-50 p-3">
                    <p className="font-medium text-sm capitalize">{w.skill}</p>
                    <p className="text-xs text-gray-600 mt-0.5">{w.reason}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

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
            {report.review_health.accuracy != null && (
              <p className="text-xs text-gray-500 mt-1">
                Accuracy: {Math.round(report.review_health.accuracy * 100)}%
              </p>
            )}
          </div>

          {/* Prev month comparison */}
          {report.previous_month_comparison && (
            <div>
              <p className="text-sm font-medium mb-2">vs. Previous Month</p>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <Delta label="Minutes" value={report.previous_month_comparison.minutes_delta} />
                <Delta label="Active Days" value={report.previous_month_comparison.active_days_delta} />
                <Delta label="Cards Reviewed" value={report.previous_month_comparison.cards_reviewed_delta} />
                <Delta label="Lessons" value={report.previous_month_comparison.lessons_delta} />
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
