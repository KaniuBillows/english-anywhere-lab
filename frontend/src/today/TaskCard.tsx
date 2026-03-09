import { useState } from 'react';
import { completeTask } from '../api/plans';
import type { PlanTask } from '../api/types';
import { useSyncStatus } from '../sync/useSyncStatus';

const TYPE_COLORS: Record<string, string> = {
  input: 'bg-blue-500',
  retrieval_quiz: 'bg-green-500',
  review: 'bg-green-500',
  output: 'bg-orange-500',
};

interface Props {
  task: PlanTask;
  planId: string;
  onComplete: () => void;
}

export default function TaskCard({ task, planId, onComplete }: Props) {
  const [submitting, setSubmitting] = useState(false);
  const isCompleted = task.status === 'completed';
  const { enqueue } = useSyncStatus();

  const stripe = TYPE_COLORS[task.task_type] || 'bg-gray-400';

  async function handleComplete() {
    setSubmitting(true);
    try {
      await completeTask(planId, task.task_id, {
        completed_at: new Date().toISOString(),
      });
      void enqueue('task_completed', {
        task_id: task.task_id,
        plan_id: planId,
      });
      onComplete();
    } catch {
      // ignore — user can retry
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div
      className={`flex rounded-lg border overflow-hidden ${
        isCompleted ? 'opacity-60' : ''
      }`}
    >
      <div className={`w-1.5 shrink-0 ${stripe}`} />
      <div className="flex-1 p-3 flex items-center justify-between gap-3">
        <div>
          <p className={`font-medium ${isCompleted ? 'line-through text-gray-400' : ''}`}>
            {task.title}
          </p>
          <p className="text-xs text-gray-500 mt-0.5">
            {task.estimated_minutes} min · {task.task_type.replace('_', ' ')}
          </p>
        </div>
        {isCompleted ? (
          <span className="text-success-600 text-lg shrink-0">✓</span>
        ) : (
          <button
            onClick={handleComplete}
            disabled={submitting}
            className="shrink-0 rounded-lg bg-primary-600 text-white px-3 py-1.5 text-sm font-medium hover:bg-primary-700 disabled:opacity-50"
          >
            {submitting ? '…' : 'Complete'}
          </button>
        )}
      </div>
    </div>
  );
}
