'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { apiClient } from '@/lib/api-client';

export function SourcesContainer() {
  const { data: sourcesData, isLoading } = useQuery({
    queryKey: ['sources'],
    queryFn: () => apiClient.sources.list({ per_page: 100 }),
  });

  const sources = sourcesData?.data ?? [];

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">ソース一覧</h1>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-4 animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-1/3 mb-2" />
              <div className="h-3 bg-gray-100 rounded w-2/3" />
            </div>
          ))}
        </div>
      )}

      <div className="space-y-3">
        {sources.map((source) => {
          return (
            <div
              key={source.id}
              className="bg-white rounded-xl border border-gray-200 p-4 flex items-center justify-between gap-4"
            >
              <Link href={`/sources/${source.id}`} className="min-w-0 flex-1 hover:opacity-75">
                <p className="font-medium text-gray-900 text-sm truncate">{source.name}</p>
                <p className="text-xs text-gray-500 mt-0.5">
                  {source.category} · {source.language.toUpperCase()}
                </p>
              </Link>
            </div>
          );
        })}
      </div>
    </main>
  );
}
