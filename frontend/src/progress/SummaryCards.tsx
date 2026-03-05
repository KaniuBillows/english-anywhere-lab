import type { ProgressSummary } from '../api/types';

interface Props {
  summary: ProgressSummary;
}

export default function SummaryCards({ summary }: Props) {
  const items = [
    { label: 'Total Minutes', value: summary.total_minutes },
    { label: 'Active Days', value: summary.active_days },
    { label: 'Review Accuracy', value: `${Math.round(summary.review_accuracy * 100)}%` },
    { label: 'Cards Reviewed', value: summary.cards_reviewed },
    { label: 'Streak', value: `${summary.streak_count}d` },
  ];

  return (
    <div className="grid grid-cols-2 gap-3 mb-4">
      {items.map((item) => (
        <div
          key={item.label}
          className="rounded-lg border border-gray-200 bg-white p-3"
        >
          <p className="text-xs text-gray-500">{item.label}</p>
          <p className="text-lg font-bold mt-0.5">{item.value}</p>
        </div>
      ))}
    </div>
  );
}
