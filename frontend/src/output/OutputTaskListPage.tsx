import { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router';
import { listOutputTasks } from '../api/output';
import type { OutputTask } from '../api/types';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

export default function OutputTaskListPage() {
  const { lessonId } = useParams<{ lessonId: string }>();
  const navigate = useNavigate();
  const [tasks, setTasks] = useState<OutputTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!lessonId) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        const res = await listOutputTasks(lessonId!);
        if (cancelled) return;
        setTasks(res.items);
      } catch (err: unknown) {
        if (cancelled) return;
        const msg =
          err && typeof err === 'object' && 'message' in err
            ? (err as { message: string }).message
            : 'Failed to load tasks';
        setError(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [lessonId]);

  if (loading) return <LoadingSpinner />;

  return (
    <div className="p-4 pb-24 max-w-lg mx-auto">
      <button
        onClick={() => navigate(-1)}
        className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-flex items-center gap-1"
      >
        &larr; Back
      </button>

      <h1 className="text-xl font-bold mb-4">Output Tasks</h1>

      {error && <ErrorMessage message={error} />}

      {!error && tasks.length === 0 && (
        <p className="text-sm text-gray-500">No output tasks for this lesson.</p>
      )}

      <div className="space-y-3">
        {tasks.map((task) => (
          <Link
            key={task.id}
            to={`/output-tasks/${task.id}?lessonId=${lessonId}`}
            state={{ task }}
            className="block border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow"
          >
            <p className="text-sm text-gray-900 mb-2 line-clamp-2">{task.prompt_text}</p>
            <div className="flex flex-wrap gap-1.5">
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-50 text-purple-700">
                {task.task_type}
              </span>
              {task.level && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">
                  {task.level}
                </span>
              )}
              {task.exercise_type && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                  {task.exercise_type}
                </span>
              )}
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
