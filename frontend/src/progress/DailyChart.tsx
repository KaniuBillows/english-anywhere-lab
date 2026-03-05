import type { DailyPoint } from '../api/types';
import { formatShortDate } from '../lib/date';

interface Props {
  points: DailyPoint[];
}

export default function DailyChart({ points }: Props) {
  const max = Math.max(...points.map((p) => p.minutes_learned), 1);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <p className="text-sm font-medium text-gray-700 mb-3">Daily Minutes</p>
      <div className="flex items-end gap-1" style={{ height: '120px' }}>
        {points.map((p) => {
          const pct = (p.minutes_learned / max) * 100;
          return (
            <div key={p.date} className="flex-1 flex flex-col items-center justify-end h-full">
              <div
                className="w-full bg-primary-500 rounded-t"
                style={{ height: `${Math.max(pct, 2)}%`, minHeight: '2px' }}
                title={`${p.minutes_learned} min`}
              />
            </div>
          );
        })}
      </div>
      <div className="flex gap-1 mt-1">
        {points.map((p, i) => (
          <div key={p.date} className="flex-1 text-center">
            {/* Show label for every ~Nth point to avoid crowding */}
            {(i % Math.ceil(points.length / 7) === 0 || i === points.length - 1) && (
              <span className="text-[10px] text-gray-400">{formatShortDate(p.date)}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
