import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router';
import { getPackDetail, enrollPack } from '../api/packs';
import type { Pack, Lesson } from '../api/types';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

type EnrollState = 'idle' | 'enrolling' | 'enrolled';

export default function PackDetailPage() {
  const { packId } = useParams<{ packId: string }>();
  const navigate = useNavigate();
  const [pack, setPack] = useState<Pack | null>(null);
  const [lessons, setLessons] = useState<Lesson[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [enrollState, setEnrollState] = useState<EnrollState>('idle');
  const [enrollError, setEnrollError] = useState('');

  useEffect(() => {
    if (!packId) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        const res = await getPackDetail(packId!);
        if (cancelled) return;
        setPack(res.pack);
        setLessons(res.lessons);
      } catch (err: unknown) {
        if (cancelled) return;
        const msg =
          err && typeof err === 'object' && 'message' in err
            ? (err as { message: string }).message
            : 'Failed to load pack';
        setError(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [packId]);

  async function handleEnroll() {
    if (!packId || enrollState !== 'idle') return;
    setEnrollState('enrolling');
    setEnrollError('');
    try {
      await enrollPack(packId);
      setEnrollState('enrolled');
    } catch (err: unknown) {
      setEnrollState('idle');
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to enroll';
      setEnrollError(msg);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <div className="p-4"><ErrorMessage message={error} /></div>;
  if (!pack) return null;

  const isAI = pack.source === 'ai';

  const lessonTypeBadge = (type: string) => {
    const colors: Record<string, string> = {
      writing: 'bg-purple-50 text-purple-700',
      listening: 'bg-yellow-50 text-yellow-700',
      speaking: 'bg-pink-50 text-pink-700',
      reading: 'bg-teal-50 text-teal-700',
      mixed: 'bg-indigo-50 text-indigo-700',
    };
    return colors[type] || 'bg-gray-100 text-gray-600';
  };

  return (
    <div className="p-4 pb-24 max-w-lg mx-auto">
      {/* Back nav */}
      <button
        onClick={() => navigate(-1)}
        className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-flex items-center gap-1"
      >
        &larr; Back
      </button>

      {/* Pack header */}
      <div className="mb-6">
        <h1 className="text-xl font-bold mb-2">{pack.title}</h1>
        {pack.description && (
          <p className="text-sm text-gray-600 mb-3">{pack.description}</p>
        )}
        <div className="flex flex-wrap gap-1.5 mb-4">
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
            {pack.domain}
          </span>
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">
            {pack.level}
          </span>
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
            {pack.estimated_minutes} min
          </span>
          <span
            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
              isAI ? 'bg-orange-50 text-orange-700' : 'bg-blue-50 text-blue-700'
            }`}
          >
            {isAI ? 'AI Generated' : 'Official'}
          </span>
        </div>

        {/* Enroll button */}
        <button
          onClick={handleEnroll}
          disabled={enrollState !== 'idle'}
          className={`w-full rounded-lg py-2.5 font-medium transition-colors ${
            enrollState === 'enrolled'
              ? 'bg-green-600 text-white'
              : 'bg-primary-600 text-white hover:bg-primary-700 disabled:opacity-50'
          }`}
        >
          {enrollState === 'idle' && 'Enroll'}
          {enrollState === 'enrolling' && 'Enrolling…'}
          {enrollState === 'enrolled' && 'Enrolled!'}
        </button>
        {enrollError && (
          <p className="text-sm text-danger-600 mt-2">{enrollError}</p>
        )}
      </div>

      {/* Lessons */}
      <h2 className="text-lg font-semibold mb-3">Lessons</h2>
      {lessons.length === 0 ? (
        <p className="text-sm text-gray-500">No lessons in this pack.</p>
      ) : (
        <ol className="space-y-2">
          {lessons.map((lesson) => {
            const hasOutputTasks =
              lesson.lesson_type === 'writing' ||
              lesson.lesson_type === 'mixed' ||
              (lesson.output_task_count && lesson.output_task_count > 0);

            const content = (
              <div className="flex items-center justify-between border border-gray-200 rounded-lg p-3">
                <div className="flex items-center gap-3 min-w-0">
                  <span className="text-sm text-gray-400 font-mono w-6 text-right shrink-0">
                    {lesson.position}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gray-900 truncate">
                      {lesson.title}
                    </p>
                    <span
                      className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium mt-1 ${lessonTypeBadge(lesson.lesson_type)}`}
                    >
                      {lesson.lesson_type}
                    </span>
                  </div>
                </div>
                {hasOutputTasks && (
                  <span className="text-xs text-gray-400 shrink-0">&rarr;</span>
                )}
              </div>
            );

            return hasOutputTasks ? (
              <li key={lesson.lesson_id}>
                <Link to={`/lessons/${lesson.lesson_id}/tasks`} className="block hover:shadow-sm transition-shadow">
                  {content}
                </Link>
              </li>
            ) : (
              <li key={lesson.lesson_id}>{content}</li>
            );
          })}
        </ol>
      )}
    </div>
  );
}
