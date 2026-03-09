import type { Pack } from '../api/types';
import { Link } from 'react-router';

interface Props {
  pack: Pack;
}

export default function PackCard({ pack }: Props) {
  const isAI = pack.source === 'ai';
  const stripeColor = isAI ? 'border-l-orange-400' : 'border-l-blue-500';

  return (
    <Link
      to={`/packs/${pack.id}`}
      className={`block border border-gray-200 rounded-lg p-4 border-l-4 ${stripeColor} hover:shadow-sm transition-shadow`}
    >
      <h3 className="font-semibold text-gray-900 mb-2">{pack.title}</h3>
      {pack.description && (
        <p className="text-sm text-gray-500 mb-3 line-clamp-2">{pack.description}</p>
      )}
      <div className="flex flex-wrap gap-1.5">
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
          {isAI ? 'AI' : 'Official'}
        </span>
      </div>
    </Link>
  );
}
