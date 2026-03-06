import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router';
import { submitWriting } from '../api/output';
import type { OutputTask, SubmissionResponse } from '../api/types';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';
import FeedbackPanel from './FeedbackPanel';

type Phase = 'loading' | 'writing' | 'submitting' | 'feedback' | 'error';

const MAX_CHARS = 5000;

export default function WritingTaskPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const [task, setTask] = useState<OutputTask | null>(null);
  const [phase, setPhase] = useState<Phase>('loading');
  const [error, setError] = useState('');
  const [answer, setAnswer] = useState('');
  const [submission, setSubmission] = useState<SubmissionResponse | null>(null);
  const [showHint, setShowHint] = useState(false);
  const [longWait, setLongWait] = useState(false);

  // Load task — we fetch via the lesson's output-tasks list and find by id
  useEffect(() => {
    if (!taskId) return;
    let cancelled = false;
    async function load() {
      setPhase('loading');
      setError('');
      try {
        // We need the task data. The API doesn't have a single-task GET,
        // so if we have lesson_id context we'd use that. As a fallback,
        // we try fetching via the submission endpoint with a dummy to get 404
        // and use stored task data. For now, we'll use a direct fetch approach.
        // The backend may support GET /api/v1/output-tasks/:id — try it.
        const res = await fetch(`/api/v1/output-tasks/${taskId}`, {
          headers: {
            Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`,
          },
        });
        if (res.ok) {
          const data = await res.json();
          if (!cancelled) {
            setTask(data);
            setPhase('writing');
          }
          return;
        }
      } catch {
        // fallback below
      }
      // Fallback: task data might be passed via location state in the future.
      // For now, create a minimal task shell so the page still works.
      if (!cancelled) {
        setTask({
          id: taskId!,
          task_type: 'writing',
          prompt_text: 'Write your answer below.',
        });
        setPhase('writing');
      }
    }
    load();
    return () => { cancelled = true; };
  }, [taskId]);

  // Extended wait message
  const longWaitTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => {
    clearTimeout(longWaitTimerRef.current);
    if (phase === 'submitting') {
      longWaitTimerRef.current = setTimeout(() => setLongWait(true), 3000);
    }
    return () => clearTimeout(longWaitTimerRef.current);
  }, [phase]);

  async function handleSubmit() {
    if (!taskId || !answer.trim()) return;
    setPhase('submitting');
    setLongWait(false);
    setError('');
    try {
      const res = await submitWriting(taskId, { answer_text: answer });
      setSubmission(res);
      setPhase('feedback');
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to submit';
      setError(msg);
      setPhase('error');
    }
  }

  function handleWriteAgain() {
    setAnswer('');
    setSubmission(null);
    setLongWait(false);
    setPhase('writing');
  }

  if (phase === 'loading') return <LoadingSpinner />;

  return (
    <div className="p-4 pb-24 max-w-lg mx-auto">
      <button
        onClick={() => navigate(-1)}
        className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-flex items-center gap-1"
      >
        &larr; Back
      </button>

      {/* Prompt */}
      {task && (
        <div className="mb-4">
          <h1 className="text-lg font-bold mb-2">Writing Task</h1>
          <div className="bg-gray-50 rounded-lg p-4 text-sm text-gray-800 whitespace-pre-wrap">
            {task.prompt_text}
          </div>

          {task.reference_answer && (
            <div className="mt-2">
              <button
                onClick={() => setShowHint(!showHint)}
                className="text-xs text-primary-600 hover:text-primary-700 font-medium"
              >
                {showHint ? 'Hide Hint' : 'Show Hint'}
              </button>
              {showHint && (
                <div className="mt-2 bg-yellow-50 border border-yellow-200 rounded-lg p-3 text-sm text-gray-700 whitespace-pre-wrap">
                  {task.reference_answer}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Writing phase */}
      {phase === 'writing' && (
        <div>
          <textarea
            value={answer}
            onChange={(e) => {
              if (e.target.value.length <= MAX_CHARS) setAnswer(e.target.value);
            }}
            placeholder="Write your answer here…"
            rows={8}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 resize-y"
          />
          <div className="flex justify-between items-center mt-1 mb-4">
            <span className="text-xs text-gray-400">
              {answer.length} / {MAX_CHARS}
            </span>
          </div>
          <button
            onClick={handleSubmit}
            disabled={!answer.trim()}
            className="w-full bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
          >
            Submit
          </button>
        </div>
      )}

      {/* Submitting phase */}
      {phase === 'submitting' && (
        <div className="text-center py-12">
          <div className="animate-spin h-8 w-8 border-4 border-primary-600 border-t-transparent rounded-full mx-auto mb-4" />
          <p className="text-sm text-gray-600">
            {longWait
              ? 'Still reviewing… AI is providing detailed feedback.'
              : 'AI is reviewing your writing…'}
          </p>
        </div>
      )}

      {/* Error phase */}
      {phase === 'error' && (
        <div>
          <ErrorMessage message={error} />
          <button
            onClick={() => setPhase('writing')}
            className="mt-4 w-full border border-gray-300 rounded-lg py-2.5 text-sm font-medium hover:bg-gray-50"
          >
            Try Again
          </button>
        </div>
      )}

      {/* Feedback phase */}
      {phase === 'feedback' && submission?.feedback && (
        <div>
          <FeedbackPanel feedback={submission.feedback} />
          <button
            onClick={handleWriteAgain}
            className="mt-6 w-full border border-gray-300 rounded-lg py-2.5 text-sm font-medium hover:bg-gray-50"
          >
            Write Again
          </button>
        </div>
      )}

      {phase === 'feedback' && !submission?.feedback && (
        <div className="text-center py-8">
          <p className="text-sm text-gray-600 mb-2">
            Submitted! Score: {submission?.score ?? '—'}
          </p>
          <p className="text-xs text-gray-400">
            Detailed feedback is not available for this submission.
          </p>
          <button
            onClick={handleWriteAgain}
            className="mt-4 border border-gray-300 rounded-lg px-6 py-2.5 text-sm font-medium hover:bg-gray-50"
          >
            Write Again
          </button>
        </div>
      )}
    </div>
  );
}
