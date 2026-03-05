import { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { getTodayPlan } from '../api/plans';
import type { DailyPlan } from '../api/types';
import { getUserTimezone, formatDate } from '../lib/date';
import TaskCard from './TaskCard';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

export default function TodayPage() {
  const [plan, setPlan] = useState<DailyPlan | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function loadPlan() {
    setLoading(true);
    setError('');
    try {
      const res = await getTodayPlan(getUserTimezone());
      setPlan(res.daily_plan);
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to load plan';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadPlan();
  }, []);

  if (loading) return <LoadingSpinner />;

  if (error) {
    return (
      <div>
        <h1 className="text-xl font-bold mb-4">Today's Plan</h1>
        <ErrorMessage message={error} onRetry={loadPlan} />
      </div>
    );
  }

  if (!plan || plan.tasks.length === 0) {
    return (
      <div className="text-center py-12">
        <h1 className="text-xl font-bold mb-2">No plan yet</h1>
        <p className="text-gray-500 mb-4">Set up your learning profile to get started.</p>
        <Link
          to="/onboarding"
          className="inline-block bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
        >
          Get Started
        </Link>
      </div>
    );
  }

  const completedCount = plan.tasks.filter((t) => t.status === 'completed').length;

  return (
    <div>
      <div className="mb-4">
        <h1 className="text-xl font-bold">{formatDate(plan.date)}</h1>
        <div className="flex items-center gap-3 mt-1 text-sm text-gray-500">
          <span>{plan.total_estimated_minutes} min</span>
          <span className="inline-block rounded-full bg-primary-100 text-primary-700 px-2 py-0.5 text-xs font-medium">
            {plan.mode}
          </span>
          <span>
            {completedCount}/{plan.tasks.length} done
          </span>
        </div>
      </div>

      <div className="space-y-3">
        {plan.tasks.map((task) => (
          <TaskCard
            key={task.task_id}
            task={task}
            planId={plan.plan_id}
            onComplete={loadPlan}
          />
        ))}
      </div>
    </div>
  );
}
