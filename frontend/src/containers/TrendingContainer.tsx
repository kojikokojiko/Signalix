'use client';

import { useState } from 'react';
import { useTrending, TrendingPeriod } from '@/hooks/useTrending';
import { ArticleCard, ArticleCardSkeleton } from '@/components/ArticleCard';
import { useAuth } from '@/contexts/AuthContext';
import { useAddBookmark, useRemoveBookmark } from '@/hooks/useBookmarks';

export function TrendingContainer() {
  const [period, setPeriod] = useState<TrendingPeriod>('24h');
  const { data, isLoading } = useTrending(period);
  const { user } = useAuth();
  const addBookmark = useAddBookmark();
  const removeBookmark = useRemoveBookmark();

  const handleBookmark = (articleId: string, isBookmarked: boolean) => {
    if (isBookmarked) removeBookmark.mutate(articleId);
    else addBookmark.mutate(articleId);
  };

  const articles = data?.data ?? [];

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-4">Trending</h1>

      <div className="flex gap-1 bg-gray-100 p-1 rounded-lg w-fit mb-6">
        {(['24h', '7d'] as TrendingPeriod[]).map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
              period === p ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            {p === '24h' ? '24時間' : '7日間'}
          </button>
        ))}
      </div>

      {isLoading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => <ArticleCardSkeleton key={i} />)}
        </div>
      )}

      <div className="space-y-4">
        {articles.map((article, index) => (
          <div key={article.id} className="flex gap-3">
            <span className="text-2xl font-bold text-gray-200 w-8 flex-shrink-0 pt-4">
              {index + 1}
            </span>
            <div className="flex-1">
              <ArticleCard
                article={article}
                isAuthenticated={!!user}
                isBookmarkPending={addBookmark.isPending || removeBookmark.isPending}
                callbacks={{ onBookmark: handleBookmark }}
              />
            </div>
          </div>
        ))}
      </div>

      {!isLoading && articles.length === 0 && (
        <div className="text-center text-gray-500 py-12">トレンド記事がありません</div>
      )}
    </main>
  );
}
