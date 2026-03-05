import type { ReviewCard } from '../api/types';

interface Props {
  card: ReviewCard;
  flipped: boolean;
  onFlip: () => void;
}

export default function FlashCard({ card, flipped, onFlip }: Props) {
  return (
    <div
      className="cursor-pointer"
      style={{ perspective: '1000px' }}
      onClick={() => !flipped && onFlip()}
    >
      <div
        className="relative w-full transition-transform duration-500"
        style={{
          transformStyle: 'preserve-3d',
          transform: flipped ? 'rotateY(180deg)' : 'rotateY(0deg)',
          minHeight: '200px',
        }}
      >
        {/* Front */}
        <div
          className="absolute inset-0 flex items-center justify-center rounded-xl bg-white border border-gray-200 shadow-sm p-6"
          style={{ backfaceVisibility: 'hidden' }}
        >
          <p className="text-xl font-semibold text-center">{card.front_text}</p>
        </div>

        {/* Back */}
        <div
          className="absolute inset-0 flex flex-col items-center justify-center rounded-xl bg-white border border-gray-200 shadow-sm p-6"
          style={{ backfaceVisibility: 'hidden', transform: 'rotateY(180deg)' }}
        >
          <p className="text-xl font-semibold text-center mb-3">{card.back_text}</p>
          {card.example_text && (
            <p className="text-sm italic text-gray-500 text-center">{card.example_text}</p>
          )}
        </div>
      </div>

      {!flipped && (
        <p className="text-center text-xs text-gray-400 mt-3">Tap to flip</p>
      )}
    </div>
  );
}
