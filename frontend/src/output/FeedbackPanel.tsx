import type { WritingFeedback } from '../api/types';

interface Props {
  feedback: WritingFeedback;
}

export default function FeedbackPanel({ feedback }: Props) {
  const scoreColor =
    feedback.overall_score >= 80
      ? 'bg-green-500'
      : feedback.overall_score >= 60
        ? 'bg-yellow-500'
        : 'bg-red-500';

  return (
    <div className="space-y-5">
      {/* Score bar */}
      <div>
        <div className="flex items-center justify-between mb-1">
          <span className="text-sm font-medium text-gray-700">Score</span>
          <span className="text-sm font-semibold">{feedback.overall_score} / 100</span>
        </div>
        <div className="h-3 bg-gray-200 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${scoreColor}`}
            style={{ width: `${feedback.overall_score}%` }}
          />
        </div>
      </div>

      {/* Errors */}
      {feedback.errors.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-gray-700 mb-2">
            Corrections ({feedback.errors.length})
          </h3>
          <div className="space-y-2">
            {feedback.errors.map((err, i) => (
              <div key={i} className="border border-gray-200 rounded-lg p-3">
                <div className="mb-1">
                  <span className="line-through text-red-600 text-sm">{err.original}</span>
                  <span className="mx-2 text-gray-400">&rarr;</span>
                  <span className="text-green-700 font-medium text-sm">{err.correction}</span>
                </div>
                <p className="text-xs text-gray-500">{err.explanation}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Revised text */}
      {feedback.revised_text && (
        <details className="group">
          <summary className="text-sm font-semibold text-gray-700 cursor-pointer hover:text-gray-900">
            Revised Text
          </summary>
          <div className="mt-2 bg-gray-50 rounded-lg p-3 text-sm text-gray-700 whitespace-pre-wrap">
            {feedback.revised_text}
          </div>
        </details>
      )}

      {/* Next actions */}
      {feedback.next_actions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-gray-700 mb-2">Next Steps</h3>
          <ul className="list-disc list-inside space-y-1">
            {feedback.next_actions.map((action, i) => (
              <li key={i} className="text-sm text-gray-600">{action}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
