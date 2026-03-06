import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router';
import { submitWriting } from '../api/output';
import type { OutputTask, SubmissionResponse } from '../api/types';
import ErrorMessage from '../components/ErrorMessage';
import FeedbackPanel from './FeedbackPanel';

type Phase = 'writing' | 'submitting' | 'feedback' | 'error';

const MAX_CHARS = 5000;

export default function WritingTaskPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const location = useLocation();

  // Task data is passed via route state from OutputTaskListPage
  const task: OutputTask = (location.state as { task?: OutputTask })?.task ?? {
    id: taskId ?? '',
    task_type: 'writing',
    prompt_text: '',
  };

  const [phase, setPhase] = useState<Phase>('writing');
  const [error, setError] = useState('');
  const [answer, setAnswer] = useState('');
  const [submission, setSubmission] = useState<SubmissionResponse | null>(null);
  const [showHint, setShowHint] = useState(false);
  const [longWait, setLongWait] = useState(false);

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

  return (
    <div className="p-4 pb-24 max-w-lg mx-auto">
      <button
        onClick={() => navigate(-1)}
        className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-flex items-center gap-1"
      >
        &larr; Back
      </button>

      {/* Prompt */}
      <div className="mb-4">
        <h1 className="text-lg font-bold mb-2">Writing Task</h1>
        {task.prompt_text ? (
          <div className="bg-gray-50 rounded-lg p-4 text-sm text-gray-800 whitespace-pre-wrap">
            {task.prompt_text}
          </div>
        ) : (
          <p className="text-sm text-gray-400 italic">
            No prompt available. Please navigate here from a lesson's task list.
          </p>
        )}

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
