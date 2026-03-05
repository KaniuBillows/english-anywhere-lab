import { useEffect, useState, useRef } from 'react';
import { getReviewQueue, submitReview } from '../api/reviews';
import type { ReviewCard, Rating } from '../api/types';
import FlashCard from './FlashCard';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

export default function ReviewPage() {
  const [cards, setCards] = useState<ReviewCard[]>([]);
  const [currentIdx, setCurrentIdx] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [sessionDone, setSessionDone] = useState(false);
  const showTimeRef = useRef(Date.now());

  // Session stats
  const [stats, setStats] = useState({ reviewed: 0, again: 0, good: 0 });

  async function loadQueue() {
    setLoading(true);
    setError('');
    try {
      const res = await getReviewQueue(20);
      setCards(res.cards);
      setCurrentIdx(0);
      setFlipped(false);
      setSessionDone(res.cards.length === 0);
      showTimeRef.current = Date.now();
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to load reviews';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadQueue();
  }, []);

  async function handleRate(rating: Rating) {
    const card = cards[currentIdx];
    if (!card || submitting) return;

    const responseMs = Date.now() - showTimeRef.current;
    const idempotencyKey = crypto.randomUUID();

    setSubmitting(true);
    try {
      await submitReview(
        {
          card_id: card.card_id,
          user_card_state_id: card.user_card_state_id,
          rating,
          reviewed_at: new Date().toISOString(),
          response_ms: responseMs,
          client_event_id: idempotencyKey,
        },
        idempotencyKey,
      );

      setStats((s) => ({
        reviewed: s.reviewed + 1,
        again: s.again + (rating === 'again' ? 1 : 0),
        good: s.good + (rating === 'good' || rating === 'easy' ? 1 : 0),
      }));

      const nextIdx = currentIdx + 1;
      if (nextIdx >= cards.length) {
        setSessionDone(true);
      } else {
        setCurrentIdx(nextIdx);
        setFlipped(false);
        showTimeRef.current = Date.now();
      }
    } catch {
      // user can retry
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} onRetry={loadQueue} />;

  if (sessionDone) {
    return (
      <div className="text-center py-12">
        <h1 className="text-xl font-bold mb-2">
          {stats.reviewed > 0 ? 'Session Complete!' : 'No cards to review'}
        </h1>
        {stats.reviewed > 0 && (
          <div className="text-sm text-gray-500 mb-6 space-y-1">
            <p>Reviewed: {stats.reviewed}</p>
            <p>Good/Easy: {stats.good}</p>
            <p>Again: {stats.again}</p>
          </div>
        )}
        <button
          onClick={() => {
            setStats({ reviewed: 0, again: 0, good: 0 });
            loadQueue();
          }}
          className="bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
        >
          Load More
        </button>
      </div>
    );
  }

  const card = cards[currentIdx];

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">Review</h1>
        <span className="text-sm text-gray-500">
          {currentIdx + 1} / {cards.length}
        </span>
      </div>

      <FlashCard card={card} flipped={flipped} onFlip={() => setFlipped(true)} />

      {flipped && (
        <div className="grid grid-cols-4 gap-2 mt-4">
          <button
            onClick={() => handleRate('again')}
            disabled={submitting}
            className="rounded-lg bg-danger-500 text-white py-2.5 text-sm font-medium hover:bg-danger-600 disabled:opacity-50"
          >
            Again
          </button>
          <button
            onClick={() => handleRate('hard')}
            disabled={submitting}
            className="rounded-lg bg-warning-500 text-white py-2.5 text-sm font-medium hover:bg-warning-600 disabled:opacity-50"
          >
            Hard
          </button>
          <button
            onClick={() => handleRate('good')}
            disabled={submitting}
            className="rounded-lg bg-success-500 text-white py-2.5 text-sm font-medium hover:bg-success-600 disabled:opacity-50"
          >
            Good
          </button>
          <button
            onClick={() => handleRate('easy')}
            disabled={submitting}
            className="rounded-lg bg-primary-500 text-white py-2.5 text-sm font-medium hover:bg-primary-600 disabled:opacity-50"
          >
            Easy
          </button>
        </div>
      )}
    </div>
  );
}
