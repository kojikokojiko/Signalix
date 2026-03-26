import Link from 'next/link';
import { formatRelativeTime } from '@/lib/utils';
import type { ArticleSummary, RecommendationItem } from '@/types/api';

// ─── Callbacks ────────────────────────────────────────────────────────────────

export interface ArticleCardCallbacks {
  onBookmark?: (articleId: string, isBookmarked: boolean) => void;
  onFeedback?: (articleId: string, type: 'like' | 'dislike') => void;
}

// ─── Props ────────────────────────────────────────────────────────────────────

export interface ArticleCardProps {
  article: ArticleSummary;
  /** If provided, shows recommendation-specific UI (reason, feedback buttons) */
  recommendation?: RecommendationItem;
  /** Whether to show the bookmark button (false on bookmarks page) */
  showBookmark?: boolean;
  /** Whether the user is authenticated (controls action visibility) */
  isAuthenticated?: boolean;
  /** Pending states from parent mutations */
  isBookmarkPending?: boolean;
  callbacks?: ArticleCardCallbacks;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function ArticleCard({
  article,
  recommendation,
  showBookmark = true,
  isAuthenticated = false,
  isBookmarkPending = false,
  callbacks,
}: ArticleCardProps) {
  const tags = article.tags.slice(0, 3);
  const isBookmarked = recommendation?.is_bookmarked ?? false;
  const currentFeedback = recommendation?.user_feedback ?? null;

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4 hover:shadow-sm transition-shadow">
      {/* Source + date */}
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-gray-500">{article.source?.name ?? '不明'}</span>
        <span className="text-xs text-gray-400">{formatRelativeTime(article.published_at)}</span>
      </div>

      {/* Title */}
      <Link href={`/articles/${article.id}`}>
        <h3 className="font-medium text-gray-900 hover:text-primary-600 line-clamp-2 mb-2">
          {article.title}
        </h3>
      </Link>

      {/* Summary */}
      {article.summary && (
        <p className="text-sm text-gray-600 line-clamp-4 mb-3">{article.summary}</p>
      )}

      {/* Tags */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-3">
          {tags.map((tag) => (
            <span key={tag.id} className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full">
              {tag.name}
            </span>
          ))}
          {article.tags.length > 3 && (
            <span className="text-xs text-gray-400">+{article.tags.length - 3}</span>
          )}
        </div>
      )}

      {/* Score bar */}
      <div className="flex items-center gap-3 mb-3">
        {article.trend_score != null && (
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-gray-400">トレンド</span>
            <div className="w-16 bg-gray-100 rounded-full h-1.5">
              <div
                className="bg-amber-400 h-1.5 rounded-full"
                style={{ width: `${Math.min(article.trend_score * 100, 100)}%` }}
              />
            </div>
            <span className="text-xs text-gray-500 tabular-nums">
              {(article.trend_score * 100).toFixed(0)}
            </span>
          </div>
        )}
        {recommendation && (
          <div className="flex items-center gap-1.5 ml-auto">
            <span className="text-xs text-gray-400">マッチ</span>
            <div className="w-16 bg-gray-100 rounded-full h-1.5">
              <div
                className="bg-primary-500 h-1.5 rounded-full"
                style={{ width: `${Math.min(recommendation.score * 100, 100)}%` }}
              />
            </div>
            <span className="text-xs text-gray-500 tabular-nums">
              {(recommendation.score * 100).toFixed(0)}
            </span>
          </div>
        )}
      </div>

      {/* Recommendation reason */}
      {recommendation?.reason && (
        <p className="text-xs text-gray-400 mb-3">💡 {recommendation.reason}</p>
      )}

      {/* Action bar */}
      <div className="flex items-center gap-3 pt-2 border-t border-gray-100">
        {isAuthenticated && recommendation && (
          <>
            <button
              aria-label="いいね"
              onClick={() => callbacks?.onFeedback?.(article.id, 'like')}
              className={`text-sm ${
                currentFeedback === 'like' ? 'text-primary-600' : 'text-gray-400 hover:text-gray-600'
              }`}
            >
              👍
            </button>
            <button
              aria-label="非表示"
              onClick={() => callbacks?.onFeedback?.(article.id, 'dislike')}
              className={`text-sm ${
                currentFeedback === 'dislike' ? 'text-red-500' : 'text-gray-400 hover:text-gray-600'
              }`}
            >
              🚫
            </button>
          </>
        )}

        {isAuthenticated && showBookmark && (
          <button
            aria-label={isBookmarked ? 'ブックマーク解除' : 'ブックマーク'}
            onClick={() => callbacks?.onBookmark?.(article.id, isBookmarked)}
            disabled={isBookmarkPending}
            className={`text-sm ml-auto ${
              isBookmarked ? 'text-primary-600' : 'text-gray-400 hover:text-gray-600'
            }`}
          >
            🔖
          </button>
        )}

        <a
          href={article.url}
          target="_blank"
          rel="noopener noreferrer"
          className={`text-xs text-gray-400 hover:text-gray-600 ${
            isAuthenticated ? '' : 'ml-auto'
          }`}
        >
          ↗
        </a>
      </div>
    </div>
  );
}

// ─── Skeleton ────────────────────────────────────────────────────────────────

export function ArticleCardSkeleton() {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4 animate-pulse">
      <div className="flex justify-between mb-2">
        <div className="h-3 bg-gray-200 rounded w-20" />
        <div className="h-3 bg-gray-200 rounded w-16" />
      </div>
      <div className="h-5 bg-gray-200 rounded w-3/4 mb-2" />
      <div className="h-4 bg-gray-200 rounded w-full mb-1" />
      <div className="h-4 bg-gray-200 rounded w-5/6 mb-3" />
      <div className="flex gap-2">
        <div className="h-5 bg-gray-100 rounded-full w-16" />
        <div className="h-5 bg-gray-100 rounded-full w-12" />
      </div>
    </div>
  );
}
