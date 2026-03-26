'use client';

import { useArticle } from '@/hooks/useArticle';
import { useAddBookmark } from '@/hooks/useBookmarks';
import { useSubmitFeedback } from '@/hooks/useFeedback';
import { useAuth } from '@/contexts/AuthContext';
import { formatRelativeTime } from '@/lib/utils';
import { Button } from '@/components/ui/Button';

interface Props {
  articleId: string;
}

export function ArticleDetailContainer({ articleId }: Props) {
  const { user } = useAuth();
  const { data, isLoading, isError } = useArticle(articleId);
  const addBookmark = useAddBookmark();
  const feedbackMutation = useSubmitFeedback();

  if (isLoading) {
    return (
      <main className="max-w-2xl mx-auto px-4 py-6 animate-pulse space-y-4">
        <div className="h-6 bg-gray-200 rounded w-3/4" />
        <div className="h-4 bg-gray-200 rounded w-1/2" />
        <div className="h-32 bg-gray-100 rounded" />
      </main>
    );
  }

  if (isError || !data) {
    return (
      <main className="max-w-2xl mx-auto px-4 py-12 text-center text-gray-500">
        記事が見つかりませんでした
      </main>
    );
  }

  const article = data.data;

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <div className="mb-4">
        <div className="flex items-center gap-2 text-sm text-gray-500 mb-2">
          <span>{article.source?.name}</span>
          <span>·</span>
          <span>{formatRelativeTime(article.published_at)}</span>
        </div>
        <h1 className="text-2xl font-bold text-gray-900 leading-tight">{article.title}</h1>
      </div>

      {article.summary && (
        <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 mb-6">
          <p className="text-xs font-medium text-blue-700 mb-2">✨ AI 要約</p>
          <p className="text-sm text-gray-700 leading-relaxed">{article.summary}</p>
          <p className="text-xs text-gray-400 mt-2">要約は gpt-4o-mini で生成されました</p>
        </div>
      )}

      {article.tags.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-6">
          {article.tags.map((tag) => (
            <span key={tag.id} className="text-xs bg-gray-100 text-gray-700 px-3 py-1 rounded-full">
              {tag.name}
            </span>
          ))}
        </div>
      )}

      {user && (
        <div className="flex items-center gap-3 mb-6 pb-6 border-b border-gray-200">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => addBookmark.mutate(article.id)}
            loading={addBookmark.isPending}
          >
            🔖 保存
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => feedbackMutation.mutate({ articleId: article.id, feedbackType: 'like' })}
            loading={feedbackMutation.isPending}
          >
            👍 いいね
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => feedbackMutation.mutate({ articleId: article.id, feedbackType: 'dislike' })}
            loading={feedbackMutation.isPending}
          >
            👎 非表示
          </Button>
        </div>
      )}

      <a
        href={article.url}
        target="_blank"
        rel="noopener noreferrer"
        className="block w-full text-center bg-primary-600 text-white py-3 rounded-xl font-medium hover:bg-primary-700 mb-6"
      >
        元記事を読む ↗
      </a>
    </main>
  );
}
