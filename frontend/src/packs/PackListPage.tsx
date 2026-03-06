import { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { listPacks } from '../api/packs';
import type { Pack } from '../api/types';
import PackCard from './PackCard';
import LoadingSpinner from '../components/LoadingSpinner';
import ErrorMessage from '../components/ErrorMessage';

const DOMAINS = ['All', 'General', 'Business', 'Travel', 'Tech', 'Academic', 'Daily Life'];
const LEVELS = ['All', 'A1', 'A2', 'B1', 'B2', 'C1', 'C2'];
const SOURCES = ['all', 'official', 'ai'] as const;

export default function PackListPage() {
  const [packs, setPacks] = useState<Pack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [domain, setDomain] = useState('All');
  const [level, setLevel] = useState('All');
  const [source, setSource] = useState<'all' | 'official' | 'ai'>('all');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 10;

  useEffect(() => {
    setPage(1);
  }, [domain, level, source]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        const params: Record<string, string | number | undefined> = {
          page,
          page_size: pageSize,
        };
        if (domain !== 'All') params.domain = domain;
        if (level !== 'All') params.level = level;
        if (source !== 'all') params.source = source;

        const res = await listPacks(params);
        if (cancelled) return;
        if (page === 1) {
          setPacks(res.items);
        } else {
          setPacks((prev) => [...prev, ...res.items]);
        }
        setTotal(res.total);
      } catch (err: unknown) {
        if (cancelled) return;
        const msg =
          err && typeof err === 'object' && 'message' in err
            ? (err as { message: string }).message
            : 'Failed to load packs';
        setError(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [domain, level, source, page]);

  const hasMore = packs.length < total;

  return (
    <div className="p-4 pb-24 max-w-lg mx-auto">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-bold">Packs</h1>
        <Link
          to="/packs/generate"
          className="text-sm font-medium text-primary-600 hover:text-primary-700"
        >
          + Generate
        </Link>
      </div>

      {/* Domain filter */}
      <div className="flex flex-wrap gap-1.5 mb-3">
        {DOMAINS.map((d) => (
          <button
            key={d}
            onClick={() => setDomain(d)}
            className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
              domain === d
                ? 'border-primary-600 bg-primary-50 text-primary-700'
                : 'border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            {d}
          </button>
        ))}
      </div>

      {/* Level filter */}
      <div className="flex flex-wrap gap-1.5 mb-3">
        {LEVELS.map((l) => (
          <button
            key={l}
            onClick={() => setLevel(l)}
            className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
              level === l
                ? 'border-primary-600 bg-primary-50 text-primary-700'
                : 'border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            {l}
          </button>
        ))}
      </div>

      {/* Source toggle */}
      <div className="flex gap-1.5 mb-4">
        {SOURCES.map((s) => (
          <button
            key={s}
            onClick={() => setSource(s)}
            className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors capitalize ${
              source === s
                ? 'border-primary-600 bg-primary-50 text-primary-700'
                : 'border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      {error && <ErrorMessage message={error} />}

      {/* Pack list */}
      <div className="space-y-3">
        {packs.map((pack) => (
          <PackCard key={pack.id} pack={pack} />
        ))}
      </div>

      {loading && <LoadingSpinner />}

      {!loading && packs.length === 0 && !error && (
        <div className="text-center py-12">
          <p className="text-gray-500 mb-4">No packs found</p>
          <Link
            to="/packs/generate"
            className="inline-block bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
          >
            Generate an AI Pack
          </Link>
        </div>
      )}

      {!loading && hasMore && (
        <button
          onClick={() => setPage((p) => p + 1)}
          className="mt-4 w-full border border-gray-300 rounded-lg py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Load More
        </button>
      )}
    </div>
  );
}
