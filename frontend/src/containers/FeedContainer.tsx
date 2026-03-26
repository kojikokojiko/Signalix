'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { useRecommendations, useRequestRecommendationRefresh } from '@/hooks/useRecommendations';
import { useAddBookmark, useRemoveBookmark } from '@/hooks/useBookmarks';
import { useSubmitFeedback } from '@/hooks/useFeedback';
import { ArticleCard, ArticleCardSkeleton } from '@/components/ArticleCard';
import { Button } from '@/components/ui/Button';

export function FeedContainer() {
  const { user, status } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (status === 'unauthenticated') router.push('/login');
  }, [status, router]);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
  } = useRecommendations(status === 'authenticated');

  const refreshMutation = useRequestRecommendationRefresh();
  const addBookmark = useAddBookmark();
  const removeBookmark = useRemoveBookmark();
  const feedbackMutation = useSubmitFeedback();

  const handleBookmark = (articleId: string, isBookmarked: boolean) => {
    if (isBookmarked) {
      removeBookmark.mutate(articleId);
    } else {
      addBookmark.mutate(articleId);
    }
  };

  const handleFeedback = (articleId: string, type: 'like' | 'dislike') => {
    feedbackMutation.mutate({ articleId, feedbackType: type });
  };

  if (status === 'loading' || status === 'unauthenticated') return null;

  const items = data?.pages.flatMap((p) => p.data) ?? [];

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900">For You</h1>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => refreshMutation.mutate()}
          loading={refreshMutation.isPending}
        >
          🔄 更新
        </Button>
      </div>

      {isError && (
        <div className="text-center text-gray-500 py-12">フィードの読み込みに失敗しました</div>
      )}

      {isLoading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => <ArticleCardSkeleton key={i} />)}
        </div>
      )}

      {!isLoading && items.length === 0 && (
        <div className="text-center text-gray-500 py-12">
          <p className="mb-4">まだレコメンドがありません</p>
          <p className="text-sm">興味タグを設定すると、あなた専用のフィードが表示されます</p>
        </div>
      )}

      <div className="space-y-4">
        {items.map((item) => (
          <ArticleCard
            key={item.article.id}
            article={item.article}
            recommendation={item}
            isAuthenticated={!!user}
            isBookmarkPending={addBookmark.isPending || removeBookmark.isPending}
            callbacks={{ onBookmark: handleBookmark, onFeedback: handleFeedback }}
          />
        ))}
      </div>

      {hasNextPage && (
        <div className="mt-6 text-center">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => fetchNextPage()}
            loading={isFetchingNextPage}
          >
            もっと見る
          </Button>
        </div>
      )}
    </main>
  );
}
