'use client';

import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { apiClient } from '@/lib/api-client';
import { ArticleCard, ArticleCardSkeleton } from '@/components/ArticleCard';
import Link from 'next/link';

interface Props {
  sourceId: string;
}

export function SourceDetailContainer({ sourceId }: Props) {
  const { status } = useAuth();
  const isAuthenticated = status === 'authenticated';

  const { data: sourceData, isLoading: sourceLoading } = useQuery({
    queryKey: ['sources', sourceId],
    queryFn: () => apiClient.sources.getById(sourceId),
  });

  const { data: articlesData, isLoading: articlesLoading } = useQuery({
    queryKey: ['articles', 'by-source', sourceId],
    queryFn: () => apiClient.articles.list({ source_id: sourceId, per_page: 20 } as Parameters<typeof apiClient.articles.list>[0]),
  });

  const source = sourceData?.data?.source;
  const articles = articlesData?.data ?? [];

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <Link href="/sources" className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-block">
        ← ソース一覧
      </Link>

      {sourceLoading ? (
        <div className="animate-pulse mb-6">
          <div className="h-6 bg-gray-200 rounded w-1/3 mb-2" />
          <div className="h-4 bg-gray-100 rounded w-1/4" />
        </div>
      ) : source ? (
        <div className="bg-white rounded-xl border border-gray-200 p-5 mb-6">
          <h1 className="text-lg font-semibold text-gray-900">{source.name}</h1>
          <p className="text-xs text-gray-500 mt-1">{source.category} · {source.language.toUpperCase()}</p>
          {source.description && <p className="text-sm text-gray-600 mt-2">{source.description}</p>}
          <a href={source.site_url} target="_blank" rel="noopener noreferrer" className="text-xs text-primary-600 hover:underline mt-1 inline-block">
            {source.site_url} ↗
          </a>
        </div>
      ) : null}

      <h2 className="text-sm font-medium text-gray-700 mb-3">記事一覧</h2>

      {articlesLoading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => <ArticleCardSkeleton key={i} />)}
        </div>
      )}

      <div className="space-y-4">
        {articles.map((article: import('@/types/api').ArticleSummary) => (
          <ArticleCard
            key={article.id}
            article={article}
            isAuthenticated={isAuthenticated}
            showBookmark={isAuthenticated}
          />
        ))}
      </div>

      {articles.length === 0 && !articlesLoading && (
        <p className="text-center text-gray-500 py-12 text-sm">記事がまだありません</p>
      )}
    </main>
  );
}
