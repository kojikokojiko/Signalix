'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { useBookmarkList, useRemoveBookmark } from '@/hooks/useBookmarks';
import { ArticleCard, ArticleCardSkeleton } from '@/components/ArticleCard';
import { Pagination } from '@/components/ui/Pagination';

export function BookmarksContainer() {
  const { status } = useAuth();
  const router = useRouter();
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (status === 'unauthenticated') router.push('/login');
  }, [status, router]);

  const { data, isLoading } = useBookmarkList(page, status === 'authenticated');
  const removeBookmark = useRemoveBookmark();

  if (status !== 'authenticated') return null;

  const bookmarks = data?.data ?? [];
  const pagination = data?.pagination;

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900">保存した記事</h1>
        {pagination && <span className="text-sm text-gray-500">{pagination.total} 件</span>}
      </div>

      {isLoading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => <ArticleCardSkeleton key={i} />)}
        </div>
      )}

      {!isLoading && bookmarks.length === 0 && (
        <div className="text-center text-gray-500 py-12">保存した記事はありません</div>
      )}

      <div className="space-y-4">
        {bookmarks.map((bm) => (
          <div key={bm.id} className="relative">
            <ArticleCard article={bm.article} isAuthenticated showBookmark={false} />
            <button
              onClick={() => removeBookmark.mutate(bm.article_id)}
              disabled={removeBookmark.isPending}
              className="absolute top-3 right-3 text-xs text-red-400 hover:text-red-600"
            >
              削除
            </button>
          </div>
        ))}
      </div>

      {pagination && pagination.total_pages > 1 && (
        <Pagination
          page={page}
          totalPages={pagination.total_pages}
          hasNext={pagination.has_next}
          hasPrev={pagination.has_prev}
          onNext={() => setPage((p) => p + 1)}
          onPrev={() => setPage((p) => p - 1)}
        />
      )}
    </main>
  );
}
